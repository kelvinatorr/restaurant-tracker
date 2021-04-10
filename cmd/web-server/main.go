package main

import (
	"crypto/rand"
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/csrf"
	"github.com/kelvinatorr/restaurant-tracker/internal/adder"
	"github.com/kelvinatorr/restaurant-tracker/internal/auther"
	"github.com/kelvinatorr/restaurant-tracker/internal/http/web"
	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
	"github.com/kelvinatorr/restaurant-tracker/internal/mapper"
	"github.com/kelvinatorr/restaurant-tracker/internal/remover"
	"github.com/kelvinatorr/restaurant-tracker/internal/storage/sqlite"
	"github.com/kelvinatorr/restaurant-tracker/internal/updater"
)

func main() {
	log.Println("Starting api server.")
	// Flag for database path
	dbPathPtr := flag.String("db", "", "Path to the sqlite database. See README for instructions on how to make one.")
	verbosePtr := flag.Bool("v", false, "Set -v for verbose logging.")
	csrfKeyPtr := flag.String("csrf", "", "A string that will be used as the anti-csrf key. A random one will be generated if not provided.")
	prodPtr := flag.Bool("prod", false, "Provide this flag in production.")
	flag.Parse()
	dbPath := *dbPathPtr
	verbose := *verbosePtr
	csrfKey := *csrfKeyPtr
	isProd := *prodPtr

	// Read the secret key env variable.
	secretKey := os.Getenv("SECRETKEY")
	if secretKey == "" {
		log.Fatalln("No SECRETKEY environment variable set.")
	}

	gmapsKey := os.Getenv("GMAPSKEY")
	if gmapsKey == "" {
		log.Println("GMAPSKEY not set. Google Maps functionality will be disabled")
	}

	log.Printf("Connecting to database: %s\n", dbPath)
	s, err := sqlite.NewStorage(dbPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer s.CloseStorage()

	var m mapper.Service = mapper.NewService(gmapsKey)
	var add adder.Service = adder.NewService(&s, m)
	var list lister.Service = lister.NewService(&s)
	var update updater.Service = updater.NewService(&s, m)
	var remove remover.Service = remover.NewService(&s)
	var auth auther.Service = auther.NewService(&s, secretKey)

	var csrfKeyBytes []byte
	if csrfKey == "" {
		b := make([]byte, 32)
		_, err := rand.Read(b)
		if err != nil {
			log.Fatalln(err)
		}
		csrfKeyBytes = b
	} else {
		csrfKeyBytes = []byte(csrfKey)
	}
	// Create the CSRF middleware
	csrfMw := csrf.Protect(csrfKeyBytes, csrf.Secure(isProd), csrf.MaxAge(0))

	// http endpoints to receive data
	// set up the HTTP server
	router := web.Handler(list, add, update, remove, auth, m, verbose)

	log.Println("The restaurant tracker web server is starting on: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", csrfMw(router)))

	log.Println("Done with web server")
}
