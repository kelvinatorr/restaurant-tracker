package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/kelvinatorr/restaurant-tracker/internal/adder"
	"github.com/kelvinatorr/restaurant-tracker/internal/http/rest"
	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
	"github.com/kelvinatorr/restaurant-tracker/internal/remover"
	"github.com/kelvinatorr/restaurant-tracker/internal/storage/sqlite"
	"github.com/kelvinatorr/restaurant-tracker/internal/updater"
)

func main() {
	log.Println("Starting api server.")
	// Flag for database path
	dbPathPtr := flag.String("db", "", "Path to the sqlite database. See README for instructions on how to make one.")
	flag.Parse()
	dbPath := *dbPathPtr

	s, err := sqlite.NewStorage(dbPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer s.CloseStorage()

	var add adder.Service
	add = adder.NewService(&s)

	gp := adder.GmapsPlace{
		PlaceID:              "ChIJ9_tgjT3AyIARfFErWP0PX70",
		BusinessStatus:       "OPERATIONAL",
		FormattedPhoneNumber: "(702) 778-5757",
		Name:                 "Bover",
		PriceLevel:           0,
		Rating:               4.7,
		URL:                  "https://maps.google.com/?cid=13645642976736268668",
		UserRatingsTotal:     51,
		UTCOffset:            -420,
		Website:              "",
	}
	r := adder.Restaurant{
		Name:       "Bover",
		Cuisine:    "Coffee & Tea",
		Note:       "First boba outing together during coronavirus time",
		CityState:  adder.CityState{Name: "Las Vegas", State: "NV"},
		GmapsPlace: gp,
		Address:    "1780 N Buffalo Dr #107",
		Zipcode:    "89128",
		Latitude:   36.1914303,
		Longitude:  -115.2592753,
	}
	newRID, err := add.AddRestaurant(r)
	if err != nil {
		log.Println(err)
	} else {
		log.Printf("New restaurant id: %d", newRID)
	}

	var list lister.Service = lister.NewService(&s)

	newR, err := list.GetRestaurant(12)
	if err != nil {
		log.Println(err)
	} else {
		log.Printf("Found it: %#v\n", newR)
	}

	gpu := updater.GmapsPlace{
		ID:                   466,
		LastUpdated:          "2020-07-07T22:15:44Z",
		PlaceID:              "ChIJ9_tgjT3AyIARfFErWP0PX70",
		BusinessStatus:       "OPERATIONAL",
		FormattedPhoneNumber: "(702) 778-5757",
		Name:                 "Bover",
		PriceLevel:           0,
		Rating:               4.7,
		URL:                  "https://maps.google.com/?cid=13645642976736268668",
		UserRatingsTotal:     51,
		UTCOffset:            -420,
		Website:              "",
		RestaurantID:         496,
	}
	// Test updating.
	ru := updater.Restaurant{
		ID:         496,
		Name:       "Bover",
		Cuisine:    "Coffee & Tea",
		Note:       "First boba outing together during coronavirus time",
		CityState:  updater.CityState{Name: "Las Vegas", State: "NV"},
		GmapsPlace: gpu,
		Address:    "1780 N Buffalo Dr #107",
		Zipcode:    "89128",
		Latitude:   36.1914303,
		Longitude:  -115.2592753,
	}

	var update updater.Service = updater.NewService(&s)
	rowsAffected := update.UpdateRestaurant(ru)
	log.Printf("Updated %s. Rows affected %d\n", ru.Name, rowsAffected)

	rr := remover.Restaurant{
		ID:     1,
		CityID: 22,
	}
	var remove remover.Service = remover.NewService(&s)
	rowsAffected = remove.RemoveRestaurant(rr)
	log.Printf("Removed %d. Total Rows affected %d\n", rr.ID, rowsAffected)

	// TODO: Add http endpoints to receive data
	// set up the HTTP server
	router := rest.Handler(list)

	log.Println("The restaurant tracker api server is on tap now: http://localhost:8888")
	log.Fatal(http.ListenAndServe(":8888", router))

	log.Println("Done with api server")
}
