package web

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/schema"
	"github.com/julienschmidt/httprouter"
	"github.com/kelvinatorr/restaurant-tracker/internal/adder"
	"github.com/kelvinatorr/restaurant-tracker/internal/auther"
	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
	"github.com/kelvinatorr/restaurant-tracker/internal/remover"
	"github.com/kelvinatorr/restaurant-tracker/internal/updater"
)

// Handler sets the httprouter routes for the web package
func Handler(l lister.Service, a adder.Service, u updater.Service, r remover.Service, auth auther.Service, verbose bool) http.Handler {

	router := httprouter.New()

	// Initialize a map of endpoints whose body should not be logged out because it might contain passwords
	dontLogBodyURLs := make(map[string]bool)

	authRequiredURLs := make(map[string]bool)

	initialSignUpPath := "/initial-signup"
	router.GET(initialSignUpPath, getInitialSignup(l))
	router.HEAD(initialSignUpPath, getInitialSignup(l))
	router.POST(initialSignUpPath, postUserAdd(a))
	dontLogBodyURLs[initialSignUpPath] = true

	signinPath := "/signin"
	router.GET(signinPath, getSignIn(l))
	router.HEAD(signinPath, getSignIn(l))
	router.POST(signinPath, postSignIn(auth))
	dontLogBodyURLs[signinPath] = true

	router.GET("/", getHome())
	router.HEAD("/", getHome())
	authRequiredURLs["/"] = true

	userAddPath := "/users-add"
	router.GET(userAddPath, getUserAdd())
	router.HEAD(userAddPath, getUserAdd())
	router.POST(userAddPath, postUserAdd(a))
	dontLogBodyURLs[userAddPath] = true
	authRequiredURLs[userAddPath] = true

	userPath := "/users/:id"
	router.GET(userPath, handleUser(l, u, auth))
	router.HEAD(userPath, handleUser(l, u, auth))
	router.POST(userPath, handleUser(l, u, auth))
	authRequiredURLs[userPath] = true

	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, err interface{}) {
		log.Printf("ERROR http rest handler: %s\n", err)
		http.Error(w, "The server encountered an error processing your request.", http.StatusInternalServerError)
	}

	// Add verbose output
	var h http.Handler
	h = authRequired(router, auth, authRequiredURLs)
	if verbose {
		// Wrap handler with verbose logger
		h = verboseLogger(h, dontLogBodyURLs)
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

func authRequired(handler http.Handler, a auther.Service, dontLogBodyURLs map[string]bool) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, check := dontLogBodyURLs[r.URL.String()]; check {
			log.Println("Checking rt cookie")
			rememberTokenCookie, err := r.Cookie("rt")
			if err != nil {
				log.Println(err.Error())
				// Redirect to sign in page
				http.Redirect(w, r, "/signin", http.StatusFound)
				return
			}
			// Check cookie and redirect to sign in page if err
			err = a.CheckJWT(rememberTokenCookie.Value)
			if err != nil {
				log.Println(err.Error())
				// Redirect to sign in page
				http.Redirect(w, r, "/signin", http.StatusFound)
				return
			}
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

func getInitialSignup(l lister.Service) func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		// If there already are users in the database then send them to the home page
		userCount := l.GetUserCount()
		if userCount > 0 {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		v := newView("base", "../../web/template/create-user.html")
		data := struct {
			Title  string
			Header string
			Text   string
		}{
			"Initial Signup",
			"Initial Signup",
			"Create your first user by entering an email address and password below.",
		}
		v.render(w, data)
	}
}

func getSignIn(l lister.Service) func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		// If there are no users in the database then send them to the initial signup page
		userCount := l.GetUserCount()
		if userCount == 0 {
			http.Redirect(w, r, "/initial-signup", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		v := newView("base", "../../web/template/signin.html")
		data := struct {
			Title string
		}{"Sign In"}
		v.render(w, data)
	}
}

func postUserAdd(a adder.Service) func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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
		log.Printf("New user created with ID: %d\n", newUserID)
		// Redirect to homepage
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func postSignIn(a auther.Service) func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var u auther.UserSignIn
		if err := parseForm(r, &u); err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		jwt, err := a.SignIn(u)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Given them a cookie.
		cookie := http.Cookie{
			Name:     "rt",
			Value:    jwt,
			HttpOnly: true,
			MaxAge:   31536000, // One Year
			SameSite: http.SameSiteLaxMode,
		}

		http.SetCookie(w, &cookie)

		// Redirect to Home Page
		// TODO: Redirect to the protected route they tried to access if any
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func getHome() func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "text/html")
		v := newView("base", "../../web/template/index.html")
		// TODO: Pull in Site Name from the database.
		data := struct {
			Title string
		}{"Our Restaurant Tracker"}
		v.render(w, data)
	}
}

func getUserAdd() func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "text/html")
		v := newView("base", "../../web/template/create-user.html")
		data := struct {
			Title  string
			Header string
			Text   string
		}{
			"Add A New User",
			"Add A New User",
			"Add another user by adding the information below.",
		}
		v.render(w, data)
	}
}

func handleUser(l lister.Service, u updater.Service, auth auther.Service) func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		// get the route parameter
		ID, err := strconv.Atoi(p.ByName("id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid user ID, it must be a number.", p.ByName("id")),
				http.StatusBadRequest)
			return
		}

		user := l.GetUserByID(int64(ID))
		// Check that the user exists.
		if user.ID == 0 {
			http.Error(w, fmt.Sprintf("There is no user with id %s", p.ByName("id")), http.StatusBadRequest)
			return
		}

		// Check that the signed in in user is not editing someone else.
		// First get the cookie
		rememberTokenCookie, err := r.Cookie("rt")
		if err != nil {
			log.Println(err.Error())
			// Redirect to sign in page
			http.Redirect(w, r, "/signin", http.StatusFound)
			return
		}
		// Decode the payload (we already know it is valid because it was checked by the auth middleware)
		signedInUser, err := auth.GetCookiePayload(rememberTokenCookie.Value)
		if err != nil {
			log.Println(err.Error())
			http.Redirect(w, r, "/signin", http.StatusFound)
			return
		}
		// Then compare that the email addresses are the same
		if user.Email != signedInUser.Email {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		method := r.Method
		if method == "GET" || method == "HEAD" {
			getUser(w, r, user)
		} else if method == "POST" {
			putUser(w, r, user, u)
		}
	}
}

func getUser(w http.ResponseWriter, r *http.Request, user lister.User) {
	w.Header().Set("Content-Type", "text/html")
	v := newView("base", "../../web/template/user.html")
	data := struct {
		Title  string
		Header string
		Text   string
		User   lister.User
	}{
		fmt.Sprintf("Profile: %s %s", user.FirstName, user.LastName),
		fmt.Sprintf("Profile: %s %s", user.FirstName, user.LastName),
		"Edit your profile by changing the information below.",
		user,
	}
	v.render(w, data)
}

func putUser(w http.ResponseWriter, r *http.Request, user lister.User, u updater.Service) {
	var userUpdate updater.User
	if err := parseForm(r, &userUpdate); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	userUpdate.ID = user.ID
	recordsAffected, err := u.UpdateUser(userUpdate)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("Updated user with ID: %d. %d records affected\n", user.ID, recordsAffected)
	// Redirect to homepage
	http.Redirect(w, r, fmt.Sprintf("/users/%d", userUpdate.ID), http.StatusFound)

}
