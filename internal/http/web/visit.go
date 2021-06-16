package web

import (
	"fmt"
	"log"
	"net/http"

	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
)

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
