package rest

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/kelvinatorr/restaurant-tracker/internal/updater"

	"github.com/julienschmidt/httprouter"
	"github.com/kelvinatorr/restaurant-tracker/internal/adder"
	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
	"github.com/rs/cors"
)

type responseMessage struct {
	Message string
}

// Handler sets the httprouter routes for the rest package
func Handler(l lister.Service, a adder.Service, u updater.Service) http.Handler {
	router := httprouter.New()

	router.GET("/restaurants", getRestaurants(l))
	router.GET("/restaurants/:id", getRestaurant(l))

	router.POST("/restaurants", addRestaurant(a))

	router.PUT("/restaurants", updateRestaurant(u))

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"https://postwoman.io"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		// Enable Debugging for testing, consider disabling in production
		Debug: false,
	})

	corsRouter := c.Handler(router)

	return corsRouter
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
			http.Error(w, fmt.Sprintf("%s is not a valid restaurant ID, it must be a number.", p.ByName("id")),
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

func addRestaurant(s adder.Service) func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		decoder := json.NewDecoder(r.Body)

		var newRestaurant adder.Restaurant
		err := decoder.Decode(&newRestaurant)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		newRestaurantID, err := s.AddRestaurant(newRestaurant)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			log.Printf("New restaurant id: %d", newRestaurantID)
			w.Header().Set("Content-Type", "application/json")
			rm := responseMessage{Message: fmt.Sprintf("New Restaurant added. ID: %d", newRestaurantID)}
			json.NewEncoder(w).Encode(rm)
		}

	}
}

func updateRestaurant(s updater.Service) func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		decoder := json.NewDecoder(r.Body)

		var updatedRestaurant updater.Restaurant
		err := decoder.Decode(&updatedRestaurant)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("Updating restaurant ID: %d\n", updatedRestaurant.ID)
		recordsAffected := s.UpdateRestaurant(updatedRestaurant)
		log.Printf("Number of records affected %d", recordsAffected)
		w.Header().Set("Content-Type", "application/json")
		rm := responseMessage{Message: fmt.Sprintf("Restaurant ID: %d updated", updatedRestaurant.ID)}
		json.NewEncoder(w).Encode(rm)
	}
}
