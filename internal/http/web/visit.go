package web

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"github.com/kelvinatorr/restaurant-tracker/internal/adder"
	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
	"github.com/kelvinatorr/restaurant-tracker/internal/remover"
	"github.com/kelvinatorr/restaurant-tracker/internal/updater"
)

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

		// get the query parameters parameter
		queryParams := r.URL.Query()

		// Then we get its visits
		visits, err := l.GetVisitsByRestaurantID(resID, queryParams)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "There was a problem processing your request", http.StatusBadRequest)
			return
		}

		v := newView("base", "./web/template/visits.html")

		data := Data{}

		data.Head = Head{fmt.Sprintf("%s Visits", restaurant.Name)}
		data.Yield = struct {
			Heading      string
			RestaurantID int64
			Visits       []lister.Visit
		}{
			fmt.Sprintf("%s", restaurant.Name),
			restaurant.ID,
			visits,
		}

		v.render(w, r, data)
	}
}

func getVisit(l lister.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		resID, err := strconv.Atoi(p.ByName("resid"))
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid restaurant ID, it must be a number.", p.ByName("resid")),
				http.StatusBadRequest)
			return
		}

		resID64 := int64(resID)
		// Get the restaurant 1st so we can show its name and make sure it exists
		restaurant, err := l.GetRestaurant(resID64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		ID, err := strconv.Atoi(p.ByName("id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid visit ID, it must be a number.", p.ByName("id")),
				http.StatusBadRequest)
			return
		}

		renderVisit(w, r, restaurant, l, ID, Alert{})
	}
}

func postVisit(u updater.Service, a adder.Service, l lister.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ID, err := strconv.Atoi(p.ByName("id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid visit ID, it must be a number.", p.ByName("id")),
				http.StatusBadRequest)
			return
		}
		if ID != 0 {
			updateVisit(u, l, w, r)
		} else {
			addVisit(a, l, w, r)
		}
	}
}

func updateVisit(u updater.Service, l lister.Service, w http.ResponseWriter, r *http.Request) {
	var visitUpdate updater.Visit
	if err := parseForm(r, &visitUpdate); err != nil {
		log.Println(err)
		http.Error(w, AlertFormParseErrorGeneric, http.StatusInternalServerError)
		return
	}

	restaurant, err := l.GetRestaurant(visitUpdate.RestaurantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	recordsAffected, err := u.UpdateVisit(visitUpdate)
	if err != nil {
		updateErrorMsg := err.Error()
		log.Println(updateErrorMsg)

		visit := lister.Visit{
			ID:            visitUpdate.ID,
			RestaurantID:  visitUpdate.RestaurantID,
			VisitDateTime: visitUpdate.VisitDateTime,
			Note:          visitUpdate.Note,
		}
		for _, vu := range visitUpdate.VisitUsers {
			lvu := lister.VisitUser{ID: vu.ID, User: l.GetUserByID(vu.UserID), Rating: vu.Rating}
			visit.VisitUsers = append(visit.VisitUsers, lvu)
		}

		v := newView("base", "./web/template/visit.html")

		data := Data{}
		// Show the user the error.
		data.Alert = Alert{Message: updateErrorMsg, Class: AlertClassError}
		data.Head = Head{fmt.Sprintf("Edit Visit %s", restaurant.Name)}
		data.Yield = struct {
			Heading string
			Text    string
			Visit   lister.Visit
		}{
			fmt.Sprintf("Edit Visit to %s", restaurant.Name),
			"Add the date and optional note for your visit below",
			visit,
		}
		v.render(w, r, data)
		return
	}

	log.Printf("Updated visit with ID: %d. %d records affected\n", visitUpdate.ID, recordsAffected)

	updateSuccessMsg := fmt.Sprintf("Visit to %s updated.", restaurant.Name)
	renderVisit(w, r, restaurant, l, int(visitUpdate.ID), Alert{Class: AlertClassSuccess, Message: updateSuccessMsg})
}

func addVisit(a adder.Service, l lister.Service, w http.ResponseWriter, r *http.Request) {
	var visitNew adder.Visit
	if err := parseForm(r, &visitNew); err != nil {
		log.Println(err)
		http.Error(w, AlertFormParseErrorGeneric, http.StatusInternalServerError)
		return
	}

	newVisitID, err := a.AddVisit(visitNew)
	if err != nil {
		errorMsg := err.Error()
		log.Println(errorMsg)
		restaurant, err := l.GetRestaurant(visitNew.RestaurantID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// Create a new visit but with what the user typed in
		visit := lister.Visit{
			ID:            0,
			RestaurantID:  visitNew.RestaurantID,
			VisitDateTime: visitNew.VisitDateTime,
			Note:          visitNew.Note,
		}
		for _, vu := range visitNew.VisitUsers {
			lvu := lister.VisitUser{ID: 0, User: l.GetUserByID(vu.UserID), Rating: vu.Rating}
			visit.VisitUsers = append(visit.VisitUsers, lvu)
		}

		v := newView("base", "./web/template/visit.html")

		data := Data{}
		// Show the user the error.
		data.Alert = Alert{Message: errorMsg, Class: AlertClassError}
		data.Head = Head{fmt.Sprintf("Add Visit %s", restaurant.Name)}
		data.Yield = struct {
			Heading string
			Text    string
			Visit   lister.Visit
		}{
			fmt.Sprintf("Add a Visit to %s", restaurant.Name),
			"Add the date and optional note for your visit below",
			visit,
		}
		v.render(w, r, data)
		return
	}

	log.Printf("Added new visit to restaurant %d with ID: %d.\n", visitNew.RestaurantID, newVisitID)
	// Redirect to the list which should show the new entry
	http.Redirect(w, r, fmt.Sprintf("/r/%d/visits", visitNew.RestaurantID), http.StatusFound)
}

func getDeleteVisit(l lister.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		resID, err := strconv.Atoi(p.ByName("resid"))
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid restaurant ID, it must be a number.", p.ByName("resid")),
				http.StatusBadRequest)
			return
		}

		resID64 := int64(resID)
		// Get the restaurant 1st so we can show its name and make sure it exists
		restaurant, err := l.GetRestaurant(resID64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		ID, err := strconv.Atoi(p.ByName("id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid visit ID, it must be a number.", p.ByName("id")),
				http.StatusBadRequest)
			return
		}

		visit, err := l.GetVisit(int64(ID), resID64)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		v := newView("base", "./web/template/delete-visit.html")

		data := Data{}
		titleHeading := fmt.Sprintf("Delete Visit To %s", restaurant.Name)
		data.Head = Head{titleHeading}
		data.Yield = struct {
			Heading    string
			Restaurant lister.Restaurant
			Visit      lister.Visit
		}{
			titleHeading,
			restaurant,
			visit,
		}

		v.render(w, r, data)
	}
}

func postDeleteVisit(s remover.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		ID, err := strconv.Atoi(p.ByName("id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("%s is not a valid visit ID, it must be a number.", p.ByName("id")),
				http.StatusBadRequest)
			return
		}

		deleteConfirm := struct {
			RestaurantName string `schema:"restaurantName"`
			RestaurantID   int    `schema:"restaurantID"`
			VisitDateTime  string `schema:"VisitDateTime"`
		}{
			"",
			0,
			"",
		}
		if err := parseForm(r, &deleteConfirm); err != nil {
			log.Println(err)
			http.Error(w, AlertFormParseErrorGeneric, http.StatusInternalServerError)
			return
		}

		log.Printf("Confirmed request to remove visit to %s on %s with ID: %d", deleteConfirm.RestaurantName,
			deleteConfirm.VisitDateTime, ID)
		s.RemoveVisit(remover.Visit{ID: int64(ID)})
		// Redirect to the list of other visits.
		http.Redirect(w, r, fmt.Sprintf("/r/%d/visits", deleteConfirm.RestaurantID), http.StatusSeeOther)
	}
}

func renderVisit(w http.ResponseWriter, r *http.Request, restaurant lister.Restaurant, l lister.Service, visitID int, a Alert) {
	v := newView("base", "./web/template/visit.html")

	title_template := "%s Visit %s"
	heading_template := "%s a Visit to %s"
	text := "Add the date and optional note for your visit below"

	data := Data{}

	if a.Message != "" {
		data.Alert = a
	}

	if visitID == 0 {
		visit := lister.Visit{
			ID:            0,
			RestaurantID:  restaurant.ID,
			VisitDateTime: "",
			Note:          "",
		}
		for _, user := range l.GetUsers() {
			lvu := lister.VisitUser{ID: 0, User: user, Rating: 0}
			visit.VisitUsers = append(visit.VisitUsers, lvu)
		}

		data.Head = Head{fmt.Sprintf(title_template, "Add", restaurant.Name)}
		data.Yield = struct {
			Heading string
			Text    string
			Visit   lister.Visit
		}{
			fmt.Sprintf(heading_template, "Add", restaurant.Name),
			text,
			visit,
		}

	} else {
		visit, err := l.GetVisit(int64(visitID), restaurant.ID)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		data.Head = Head{fmt.Sprintf(title_template, "Edit", restaurant.Name)}
		data.Yield = struct {
			Heading string
			Text    string
			Visit   lister.Visit
		}{
			fmt.Sprintf(heading_template, "Edit", restaurant.Name),
			text,
			visit,
		}
	}

	v.render(w, r, data)
}
