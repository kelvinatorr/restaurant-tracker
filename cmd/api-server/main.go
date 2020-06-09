package main

import (
	"flag"
	"log"

	"github.com/kelvinatorr/restaurant-tracker/internal/adder"
	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
	"github.com/kelvinatorr/restaurant-tracker/internal/storage/sqlite"
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

	// TODO: Add http endpoints to receive data
	r := adder.Restaurant{
		Name:      "Bovar",
		Cuisine:   "Coffee & Tea",
		Note:      "First boba outing together during coronavirus time",
		CityState: adder.CityState{Name: "New York", State: "NY"},
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

	allRs := list.GetRestaurants()
	log.Printf("%#v\n", allRs)

	log.Println("Done with api server")
}
