package web

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gorilla/schema"
	"github.com/julienschmidt/httprouter"
	"github.com/kelvinatorr/restaurant-tracker/internal/adder"
	"github.com/kelvinatorr/restaurant-tracker/internal/auther"
	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
	"github.com/kelvinatorr/restaurant-tracker/internal/remover"
	"github.com/kelvinatorr/restaurant-tracker/internal/updater"
)

// Used for storing user data in the http Request Context
type contextKey int

const (
	contextKeyUser contextKey = iota
)

// Handler sets the httprouter routes for the web package
func Handler(l lister.Service, a adder.Service, u updater.Service, r remover.Service, auth auther.Service, verbose bool) http.Handler {

	router := httprouter.New()

	// Initialize a map of endpoints whose body should not be logged out because it might contain passwords
	dontLogBodyURLs := make(map[string]bool)

	initialSignUpPath := "/initial-signup"
	router.GET(initialSignUpPath, getInitialSignup(l))
	router.HEAD(initialSignUpPath, getInitialSignup(l))
	router.POST(initialSignUpPath, postUserAdd(a))
	dontLogBodyURLs[initialSignUpPath] = true

	signInPath := "/sign-in"
	router.GET(signInPath, getSignIn(l))
	router.HEAD(signInPath, getSignIn(l))
	router.POST(signInPath, postSignIn(auth))
	dontLogBodyURLs[signInPath] = true

	homePath := "/"
	homeGETHandler := authRequired(getHome(), auth)
	router.GET(homePath, homeGETHandler)
	router.HEAD(homePath, homeGETHandler)

	userAddPath := "/users-add"
	userAddGETHandler := authRequired(getUserAdd(), auth)
	userAddPOSTHandler := authRequired(postUserAdd(a), auth)
	router.GET(userAddPath, userAddGETHandler)
	router.HEAD(userAddPath, userAddGETHandler)
	router.POST(userAddPath, userAddPOSTHandler)
	dontLogBodyURLs[userAddPath] = true

	userPath := "/users/:id"
	userGETHandler := authRequired(checkUser(getUser(), l, u, auth), auth)
	userPOSTHandler := authRequired(checkUser(postUser(u), l, u, auth), auth)
	router.GET(userPath, userGETHandler)
	router.HEAD(userPath, userGETHandler)
	router.POST(userPath, userPOSTHandler)

	changePasswordPath := "/users/:id/change-password"
	changePasswordGETHandler := authRequired(checkUser(getChangePassword(), l, u, auth), auth)
	changePasswordPOSTHandler := authRequired(checkUser(postChangePassword(u), l, u, auth), auth)
	router.GET(changePasswordPath, changePasswordGETHandler)
	router.HEAD(changePasswordPath, changePasswordGETHandler)
	router.POST(changePasswordPath, changePasswordPOSTHandler)
	dontLogBodyURLs[changePasswordPath] = true

	signOutPath := "/sign-out"
	signOutPOSTHandler := postSignOut()
	router.POST(signOutPath, signOutPOSTHandler)

	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, err interface{}) {
		log.Printf("ERROR http rest handler: %s\n", err)
		http.Error(w, "The server encountered an error processing your request.", http.StatusInternalServerError)
	}

	// Add verbose output
	var h http.Handler
	// h = authRequired(router, auth, authRequiredURLs)
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
		if r.Method == "PUT" || r.Method == "POST" {
			urlPath := r.URL.String()
			if _, check := dontLogBodyURLs[urlPath]; !check {
				// Don't log out the change-password path.
				// TODO: Use dontLogBodyURLs and loop regex instead of a map, but this is good enough for now.
				match, err := regexp.MatchString("/users/\\d/change-password", urlPath)
				if !match {
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
				} else if err != nil {
					log.Println(err)
				}
			}
		}

		// Call next handler
		handler.ServeHTTP(w, r)
	})
}

func authRequired(handler httprouter.Handle, auth auther.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		log.Println("Checking rt cookie")
		rememberTokenCookie, err := r.Cookie("rt")
		if err != nil {
			log.Println(err.Error())
			// Redirect to sign in page
			http.Redirect(w, r, "/sign-in", http.StatusFound)
			return
		}
		// Check cookie and redirect to sign in page if err
		err = auth.CheckJWT(rememberTokenCookie.Value)
		if err != nil {
			log.Println(err.Error())
			// Redirect to sign in page
			http.Redirect(w, r, "/sign-in", http.StatusFound)
			return
		}
		// Call the next httprouter.Handle
		handler(w, r, p)
	}
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
		v := newView("base", "../../web/template/sign-in.html")
		data := Data{}
		data.Header = Header{Title: "Sign In"}
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
		data := Data{}
		data.Header = Header{Title: "Sign In"}

		if err := parseForm(r, &u); err != nil {
			log.Println(err)
			data.Alert = &Alert{Message: AlertErrorMsgGeneric}
			return
		}
		jwt, err := a.SignIn(u)
		if err != nil {
			log.Println(err)
			v := newView("base", "../../web/template/sign-in.html")
			data.Alert = &Alert{Message: err.Error()}
			// Add the email that was submitted for convenience
			data.Yield = struct{ Email string }{u.Email}
			v.render(w, data)
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

func getHome() httprouter.Handle {
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

func checkUser(handler httprouter.Handle, l lister.Service, u updater.Service, auth auther.Service) httprouter.Handle {
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
			http.Redirect(w, r, "/sign-in", http.StatusFound)
			return
		}
		// Decode the payload (we already know it is valid because it was checked by the auth middleware)
		signedInUser, err := auth.GetCookiePayload(rememberTokenCookie.Value)
		if err != nil {
			log.Println(err.Error())
			http.Redirect(w, r, "/sign-in", http.StatusFound)
			return
		}
		// Then compare that the ids are the same
		if user.ID != signedInUser.ID {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Save the user to the context
		ctx := r.Context()

		ctx = context.WithValue(ctx, contextKeyUser, user)
		// Get new http.Request with the new context
		r = r.WithContext(ctx)

		// Call the next httprouter.Handle
		handler(w, r, p)
	}
}

func getUser() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		// Get the user from the context
		user, ok := r.Context().Value(contextKeyUser).(lister.User)
		if !ok {
			log.Println("user is not type lister.User")
			http.Error(w, "A server error occurred", http.StatusInternalServerError)
			return
		}

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
}

func postUser(u updater.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		// Get the user from the context
		user, ok := r.Context().Value(contextKeyUser).(lister.User)
		if !ok {
			log.Println("user is not type lister.User")
			http.Error(w, "A server error occurred", http.StatusInternalServerError)
			return
		}

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
		// Redirect to the same page
		http.Redirect(w, r, fmt.Sprintf("/users/%d", userUpdate.ID), http.StatusFound)
	}

}

func getChangePassword() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		w.Header().Set("Content-Type", "text/html")
		v := newView("base", "../../web/template/change-password.html")
		data := struct {
			Title  string
			Header string
			Text   string
		}{
			"Change Password",
			"Change Password",
			"Change your password by entering your current password and new password below.",
		}
		v.render(w, data)
	}
}

func postChangePassword(u updater.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		user, ok := r.Context().Value(contextKeyUser).(lister.User)
		if !ok {
			log.Println("user is not type lister.User")
			http.Error(w, "A server error occurred", http.StatusInternalServerError)
			return
		}

		var uCP auther.UserChangePassword
		if err := parseForm(r, &uCP); err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		uCP.ID = user.ID

		recordsAffected, err := u.UpdateUserPassword(uCP)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Printf("Updated password for user with ID: %d. %d records affected\n", user.ID, recordsAffected)
		// Redirect to the same page
		http.Redirect(w, r, fmt.Sprintf("/users/%d/change-password", uCP.ID), http.StatusFound)
	}
}

func postSignOut() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		// Remove their cookie value
		cookie := http.Cookie{
			Name:     "rt",
			Value:    "",
			HttpOnly: true,
			MaxAge:   -1, // Expire immediately
			SameSite: http.SameSiteLaxMode,
		}

		http.SetCookie(w, &cookie)
		http.Redirect(w, r, "/sign-in", http.StatusFound)
	}
}
