package sqlite

import (
	"database/sql"
	"log"

	"github.com/kelvinatorr/restaurant-tracker/internal/adder"

	_ "github.com/mattn/go-sqlite3" // Blank importing here because this is where we interact with the db
)

// NewStorage returns a new database/sql instance initialized with sqlite3
func NewStorage(dbPath string) (Storage, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_fk=on")
	s := Storage{db}
	return s, err
}

// Storage stores restaurant data in a sqlite3 database
type Storage struct {
	db *sql.DB
}

// CloseStorage closes the database by calling db.Close()
func (s Storage) CloseStorage() {
	s.db.Close()
}

// AddRestaurant saves the given restaurant to the sqlite database.
func (s Storage) AddRestaurant(r adder.Restaurant) int64 {
	sqlStatement := `
		INSERT INTO 
			restaurant(name, cuisine, note, city_id)
		VALUES
			($1, $2, $3, $4)
	`
	res, err := s.db.Exec(sqlStatement, r.Name, r.Cuisine, r.Note, r.CityID)
	checkAndPanic(err)
	lastID, err := res.LastInsertId()
	checkAndPanic(err)
	return lastID
}

// IsDuplicateRestaurant returns true if the database already has a restaurant with the same name in the same city and
// state
func (s Storage) IsDuplicateRestaurant(r adder.Restaurant) bool {
	// Query the database for a restaurant with the same name and city name and city state
	dbRows, err := s.db.Query(`
		SELECT 
			restaurant.id
		FROM 
			restaurant
			left join city on city.id = restaurant.city_id
		WHERE
			upper(restaurant.name) = upper($1)
			and upper(city.name) = upper($2)
			and upper(city.state) = upper($3)
		`, r.Name, r.City, r.State)
	checkAndPanic(err)
	defer dbRows.Close()
	var id int64
	for dbRows.Next() {
		err = dbRows.Scan(&id)
		checkAndPanic(err)
	}
	err = dbRows.Err()
	checkAndPanic(err)
	return id != 0
}

// GetCityIDByNameAndState queries the database for a given city name and state name, returns the id of the row if it
// exists
func (s Storage) GetCityIDByNameAndState(r adder.Restaurant) int64 {
	// upper() so we get better matching
	sqlStatement := `SELECT id FROM city WHERE upper(name)=upper($1) and upper(state)=upper($2);`
	var id int64
	row := s.db.QueryRow(sqlStatement, r.City, r.State)
	err := row.Scan(&id)
	if err != sql.ErrNoRows {
		checkAndPanic(err)
	}
	return id
}

// AddCity adds a city to the city table and returns the primary key id
func (s Storage) AddCity(r adder.Restaurant) int64 {
	cityName := r.City
	stateName := r.State
	sqlStatement := `
		INSERT INTO 
			city(name, state)
		VALUES
			($1, $2)
	`
	res, err := s.db.Exec(sqlStatement, cityName, stateName)
	checkAndPanic(err)
	lastID, err := res.LastInsertId()
	checkAndPanic(err)
	return lastID
}

func checkAndPanic(err error) {
	if err != nil {
		log.Panicln(err)
	}
}
