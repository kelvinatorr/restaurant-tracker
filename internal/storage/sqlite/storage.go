package sqlite

import (
	"database/sql"
	"log"

	"github.com/kelvinatorr/restaurant-tracker/internal/lister"

	"github.com/kelvinatorr/restaurant-tracker/internal/adder"

	_ "github.com/mattn/go-sqlite3" // Blank importing here because this is where we interact with the db
)

// NewStorage returns a new database/sql instance initialized with sqlite3
func NewStorage(dbPath string) (Storage, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_fk=on")
	s := Storage{db: db}
	return s, err
}

// Storage stores restaurant data in a sqlite3 database
type Storage struct {
	db *sql.DB
	tx *sql.Tx
}

// CloseStorage closes the database by calling db.Close()
func (s Storage) CloseStorage() {
	s.db.Close()
}

// Begin starts a transaction
func (s *Storage) Begin() {
	tx, err := s.db.Begin()
	checkAndPanic(err)
	s.tx = tx
}

// Commit "saves" the transaction to the database
func (s Storage) Commit() {
	err := s.tx.Commit()
	checkAndPanic(err)
}

// Rollback rolls the inserts/updates back
func (s Storage) Rollback() {
	s.tx.Rollback()
}

// AddRestaurant adds the given restaurant to the sqlite database. Must call Commit() to commit transaction
func (s Storage) AddRestaurant(r adder.Restaurant) int64 {
	sqlStatement := `
		INSERT INTO 
			restaurant(name, cuisine, note, city_id)
		VALUES
			($1, $2, $3, $4)
	`
	res, err := s.tx.Exec(sqlStatement, r.Name, r.Cuisine, r.Note, r.CityID)
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

// AddCity adds a city to the city table and returns the primary key id. Must call Commit() to commit transaction
func (s Storage) AddCity(r adder.Restaurant) int64 {
	cityName := r.City
	stateName := r.State
	sqlStatement := `
		INSERT INTO 
			city(name, state)
		VALUES
			($1, $2)
	`
	res, err := s.tx.Exec(sqlStatement, cityName, stateName)
	checkAndPanic(err)
	lastID, err := res.LastInsertId()
	checkAndPanic(err)
	return lastID
}

// GetRestaurant queries the restaurant table for the given id. If the returned restaurant has ID = 0 then it is not in
// the database
func (s Storage) GetRestaurant(id int64) lister.Restaurant {
	var r lister.Restaurant
	// Need COALESCE because this is the least ugly way to handle nullable columns in go
	sqlStatement := `
		SELECT
			res.id,
    		res.name,
    		cuisine,
    		note,
    		COALESCE(address, "") as address,
    		COALESCE(zipcode, "") as zipcode,
    		COALESCE(latitude, 0) as latitude,
			COALESCE(longitude, 0) as longitude,
			city.id as city_id,
			city.name as city_name,
			city.state as state_name,
			COALESCE(gp.id, 0) as gmaps_place_id,
			COALESCE(last_updated, ""),
			COALESCE(place_id, ""),
			COALESCE(business_status, "") as business_status,
			COALESCE(formatted_phone_number, "") as formatted_phone_number,
			COALESCE(gp.name, "") as gmaps_place_name,
			COALESCE(price_level, 0) as price_level,
			COALESCE(rating, 0) rating,
			COALESCE(url, "") as url,
			COALESCE(user_ratings_total, 0) as user_ratings_total,
			COALESCE(utc_offset, 0) as utc_offset,
			COALESCE(website, "") as website
		FROM
			restaurant as res
			inner join city on city.id = res.city_id
			left join gmaps_place as gp on gp.id = res.gmaps_place_id
		WHERE
			res.id=$1
		;
	`
	row := s.db.QueryRow(sqlStatement, id)
	err := row.Scan(
		&r.ID,
		&r.Name,
		&r.Cuisine,
		&r.Note,
		&r.Address,
		&r.Zipcode,
		&r.Latitude,
		&r.Longitude,
		&r.CityState.ID,
		&r.CityState.Name,
		&r.CityState.State,
		&r.GmapsPlace.ID,
		&r.GmapsPlace.LastUpdated,
		&r.GmapsPlace.PlaceID,
		&r.GmapsPlace.BusinessStatus,
		&r.GmapsPlace.FormattedPhoneNumber,
		&r.GmapsPlace.Name,
		&r.GmapsPlace.PriceLevel,
		&r.GmapsPlace.Rating,
		&r.GmapsPlace.URL,
		&r.GmapsPlace.UserRatingsTotal,
		&r.GmapsPlace.UTCOffset,
		&r.GmapsPlace.Website,
	)
	if err != sql.ErrNoRows {
		checkAndPanic(err)
	}
	return r
}

func checkAndPanic(err error) {
	if err != nil {
		log.Panicln(err)
	}
}
