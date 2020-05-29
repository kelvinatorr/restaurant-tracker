package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/kelvinatorr/restaurant-tracker/internal/adder"

	_ "github.com/mattn/go-sqlite3"
)

// Storage stores beer data in JSON files
type Storage struct {
	db *sql.DB
}

func NewStorage() (Storage, error) {
	db, err := sql.Open("sqlite3", "/home/kelvin/Github.com/restaurant-tracker/database/huh2.db?_fk=on")
	s := Storage{db}
	return s, err
}

func (s Storage) AddRestaurant(r adder.Restaurant) {
	fmt.Println("Add restaurant (from storage): " + r.Name)
}

// func (s Storage) AddSampleRestaurants(r string) {
// 	fmt.Println("Add sample resturants (from storage): " + r)
// }
