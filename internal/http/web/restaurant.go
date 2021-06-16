package web

import (
	"net/http"

	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
	"github.com/kelvinatorr/restaurant-tracker/internal/mapper"
)

func renderRestaurant(w http.ResponseWriter, r *http.Request, s lister.Service, m mapper.Service, restaurantID int, a Alert) {
	v := newView("base", "./web/template/restaurant.html")

	data := Data{}

	// If the alert is not empty then show it
	if a.Message != "" {
		data.Alert = a
	}

	cuisines := s.GetDistinct("cuisine", "restaurant")
	cities := s.GetDistinct("name", "city")
	states := s.GetDistinct("state", "city")

	haveGmapsKey := m.HaveGmapsKey()

	var restaurant lister.Restaurant
	// Get the restaurant requested
	if restaurantID != 0 {
		restaurant, err := s.GetRestaurant(int64(restaurantID))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		data.Head = Head{restaurant.Name}
		data.Yield = struct {
			Heading      string
			Text         string
			Restaurant   lister.Restaurant
			Cuisines     []string
			Cities       []string
			States       []string
			HaveGmapsKey bool
		}{
			restaurant.Name,
			"Edit this restaurant's details below",
			restaurant,
			cuisines,
			cities,
			states,
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
			Cuisines     []string
			Cities       []string
			States       []string
			HaveGmapsKey bool
		}{
			"Add A New Restaurant",
			"Add the new restaurant's details below",
			restaurant,
			cuisines,
			cities,
			states,
			haveGmapsKey,
		}
	}

	v.render(w, r, data)
}
