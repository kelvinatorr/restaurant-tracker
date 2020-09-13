package web

import (
	"fmt"
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

	// User Endpoints
	router.GET("/initial-signup", getInitialSignup())
	router.HEAD("/initial-signup", getInitialSignup())
	router.POST("/initial-signup", postInitialSignup(a))

	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, err interface{}) {
		log.Printf("ERROR http rest handler: %s\n", err)
		http.Error(w, "The server encountered an error processing your request.", http.StatusInternalServerError)
	}

	// TODO: Add verbose output
	// var h http.Handler
	// if verbose {
	// 	// Wrap cors handler with json logger
	// 	h = jsonLogger(c.Handler(router))
	// } else {
	// 	// Just do the cors handler
	// 	h = c.Handler(router)
	// }

	return router
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
