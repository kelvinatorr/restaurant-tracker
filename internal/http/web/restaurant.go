package web

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/kelvinatorr/restaurant-tracker/internal/adder"
	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
	"github.com/kelvinatorr/restaurant-tracker/internal/mapper"
	"github.com/kelvinatorr/restaurant-tracker/internal/remover"
	"github.com/kelvinatorr/restaurant-tracker/internal/updater"
)

func getRestaurant(s lister.Service, m mapper.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		// get the route parameter
		ID, err := strconv.Atoi(p.ByName("id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid restaurant ID, it must be a number.", p.ByName("id")),
				http.StatusBadRequest)
			return
		}
		renderRestaurant(w, r, s, m, ID, Alert{})
	}
}

func postRestaurant(u updater.Service, a adder.Service, m mapper.Service, l lister.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ID, err := strconv.Atoi(p.ByName("id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid restaurant ID, it must be a number.", p.ByName("id")),
				http.StatusBadRequest)
			return
		}
		if ID != 0 {
			updateRestaurant(u, m, w, r, l)
		} else {
			addRestaurant(a, m, w, r, l)
		}
	}
}

func addRestaurant(a adder.Service, m mapper.Service, w http.ResponseWriter, r *http.Request, l lister.Service) {
	var resNew adder.Restaurant
	if err := parseForm(r, &resNew); err != nil {
		log.Println(err)
		http.Error(w, AlertFormParseErrorGeneric, http.StatusInternalServerError)
		return
	}

	newRestaurantID, err := a.AddRestaurant(resNew)
	if err != nil {
		log.Println(err)
		v := newView("base", "./web/template/restaurant.html")
		data := Data{}
		data.Head = Head{"Add A New Restaurant"}
		// Show the user the error.
		data.Alert = Alert{Message: err.Error(), Class: AlertClassError}

		// Fill in the form again for convenience. Need lister.Restaurant because we need an ID property for the template
		restaurant := lister.Restaurant{
			Name:           resNew.Name,
			Cuisine:        resNew.Cuisine,
			BusinessStatus: resNew.BusinessStatus,
			Note:           resNew.Note,
			CityState: lister.CityState{
				Name:  resNew.CityState.Name,
				State: resNew.CityState.State,
			},
		}

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
			l.GetDistinct("cuisine", "restaurant"),
			l.GetDistinct("name", "city"),
			l.GetDistinct("state", "city"),
			m.HaveGmapsKey(),
		}
		v.render(w, r, data)
		return
	}
	log.Printf("%s added with id %d", resNew.Name, newRestaurantID)
	http.Redirect(w, r, fmt.Sprintf("/restaurants/%d", newRestaurantID), http.StatusFound)
}

func updateRestaurant(u updater.Service, m mapper.Service, w http.ResponseWriter, r *http.Request, l lister.Service) {
	var resUpdate updater.Restaurant
	if err := parseForm(r, &resUpdate); err != nil {
		log.Println(err)
		http.Error(w, AlertFormParseErrorGeneric, http.StatusInternalServerError)
		return
	}

	recordsAffected, err := u.UpdateRestaurant(resUpdate)
	if err != nil {
		log.Println(err)
		v := newView("base", "./web/template/restaurant.html")
		data := Data{}
		data.Head = Head{resUpdate.Name}
		// Show the user the error.
		data.Alert = Alert{Message: err.Error(), Class: AlertClassError}
		// Fill in the form again for convenience
		data.Yield = struct {
			Heading      string
			Text         string
			Restaurant   updater.Restaurant
			Cuisines     []string
			Cities       []string
			States       []string
			HaveGmapsKey bool
		}{
			resUpdate.Name,
			"Edit this restuarant's details below",
			resUpdate,
			l.GetDistinct("cuisine", "restaurant"),
			l.GetDistinct("name", "city"),
			l.GetDistinct("state", "city"),
			m.HaveGmapsKey(),
		}
		v.render(w, r, data)
		return
	}
	log.Printf("Updated restaurant with ID: %d. %d records affected\n", resUpdate.ID, recordsAffected)

	updateSuccessMsg := "Restaurant updated"
	renderRestaurant(w, r, l, m, int(resUpdate.ID), Alert{Class: AlertClassSuccess, Message: updateSuccessMsg})
}

func getDeleteRestaurant(l lister.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ID, err := strconv.Atoi(p.ByName("id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid restaurant ID, it must be a number.", p.ByName("id")),
				http.StatusBadRequest)
			return
		}

		v := newView("base", "./web/template/delete-restaurant.html")

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

		v.render(w, r, data)
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
			v := newView("base", "./web/template/delete-restaurant.html")
			data := Data{}
			data.Head = Head{deleteConfirm.Name}
			// Show the user the error.
			data.Alert = Alert{Message: fmt.Sprintf("Input: %s did not match %s", deleteConfirm.ConfirmName, deleteConfirm.Name),
				Class: AlertClassError}
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
			v.render(w, r, data)
			return
		} else {
			log.Printf("Confirmed request to remove %s with ID: %d", deleteConfirm.Name, ID)
			s.RemoveRestaurant(remover.Restaurant{ID: int64(ID)})
			// Redirect to the list of other restaurants.
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	}
}

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
		restaurant.BusinessStatus = 1
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
