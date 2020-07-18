package sqlite

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/kelvinatorr/restaurant-tracker/internal/remover"
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
				longitude
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
				CASE WHEN $8 == 0 THEN NULL ELSE $8 END
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
				website,
				restaurant_id
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
				CASE WHEN $10 == "" THEN NULL ELSE $10 END,
				$11
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
		g.RestaurantID,
	)
	checkAndPanic(err)
	lastID, err := res.LastInsertId()
	checkAndPanic(err)
	return lastID
}

func generateRestaurantSQL() string {
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
			left join gmaps_place as gp on gp.restaurant_id = res.id
	`

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
	sqlStatement := generateRestaurantSQL()
	// Add where clause by restaurant id
	sqlStatement = sqlStatement + `
		WHERE
			res.id=$1
	`
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
	sqlStatement := generateRestaurantSQL()
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

// GetRestaurantsByCity gives you all the restaurants with a given city id.
func (s Storage) GetRestaurantsByCity(cityID int64) []lister.Restaurant {
	var restaurantsInCity []lister.Restaurant
	var r lister.Restaurant
	// Generate the get sql statement without the where clause.
	sqlStatement := generateRestaurantSQL()
	sqlStatement = sqlStatement + `
		WHERE
			city.id=$1
	`
	dbRows, err := s.db.Query(sqlStatement, cityID)
	checkAndPanic(err)
	defer dbRows.Close()
	for dbRows.Next() {
		err = fillRestaurant(dbRows, &r)
		checkAndPanic(err)
		restaurantsInCity = append(restaurantsInCity, r)
	}
	err = dbRows.Err()
	checkAndPanic(err)
	return restaurantsInCity
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
			longitude = CASE WHEN $8 == 0 THEN NULL ELSE $8 END
		WHERE
			id = $9
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
		r.ID,
	)
	checkAndPanic(err)
	rowsAffected, err := res.RowsAffected()
	checkAndPanic(err)
	return rowsAffected
}

// UpdateGmapsPlace updates a given gmaps_place, returns the rows affected. Caller must call Commit() to commit the
// transaction
func (s Storage) UpdateGmapsPlace(gp updater.GmapsPlace) int64 {
	// We use case when to allow updating to nulls in the database
	sqlStatement := `
		UPDATE
			gmaps_place
		SET
			last_updated = CASE WHEN $1 == "" THEN NULL ELSE $1 END,
			place_id = $2,
			business_status = CASE WHEN $3 == "" THEN NULL ELSE $3 END,
			formatted_phone_number = CASE WHEN $4 == "" THEN NULL ELSE $4 END,
			name = $5,
			price_level = CASE WHEN $6 == 0 THEN NULL ELSE $6 END,
			rating = CASE WHEN $7 == 0 THEN NULL ELSE $7 END,
			url = CASE WHEN $8 == "" THEN NULL ELSE $8 END,
			user_ratings_total = CASE WHEN $9 == 0 THEN NULL ELSE $9 END,
			utc_offset = CASE WHEN $10 == 0 THEN NULL ELSE $10 END,
			website = CASE WHEN $11 == "" THEN NULL ELSE $11 END,
			restaurant_id = $12
		WHERE
			id = $13
	`

	res, err := s.tx.Exec(sqlStatement,
		gp.LastUpdated,
		gp.PlaceID,
		gp.BusinessStatus,
		gp.FormattedPhoneNumber,
		gp.Name,
		gp.PriceLevel,
		gp.Rating,
		gp.URL,
		gp.UserRatingsTotal,
		gp.UTCOffset,
		gp.Website,
		gp.RestaurantID,
		gp.ID,
	)
	checkAndPanic(err)
	rowsAffected, err := res.RowsAffected()
	checkAndPanic(err)
	return rowsAffected
}

// RemoveRestaurant deletes a given restaurant from the database and returns the rows affected. Caller must call
// Commit() to commit the transaction
func (s Storage) RemoveRestaurant(r remover.Restaurant) int64 {
	return s.removeRow("restaurant", r.ID)
}

// RemoveGmapsPlace deletes a given gmaps_place from the database and returns the rows affected. Caller must call
// Commit() to commit the transaction
func (s Storage) RemoveGmapsPlace(gpID int64) int64 {
	return s.removeRow("gmaps_place", gpID)
}

// RemoveCity deletes a given city and returns the number of rows affected. Caller must call Commit() to commit the
// transaction
func (s Storage) RemoveCity(cityID int64) int64 {
	return s.removeRow("city", cityID)
}

func (s Storage) removeRow(tableName string, rowID int64) int64 {
	sqlStatement := `
		DELETE FROM
			%s
		WHERE
			id = $1
	`
	// Never pass tableName from user input!
	sqlStatement = fmt.Sprintf(sqlStatement, tableName)

	res, err := s.tx.Exec(sqlStatement, rowID)
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
