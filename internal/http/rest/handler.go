package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
)

// Handler sets the httprouter routes for the rest package
func Handler(l lister.Service) http.Handler {
	router := httprouter.New()

	router.GET("/restaurants", getRestaurants(l))
	router.GET("/restaurants/:id", getRestaurant(l))

	return router
}

// getRestaurants returns a handler for GET /restaurants requests
func getRestaurants(s lister.Service) func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")
		list := s.GetRestaurants()
		json.NewEncoder(w).Encode(list)
	}
}

func getRestaurant(s lister.Service) func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		// get the route parameter
		ID, err := strconv.Atoi(p.ByName("id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid beer ID, it must be a number.", p.ByName("id")),
				http.StatusBadRequest)
			return
		}

		restaurant, err := s.GetRestaurant(int64(ID))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(restaurant)
		}
	}
}
