package web

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/kelvinatorr/restaurant-tracker/internal/mapper"
	"github.com/kelvinatorr/restaurant-tracker/internal/remover"
	"github.com/kelvinatorr/restaurant-tracker/internal/updater"
)

// Used for storing user data in the http Request Context
type contextKey int

const (
	contextKeyUser contextKey = iota
)

// Handler sets the httprouter routes for the web package
func Handler(l lister.Service, a adder.Service, u updater.Service, r remover.Service, auth auther.Service, m mapper.Service, verbose bool) http.Handler {

	router := httprouter.New()

	// Initialize a map of endpoints whose body should not be logged out because it might contain passwords
	dontLogBodyURLs := make(map[string]bool)

	initialSignUpPath := "/initial-signup"
	router.GET(initialSignUpPath, getInitialSignup(l))
	router.HEAD(initialSignUpPath, getInitialSignup(l))
	router.POST(initialSignUpPath, postUserAdd(a, "Initial Signup",
		"Create your first user by entering an email address and password below."))
	dontLogBodyURLs[initialSignUpPath] = true

	signInPath := "/sign-in"
	router.GET(signInPath, getSignIn(l))
	router.HEAD(signInPath, getSignIn(l))
	router.POST(signInPath, postSignIn(auth))
	dontLogBodyURLs[signInPath] = true

	homePath := "/"
	homeGETHandler := authRequired(getHome(l), auth)
	router.GET(homePath, homeGETHandler)
	router.HEAD(homePath, homeGETHandler)

	userAddPath := "/users-add"
	userAddGETHandler := authRequired(getUserAdd(), auth)
	userAddPOSTHandler := authRequired(postUserAdd(a, "Add A New User",
		"Add another user by adding the information below."), auth)
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

	filterPath := "/filter"
	filterGETHandler := authRequired(getFilter(l), auth)
	router.GET(filterPath, filterGETHandler)
	router.HEAD(filterPath, filterGETHandler)

	restaurantPath := "/restaurants/:id"
	restaurantGETHandler := authRequired(getRestaurant(l, m), auth)
	restaurantPOSTHandler := authRequired(postRestaurant(u, a, m), auth)
	router.GET(restaurantPath, restaurantGETHandler)
	router.HEAD(restaurantPath, restaurantGETHandler)
	router.POST(restaurantPath, restaurantPOSTHandler)

	deleteResPath := "/delete-restaurant/:id"
	deleteResGETHandler := authRequired(getDeleteRestaurant(l), auth)
	deleteResPOSTHandler := authRequired(postDeleteRestaurant(r), auth)
	router.GET(deleteResPath, deleteResGETHandler)
	router.HEAD(deleteResPath, deleteResGETHandler)
	router.POST(deleteResPath, deleteResPOSTHandler)

	mapPlaceSearchPath := "/maps/place-search"
	mapPlaceSearchGETHandler := authRequired(getPlaceSearch(m), auth)
	router.GET(mapPlaceSearchPath, mapPlaceSearchGETHandler)
	router.HEAD(mapPlaceSearchPath, mapPlaceSearchGETHandler)

	mapPlaceRefreshPath := "/maps/place-refresh/:placeID"
	mapPlaceRefreshGETHandler := authRequired(getPlaceRefresh(m), auth)
	router.GET(mapPlaceRefreshPath, mapPlaceRefreshGETHandler)
	router.HEAD(mapPlaceRefreshPath, mapPlaceRefreshGETHandler)

	mapPlacePath := "/maps/place/:id"
	mapPlaceDELETEHandler := authRequired(deletePlace(r), auth)
	router.DELETE(mapPlacePath, mapPlaceDELETEHandler)

	visitsPath := "/r/:resid/visits"
	visitsGETHandler := authRequired(getVisits(l), auth)
	router.GET(visitsPath, visitsGETHandler)
	router.HEAD(visitsPath, visitsGETHandler)

	// Serve files from the web/static directory
	router.ServeFiles("/static/*filepath", fileSystem{http.Dir("../../web/static")})

	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, err interface{}) {
		log.Printf("ERROR http rest handler: %s\n", err)
		http.Error(w, "The server encountered an error processing your request.", http.StatusInternalServerError)
	}

	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := newView("base", "../../web/template/404.html")
		data := Data{}
		v.render(w, data)
		return
	})

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
		v := newView("base", "../../web/template/create-user.html")
		data := Data{}
		data.Head = Head{"Initial Signup"}
		data.Yield = struct {
			Heading   string
			Text      string
			FirstName string
			LastName  string
			Email     string
		}{
			"Initial Signup",
			"Create your first user by entering an email address and password below.",
			"",
			"",
			"",
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
		data.Head = Head{Title: "Sign In"}
		v.render(w, data)
	}
}

func postUserAdd(a adder.Service, heading string, text string) func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var u adder.User

		data := Data{}
		data.Head = Head{Title: heading}
		v := newView("base", "../../web/template/create-user.html")

		if err := parseForm(r, &u); err != nil {
			log.Println(err)
			http.Error(w, AlertFormParseErrorGeneric, http.StatusInternalServerError)
			return
		}
		newUserID, err := a.AddUser(u)
		if err != nil {
			log.Println(err)
			data.Alert = Alert{Message: err.Error()}
			// Add the data that was submitted for convenience
			data.Yield = struct {
				Heading   string
				Text      string
				FirstName string
				LastName  string
				Email     string
			}{
				heading,
				text,
				u.FirstName,
				u.LastName,
				u.Email,
			}
			v.render(w, data)
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
		data.Head = Head{Title: "Sign In"}
		v := newView("base", "../../web/template/sign-in.html")

		if err := parseForm(r, &u); err != nil {
			log.Println(err)
			http.Error(w, AlertFormParseErrorGeneric, http.StatusInternalServerError)
			return
		}
		jwt, err := a.SignIn(u)
		if err != nil {
			log.Println(err)
			data.Alert = Alert{Message: err.Error()}
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

func getHome(s lister.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		v := newView("base", "../../web/template/index.html")
		// TODO: Pull in Site Name from the database.

		// get the query parameters parameter
		queryParams := r.URL.Query()

		data := Data{}
		data.Head = Head{"Our Restaurant Tracker"}
		// Get all restaurants
		restaurants, err := s.GetRestaurants(queryParams)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "There was a problem processing your request", http.StatusBadRequest)
			return
		}
		data.Yield = struct {
			Restaurants []lister.Restaurant
		}{
			restaurants,
		}
		v.render(w, data)
	}
}

func getUserAdd() func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		v := newView("base", "../../web/template/create-user.html")
		data := Data{}
		data.Head = Head{"Add A New User"}
		data.Yield = struct {
			Heading   string
			Text      string
			FirstName string
			LastName  string
			Email     string
		}{
			"Add A New User",
			"Add another user by adding the information below.",
			"",
			"",
			"",
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

		v := newView("base", "../../web/template/user.html")

		data := Data{}
		data.Head = Head{fmt.Sprintf("Profile: %s %s", user.FirstName, user.LastName)}
		data.Yield = struct {
			Heading string
			Text    string
			User    lister.User
		}{
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
			http.Error(w, AlertErrorMsgGeneric, http.StatusInternalServerError)
			return
		}

		var userUpdate updater.User
		if err := parseForm(r, &userUpdate); err != nil {
			log.Println(err)
			http.Error(w, AlertFormParseErrorGeneric, http.StatusInternalServerError)
			return
		}
		userUpdate.ID = user.ID
		recordsAffected, err := u.UpdateUser(userUpdate)
		if err != nil {
			log.Println(err)
			v := newView("base", "../../web/template/user.html")
			data := Data{}
			data.Head = Head{fmt.Sprintf("Profile: %s %s", user.FirstName, user.LastName)}
			// Show the user the error.
			data.Alert = Alert{err.Error()}
			// Fill in the form again for convenience
			data.Yield = struct {
				Heading string
				Text    string
				User    updater.User
			}{
				fmt.Sprintf("Profile: %s %s", user.FirstName, user.LastName),
				"Edit your profile by changing the information below.",
				userUpdate,
			}
			v.render(w, data)
			return
		}
		log.Printf("Updated user with ID: %d. %d records affected\n", user.ID, recordsAffected)
		// Redirect to the same page which will show the changed values.
		http.Redirect(w, r, fmt.Sprintf("/users/%d", userUpdate.ID), http.StatusFound)
	}

}

func getChangePassword() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		v := newView("base", "../../web/template/change-password.html")

		data := Data{}
		data.Head = Head{"Change Password"}
		data.Yield = struct {
			Heading string
			Text    string
		}{
			"ChangePassword",
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
			http.Error(w, AlertErrorMsgGeneric, http.StatusInternalServerError)
			return
		}

		var uCP auther.UserChangePassword
		if err := parseForm(r, &uCP); err != nil {
			log.Println(err)
			http.Error(w, AlertFormParseErrorGeneric, http.StatusBadRequest)
			return
		}

		uCP.ID = user.ID

		recordsAffected, err := u.UpdateUserPassword(uCP)

		v := newView("base", "../../web/template/change-password.html")

		data := Data{}
		data.Head = Head{"Change Password"}
		data.Yield = struct {
			Heading string
			Text    string
		}{
			"ChangePassword",
			"Change your password by entering your current password and new password below.",
		}
		if err != nil {
			log.Println(err)

			// Show the user the error.
			data.Alert = Alert{err.Error()}
			v.render(w, data)
			return
		}
		log.Printf("Updated password for user with ID: %d. %d records affected\n", user.ID, recordsAffected)

		// Display success alert
		data.Alert = Alert{"Success! Your password has been changed."}
		v.render(w, data)
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

func getFilter(s lister.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		// Read the query params to fill up the form
		queryParams := r.URL.Query()

		v := newView("base", "../../web/template/filter.html")

		data := Data{}
		data.Head = Head{"Filter Restaurants"}
		// Get all select filters
		filterOptions := s.GetFilterOptions(queryParams)

		lastVisitOp := s.GetFilterParam("last_visit", queryParams).Operator

		avgRatingFilterOp := s.GetFilterParam("avg_rating", queryParams)

		avgRating := struct {
			Operator string
			Value    string
		}{Operator: avgRatingFilterOp.Operator, Value: avgRatingFilterOp.Value}

		data.Yield = struct {
			Heading       string
			Text          string
			FilterOptions lister.FilterOptions
			LastVisitOp   string
			AvgRating     struct {
				Operator string
				Value    string
			}
		}{
			"Filter Restaurants",
			"Filter the restaurant table by selecting options below.",
			filterOptions,
			lastVisitOp,
			avgRating,
		}
		v.render(w, data)
	}
}

func getRestaurant(s lister.Service, m mapper.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		// get the route parameter
		ID, err := strconv.Atoi(p.ByName("id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid restaurant ID, it must be a number.", p.ByName("id")),
				http.StatusBadRequest)
			return
		}

		v := newView("base", "../../web/template/restaurant.html")

		data := Data{}

		haveGmapsKey := m.HaveGmapsKey()

		var restaurant lister.Restaurant
		// Get the restaurant requested
		if ID != 0 {
			restaurant, err = s.GetRestaurant(int64(ID))
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			data.Head = Head{restaurant.Name}
			data.Yield = struct {
				Heading      string
				Text         string
				Restaurant   lister.Restaurant
				HaveGmapsKey bool
			}{
				restaurant.Name,
				"Edit this restaurant's details below",
				restaurant,
				haveGmapsKey,
			}
		} else {
			// Adding a new restaurant
			restaurant = lister.Restaurant{}
			data.Head = Head{"Add A New Restaurant"}
			data.Yield = struct {
				Heading      string
				Text         string
				Restaurant   lister.Restaurant
				HaveGmapsKey bool
			}{
				"Add A New Restaurant",
				"Add the new restaurant's details below",
				restaurant,
				haveGmapsKey,
			}
		}

		v.render(w, data)
	}
}

func getPlaceSearch(m mapper.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		if !m.HaveGmapsKey() {
			http.Error(w, "No Google Maps Key", http.StatusPaymentRequired)
			return
		}

		queryParams := r.URL.Query()
		searchTerm := queryParams.Get("searchTerm")
		if searchTerm == "" {
			http.Error(w, "A ?searchTerm query parameter is required", http.StatusBadRequest)
			return
		}

		candidates, err := m.PlaceSearch(searchTerm)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(candidates)
	}
}

func postRestaurant(u updater.Service, a adder.Service, m mapper.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ID, err := strconv.Atoi(p.ByName("id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid restaurant ID, it must be a number.", p.ByName("id")),
				http.StatusBadRequest)
			return
		}
		if ID != 0 {
			updateRestaurant(u, m, w, r)
		} else {
			addRestaurant(a, m, w, r)
		}
	}
}

func addRestaurant(a adder.Service, m mapper.Service, w http.ResponseWriter, r *http.Request) {
	var resNew adder.Restaurant
	if err := parseForm(r, &resNew); err != nil {
		log.Println(err)
		http.Error(w, AlertFormParseErrorGeneric, http.StatusInternalServerError)
		return
	}

	newRestaurantID, err := a.AddRestaurant(resNew)
	if err != nil {
		log.Println(err)
		v := newView("base", "../../web/template/restaurant.html")
		data := Data{}
		data.Head = Head{"Add A New Restaurant"}
		// Show the user the error.
		data.Alert = Alert{err.Error()}

		// Fill in the form again for convenience. Need lister.Restaurant because we need an ID property for the template
		restaurant := lister.Restaurant{
			Name:    resNew.Name,
			Cuisine: resNew.Cuisine,
			Note:    resNew.Note,
			CityState: lister.CityState{
				Name:  resNew.CityState.Name,
				State: resNew.CityState.State,
			},
		}

		data.Yield = struct {
			Heading      string
			Text         string
			Restaurant   lister.Restaurant
			HaveGmapsKey bool
		}{
			"Add A New Restaurant",
			"Add the new restaurant's details below",
			restaurant,
			m.HaveGmapsKey(),
		}
		v.render(w, data)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/restaurants/%d", newRestaurantID), http.StatusFound)
}

func updateRestaurant(u updater.Service, m mapper.Service, w http.ResponseWriter, r *http.Request) {
	var resUpdate updater.Restaurant
	if err := parseForm(r, &resUpdate); err != nil {
		log.Println(err)
		http.Error(w, AlertFormParseErrorGeneric, http.StatusInternalServerError)
		return
	}

	recordsAffected, err := u.UpdateRestaurant(resUpdate)
	if err != nil {
		log.Println(err)
		v := newView("base", "../../web/template/restaurant.html")
		data := Data{}
		data.Head = Head{resUpdate.Name}
		// Show the user the error.
		data.Alert = Alert{err.Error()}
		// Fill in the form again for convenience
		data.Yield = struct {
			Heading      string
			Text         string
			Restaurant   updater.Restaurant
			HaveGmapsKey bool
		}{
			resUpdate.Name,
			"Edit this restuarant's details below",
			resUpdate,
			m.HaveGmapsKey(),
		}
		v.render(w, data)
		return
	}
	log.Printf("Updated restaurant with ID: %d. %d records affected\n", resUpdate.ID, recordsAffected)
	// Redirect to the same page which will show the changed values.
	http.Redirect(w, r, fmt.Sprintf("/restaurants/%d", resUpdate.ID), http.StatusFound)
}

func deletePlace(s remover.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		// get the route parameter
		ID, err := strconv.Atoi(p.ByName("id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid Gmaps Place ID, it must be a number.", p.ByName("id")),
				http.StatusBadRequest)
			return
		}

		log.Printf("Removing Gmaps Place ID: %d\n", ID)
		recordsAffected := s.RemoveGmapsPlace(int64(ID))
		log.Printf("Number of records affected %d", recordsAffected)

		rm := struct {
			Message string
		}{
			Message: fmt.Sprintf("Gmaps Place ID: %d removed", ID),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rm)
	}
}

func getPlaceRefresh(m mapper.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if !m.HaveGmapsKey() {
			http.Error(w, "No Google Maps Key", http.StatusPaymentRequired)
			return
		}

		placeID := p.ByName("placeID")

		placeDetails, err := m.PlaceDetails(placeID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		pd := struct {
			PlaceID              string  `json:"placeID"`
			BusinessStatus       string  `json:"businessStatus"`
			FormattedPhoneNumber string  `json:"formattedPhoneNumber"`
			Name                 string  `json:"name"`
			PriceLevel           int     `json:"priceLevel"`
			Rating               float32 `json:"rating"`
			URL                  string  `json:"url"`
			UserRatingsTotal     int     `json:"userRatingsTotal"`
			UTCOffset            int     `json:"utcOffset"`
			Website              string  `json:"website"`
			Address              string  `json:"address"`
			ZipCode              string  `json:"zipCode"`
		}{
			PlaceID:              placeDetails.Result.PlaceID,
			BusinessStatus:       placeDetails.Result.BusinessStatus,
			FormattedPhoneNumber: placeDetails.Result.FormattedPhoneNumber,
			Name:                 placeDetails.Result.Name,
			PriceLevel:           placeDetails.Result.PriceLevel,
			Rating:               placeDetails.Result.Rating,
			URL:                  placeDetails.Result.URL,
			UserRatingsTotal:     placeDetails.Result.UserRatingsTotal,
			UTCOffset:            placeDetails.Result.UTCOffset,
			Website:              placeDetails.Result.Website,
			Address:              placeDetails.Result.Address,
			ZipCode:              placeDetails.Result.ZipCode,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(pd)
	}
}

func getDeleteRestaurant(l lister.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ID, err := strconv.Atoi(p.ByName("id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid restaurant ID, it must be a number.", p.ByName("id")),
				http.StatusBadRequest)
			return
		}

		v := newView("base", "../../web/template/delete-restaurant.html")

		data := Data{}

		var restaurant lister.Restaurant
		// Get the restaurant requested
		restaurant, err = l.GetRestaurant(int64(ID))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		data.Head = Head{restaurant.Name}
		data.Yield = struct {
			Heading    string
			Text       string
			Restaurant lister.Restaurant
		}{
			fmt.Sprintf("Delete %s", restaurant.Name),
			fmt.Sprintf("Are you sure you want to delete %s? This will also delete all visit data for this restaurant.", restaurant.Name),
			restaurant,
		}

		v.render(w, data)
	}
}

func postDeleteRestaurant(s remover.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ID, err := strconv.Atoi(p.ByName("id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid restaurant ID, it must be a number.", p.ByName("id")),
				http.StatusBadRequest)
			return
		}

		deleteConfirm := struct {
			Name        string `schema:"name"`
			ConfirmName string `schema:"confirmName"`
		}{
			"",
			"",
		}
		if err := parseForm(r, &deleteConfirm); err != nil {
			log.Println(err)
			http.Error(w, AlertFormParseErrorGeneric, http.StatusInternalServerError)
			return
		}

		if deleteConfirm.Name != deleteConfirm.ConfirmName {
			log.Printf("Delete requested for %d, but confirmation name %s doesn't match %s", ID, deleteConfirm.ConfirmName,
				deleteConfirm.Name)
			v := newView("base", "../../web/template/delete-restaurant.html")
			data := Data{}
			data.Head = Head{deleteConfirm.Name}
			// Show the user the error.
			data.Alert = Alert{fmt.Sprintf("Input: %s did not match %s", deleteConfirm.ConfirmName, deleteConfirm.Name)}
			// Fill in the form again for convenience
			data.Yield = struct {
				Heading    string
				Text       string
				Restaurant lister.Restaurant
			}{
				fmt.Sprintf("Delete %s", deleteConfirm.Name),
				fmt.Sprintf("Are you sure you want to delete %s?", deleteConfirm.Name),
				lister.Restaurant{Name: deleteConfirm.Name},
			}
			v.render(w, data)
			return
		} else {
			log.Printf("Confirmed request to remove %s with ID: %d", deleteConfirm.Name, ID)
			s.RemoveRestaurant(remover.Restaurant{ID: int64(ID)})
			// Redirect to the list of other restaurants.
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	}
}

func getVisits(l lister.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ID, err := strconv.Atoi(p.ByName("resid"))
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid restaurant ID, it must be a number.", p.ByName("resid")),
				http.StatusBadRequest)
			return
		}
		resID := int64(ID)

		// Get the restaurant 1st so we can show its name and make sure it exists
		restaurant, err := l.GetRestaurant(resID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// Then we get its visits
		visits := l.GetVisitsByRestaurantID(resID)

		v := newView("base", "../../web/template/visits.html")

		data := Data{}

		data.Head = Head{restaurant.Name}
		data.Yield = struct {
			Heading      string
			Text         string
			RestaurantID int64
			Visits       []lister.Visit
		}{
			fmt.Sprintf("%s Visits", restaurant.Name),
			fmt.Sprintf("Are you sure you want to delete %s?", restaurant.Name),
			restaurant.ID,
			visits,
		}

		v.render(w, data)
	}
}

