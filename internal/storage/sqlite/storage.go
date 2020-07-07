package sqlite

import (
	"database/sql"
	"log"

	"github.com/kelvinatorr/restaurant-tracker/internal/updater"

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
	// We use case when to allow inserting nulls in the database
	sqlStatement := `
		INSERT INTO 
			restaurant(
				name,
				cuisine,
				note,
				city_id,
				address,
				zipcode,
				latitude,
				longitude,
				gmaps_place_id
			)
		VALUES
			(
				$1,
				$2,
				CASE WHEN $3 == "" THEN NULL ELSE $3 END,
				$4,
				CASE WHEN $5 == "" THEN NULL ELSE $5 END,
				CASE WHEN $6 == "" THEN NULL ELSE $6 END,
				CASE WHEN $7 == 0 THEN NULL ELSE $7 END,
				CASE WHEN $8 == 0 THEN NULL ELSE $8 END,
				CASE WHEN $9 == 0 THEN NULL ELSE $9 END
			)
	`
	res, err := s.tx.Exec(sqlStatement,
		r.Name,
		r.Cuisine,
		r.Note,
		r.CityID,
		r.Address,
		r.Zipcode,
		r.Latitude,
		r.Longitude,
		r.GmapsPlaceID,
	)
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
		`, r.Name, r.CityState.Name, r.CityState.State)
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
func (s Storage) GetCityIDByNameAndState(cityName string, stateName string) int64 {
	// upper() so we get better matching
	sqlStatement := `
		SELECT 
			id 
		FROM 
			city 
		WHERE 
			upper(name)=upper($1) 
			and upper(state)=upper($2)
	`
	var id int64
	row := s.db.QueryRow(sqlStatement, cityName, stateName)
	err := row.Scan(&id)
	if err != sql.ErrNoRows {
		checkAndPanic(err)
	}
	return id
}

// AddCity adds a city to the city table and returns the primary key id. Must call Commit() to commit transaction
func (s Storage) AddCity(cityName string, stateName string) int64 {
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

// AddGmapsPlace adds a Google Maps Place to the database and returns the primary key id. Must call Commit() to commit
// the transaction
func (s Storage) AddGmapsPlace(g adder.GmapsPlace) int64 {
	// We use case when to allow inserting nulls in the database
	sqlStatement := `
		INSERT INTO 
			gmaps_place(
				place_id,
				business_status,
				formatted_phone_number,
				name,
				price_level,
				rating,
				url,
				user_ratings_total,
				utc_offset,
				website
			)
		VALUES
			(
				$1,
				CASE WHEN $2 == "" THEN NULL ELSE $2 END, 
				CASE WHEN $3 == "" THEN NULL ELSE $3 END, 
				$4,
				CASE WHEN $5 == 0 THEN NULL ELSE $5 END,
				CASE WHEN $6 == 0 THEN NULL ELSE $6 END,
				CASE WHEN $7 == "" THEN NULL ELSE $7 END,
				CASE WHEN $8 == 0 THEN NULL ELSE $8 END,
				CASE WHEN $9 == 0 THEN NULL ELSE $9 END,
				CASE WHEN $10 == "" THEN NULL ELSE $10 END
			)
	`
	res, err := s.tx.Exec(sqlStatement,
		g.PlaceID,
		g.BusinessStatus,
		g.FormattedPhoneNumber,
		g.Name,
		g.PriceLevel,
		g.Rating,
		g.URL,
		g.UserRatingsTotal,
		g.UTCOffset,
		g.Website,
	)
	checkAndPanic(err)
	lastID, err := res.LastInsertId()
	checkAndPanic(err)
	return lastID
}

func generateRestaurantSQL(single bool) string {
	// Need COALESCE because this is the least ugly way to handle nullable columns in go
	sql := `
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
	`
	if single {
		sql = sql + `
				WHERE
				res.id=$1
		`
	}
	return sql
}

// Implements the Scan function of sql.Row and sql.Rows
type scanner interface {
	Scan(...interface{}) error
}

func fillRestaurant(row scanner, r *lister.Restaurant) error {
	return row.Scan(
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
}

// GetRestaurant queries the restaurant table for the given id. If the returned restaurant has ID = 0 then it is not in
// the database
func (s Storage) GetRestaurant(id int64) lister.Restaurant {
	var r lister.Restaurant
	sqlStatement := generateRestaurantSQL(true)
	row := s.db.QueryRow(sqlStatement, id)
	err := fillRestaurant(row, &r)
	if err != sql.ErrNoRows {
		checkAndPanic(err)
	}
	return r
}

// GetRestaurants queries the restaurant table for all restaurants.
func (s Storage) GetRestaurants() []lister.Restaurant {
	var allResturants []lister.Restaurant
	var r lister.Restaurant
	// Generate the get sql statement without the where clause.
	sqlStatement := generateRestaurantSQL(false)
	dbRows, err := s.db.Query(sqlStatement)
	checkAndPanic(err)
	defer dbRows.Close()
	for dbRows.Next() {
		err = fillRestaurant(dbRows, &r)
		checkAndPanic(err)
		allResturants = append(allResturants, r)
	}
	err = dbRows.Err()
	checkAndPanic(err)
	return allResturants
}

// UpdateRestaurant updates a given restaurant, returns the rows affected. Must call Commit() to commit transaction
func (s Storage) UpdateRestaurant(r updater.Restaurant) int64 {
	// We use case when to allow updating to nulls in the database
	sqlStatement := `
		UPDATE
			restaurant
		SET
			name = $1,
			cuisine = $2,
			note = CASE WHEN $3 == "" THEN NULL ELSE $3 END,
			city_id = $4,
			address = CASE WHEN $5 == "" THEN NULL ELSE $5 END,
			zipcode = CASE WHEN $6 == 0 THEN NULL ELSE $6 END,
			latitude = CASE WHEN $7 == 0 THEN NULL ELSE $7 END,
			longitude = CASE WHEN $8 == 0 THEN NULL ELSE $8 END,
			gmaps_place_id = CASE WHEN $9 == 0 THEN NULL ELSE $9 END
		WHERE
			id = $10
	`

	res, err := s.tx.Exec(sqlStatement,
		r.Name,
		r.Cuisine,
		r.Note,
		r.CityID,
		r.Address,
		r.Zipcode,
		r.Latitude,
		r.Longitude,
		r.GmapsPlaceID,
		r.ID,
	)
	checkAndPanic(err)
	rowsAffected, err := res.RowsAffected()
	checkAndPanic(err)
	return rowsAffected
}

func checkAndPanic(err error) {
	if err != nil {
		log.Panicln(err)
	}
}
