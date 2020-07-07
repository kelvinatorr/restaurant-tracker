package rest

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
)

// Handler sets the httprouter routes for the rest package
func Handler(l lister.Service) http.Handler {
	router := httprouter.New()

	router.GET("/restaurants", getRestaurants(l))

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
