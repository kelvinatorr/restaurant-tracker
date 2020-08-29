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
	verbosePtr := flag.Bool("v", false, "Set -v for verbose logging.")
	flag.Parse()
	dbPath := *dbPathPtr
	verbose := *verbosePtr

	log.Printf("Connecting to database: %s\n", dbPath)

	s, err := sqlite.NewStorage(dbPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer s.CloseStorage()

	var add adder.Service
	var list lister.Service = lister.NewService(&s)
	var update updater.Service = updater.NewService(&s)
	var remove remover.Service = remover.NewService(&s)

	// http endpoints to receive data
	// set up the HTTP server
	router := rest.Handler(list, add, update, remove, verbose)

	log.Println("The restaurant tracker api server is on tap now: http://localhost:8888")
	log.Fatal(http.ListenAndServe(":8888", router))

	log.Println("Done with api server")
}
