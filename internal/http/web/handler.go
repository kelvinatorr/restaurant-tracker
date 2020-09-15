package web

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/schema"
	"github.com/julienschmidt/httprouter"
	"github.com/kelvinatorr/restaurant-tracker/internal/adder"
	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
	"github.com/kelvinatorr/restaurant-tracker/internal/remover"
	"github.com/kelvinatorr/restaurant-tracker/internal/updater"
)

// Handler sets the httprouter routes for the web package
func Handler(l lister.Service, a adder.Service, u updater.Service, r remover.Service, verbose bool) http.Handler {
	router := httprouter.New()

	// Initialize a map of endpoints whose body should not be logged out because it might contain passwords
	dontLogBodyURLs := make(map[string]bool)
	// User Endpoints
	router.GET("/initial-signup", getInitialSignup())
	router.HEAD("/initial-signup", getInitialSignup())
	router.POST("/initial-signup", postInitialSignup(a))
	dontLogBodyURLs["/initial-signup"] = true

	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, err interface{}) {
		log.Printf("ERROR http rest handler: %s\n", err)
		http.Error(w, "The server encountered an error processing your request.", http.StatusInternalServerError)
	}

	// Add verbose output
	var h http.Handler
	if verbose {
		// Wrap handler with verbose logger
		h = verboseLogger(router, dontLogBodyURLs)
	} else {
		// Just do the handler
		h = router
	}

	return h
}

func verboseLogger(handler http.Handler, dontLogBodyURLs map[string]bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received %s with URL: %s\n", r.Method, r.URL)
		if _, check := dontLogBodyURLs[r.URL.String()]; (r.Method == "PUT" || r.Method == "POST") && !check {
			log.Printf("With body:")
			var body []byte
			buf := make([]byte, 1024)
			for {
				bytesRead, err := r.Body.Read(buf)

				body = append(body, buf[:bytesRead]...)
				if err != nil && err != io.EOF {
					log.Println(err)
					break
				}

				log.Printf(string(buf[:bytesRead]))

				if err == io.EOF {
					break
				}
			}
			// Set a new body, which will simulate the same data we read:
			r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		}

		// Call next handler
		handler.ServeHTTP(w, r)
	})
}

func parseForm(r *http.Request, dest interface{}) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	dec := schema.NewDecoder()
	if err := dec.Decode(dest, r.PostForm); err != nil {
		return err
	}
	return nil
}

func getInitialSignup() func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "text/html")
		v := newView("base", "../../web/template/initial-signup.html")
		data := struct {
			Title string
		}{"Initial Signup"}
		v.render(w, data)
	}
}

func postInitialSignup(a adder.Service) func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var u adder.User
		if err := parseForm(r, &u); err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		newUserID, err := a.AddUser(u)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Println(newUserID)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintln(w, "Works post!")
	}
}
