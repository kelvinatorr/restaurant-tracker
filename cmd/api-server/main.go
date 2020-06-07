package main

import (
	"log"

	"github.com/kelvinatorr/restaurant-tracker/internal/adder"
	"github.com/kelvinatorr/restaurant-tracker/internal/storage/sqlite"
)

func main() {
	log.Println("Starting api server.")
	// TODO: add flag for database path

	var add adder.Service

	// error handling omitted for simplicity
	// TODO: Add error handling
	s, _ := sqlite.NewStorage("/home/kelvin/Github.com/restaurant-tracker/database/kelvin-0-hand-clean.db")
	defer s.CloseStorage()

	add = adder.NewService(s)

	// TODO: Add http endpoints to receive data
	r := adder.Restaurant{
		Name:    "Bovar",
		Cuisine: "Coffee & Tea",
		Note:    "First boba outing together during coronavirus time",
		City:    "New York",
		State:   "NY",
	}
	newRID, err := add.AddRestaurant(r)
	if err != nil {
		log.Println(err)
	} else {
		log.Printf("New restaurant id: %d", newRID)
	}	
	log.Println("Done with api server")
}
