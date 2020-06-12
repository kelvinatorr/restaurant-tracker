package main

import (
	"fmt"

	"github.com/kelvinatorr/restaurant-tracker/internal/storage/sqlite"

	"github.com/kelvinatorr/restaurant-tracker/internal/adder"
)

func main() {
	fmt.Println("Hello from init data!")

	var add adder.Service

	// error handling omitted for simplicity
	s, _ := sqlite.NewStorage()

	add = adder.NewService(s)

	// add some sample data
	add.AddInitRestaurants("Kelvin")

	fmt.Println("Done adding init data.")
}
