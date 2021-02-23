package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/kelvinatorr/restaurant-tracker/internal/auther"

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
		ON CONFLICT DO NOTHING
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
		ON CONFLICT(place_id) DO UPDATE
		SET			
			place_id = $1,
			business_status = CASE WHEN $2 == "" THEN NULL ELSE $2 END,
			formatted_phone_number = CASE WHEN $3 == "" THEN NULL ELSE $3 END,
			name = $4,
			price_level = CASE WHEN $5 == 0 THEN NULL ELSE $5 END,
			rating = CASE WHEN $6 == 0 THEN NULL ELSE $6 END,
			url = CASE WHEN $7 == "" THEN NULL ELSE $7 END,
			user_ratings_total = CASE WHEN $8 == 0 THEN NULL ELSE $8 END,
			utc_offset = CASE WHEN $9 == 0 THEN NULL ELSE $9 END,
			website = CASE WHEN $10 == "" THEN NULL ELSE $10 END,
			restaurant_id = $11,
			last_updated = $12
	`
	currentDateTime := time.Now()
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
		currentDateTime.Format("2006-01-02T15:04:05Z"),
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
    		res.note,
    		COALESCE(address, "") as address,
    		COALESCE(zipcode, "") as zipcode,
    		COALESCE(latitude, 0) as latitude,
			COALESCE(longitude, 0) as longitude,
			COALESCE(last_visits.last_visit, "") as last_visit_datetime,
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
			COALESCE(website, "") as website,
			COALESCE(ratings.avg_rating, 0) as avg_rating
		FROM
			restaurant as res
			inner join city on city.id = res.city_id
			left join gmaps_place as gp on gp.restaurant_id = res.id
			left join (
				SELECT
					restaurant_id,
					max(visit_datetime) as last_visit
				FROM
					visit
				GROUP BY
					restaurant_id
			) as last_visits on last_visits.restaurant_id = res.id
			left join (
				SELECT
					v.restaurant_id,
					avg(rating) as avg_rating
				FROM
					visit_user as vu
					left join visit as v on v.id = vu.visit_id
				GROUP BY
					v.restaurant_id
			) as ratings on ratings.restaurant_id = res.id
	`

	return sql
}

func generateVisitSQL() string {
	// Need COALESCE because this is the least ugly way to handle nullable columns in go
	sql := `
		SELECT
			v.id,
			v.restaurant_id,
			visit_datetime,
			COALESCE(v.note, "") as note
		FROM
			visit as v
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
		&r.LastVisitDatetime,
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
		&r.AvgRating,
	)
}

func addSortOps(sqlStatement string, sortOp lister.SortOperation, last bool) string {
	formatString := "%s %s"
	if !last {
		formatString = formatString + ","
	}
	sqlStatement = sqlStatement + fmt.Sprintf(formatString, sortOp.Field, sortOp.Direction) + "\n"
	return sqlStatement
}

func addFilterOps(sqlStatement string, filterOp lister.FilterOperation, first bool) string {
	formatString := "%s %s CAST('%s' as %s)"
	if !first {
		formatString = "AND " + formatString
	}

	sqlStatement = sqlStatement + fmt.Sprintf(formatString, filterOp.Field, filterOp.Operator, filterOp.Value, filterOp.FieldType)
	sqlStatement = sqlStatement + "\n"
	return sqlStatement
}

func fillVisit(row scanner, v *lister.Visit) error {
	return row.Scan(
		&v.ID,
		&v.RestaurantID,
		&v.VisitDateTime,
		&v.Note,
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
func (s Storage) GetRestaurants(sortOps []lister.SortOperation, filterOps []lister.FilterOperation) []lister.Restaurant {
	var allResturants []lister.Restaurant
	var r lister.Restaurant
	// Generate the get sql statement without the where clause.
	sqlStatement := generateRestaurantSQL()

	nFilterOps := len(filterOps)
	if nFilterOps > 0 {
		sqlStatement = sqlStatement + `
			WHERE
		`
		// add filter statements
		for i, so := range filterOps {
			sqlStatement = addFilterOps(sqlStatement, so, i == 0)
		}
	}

	nSortOps := len(sortOps)
	if nSortOps > 0 {
		sqlStatement = sqlStatement + `
			ORDER BY
		`
		// add sort statements
		for i, so := range sortOps {
			sqlStatement = addSortOps(sqlStatement, so, i == nSortOps-1)
		}
	}

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

// GetVisit queries the visit table for a given visit id. If the returned visit has ID = 0 then it is not in
// the database
func (s Storage) GetVisit(id int64) lister.Visit {
	var v lister.Visit
	sqlStatement := generateVisitSQL()
	// Add where clause by id
	sqlStatement = sqlStatement + `
		WHERE
			v.id=$1
	`
	row := s.db.QueryRow(sqlStatement, id)
	err := fillVisit(row, &v)
	if err != sql.ErrNoRows {
		checkAndPanic(err)
	}
	return v
}

// GetVisitsByRestaurantID gets all the visits for a given restaurant_id
func (s Storage) GetVisitsByRestaurantID(restaurantID int64) []lister.Visit {
	var allVisits []lister.Visit
	var v lister.Visit
	sqlStatement := generateVisitSQL()
	// Add where clause by id
	sqlStatement = sqlStatement + `
		WHERE
			v.restaurant_id=$1
	`
	dbRows, err := s.db.Query(sqlStatement, restaurantID)
	checkAndPanic(err)
	defer dbRows.Close()
	for dbRows.Next() {
		err = fillVisit(dbRows, &v)
		checkAndPanic(err)
		allVisits = append(allVisits, v)
	}
	err = dbRows.Err()
	checkAndPanic(err)
	return allVisits
}

// GetVisitUsersByVisitID queries the db for user for the given visit_id
func (s Storage) GetVisitUsersByVisitID(visitID int64) []lister.VisitUser {
	var allVisitUsers []lister.VisitUser
	var vu lister.VisitUser

	sqlStatement := `
		SELECT
			vu.id,			
			COALESCE(vu.rating, 0) as rating,
			vu.user_id,
			u.first_name,
			u.last_name
		FROM
			visit_user as vu
			inner join user as u on u.id = vu.user_id
		WHERE
			visit_id = $1
	`

	dbRows, err := s.db.Query(sqlStatement, visitID)
	checkAndPanic(err)
	defer dbRows.Close()
	for dbRows.Next() {
		err = dbRows.Scan(
			&vu.ID,
			&vu.Rating,
			&vu.User.ID,
			&vu.User.FirstName,
			&vu.User.LastName,
		)
		checkAndPanic(err)
		allVisitUsers = append(allVisitUsers, vu)
	}
	err = dbRows.Err()
	checkAndPanic(err)
	return allVisitUsers
}

// AddVisit adds the given visit to the sqlite database. Must call Commit() to commit transaction
func (s Storage) AddVisit(v adder.Visit) int64 {
	// We use case when to allow inserting nulls in the database
	sqlStatement := `
		INSERT INTO 
			visit(
				restaurant_id,
				visit_datetime,
				note
			)
		VALUES
			(
				$1,
				$2,
				CASE WHEN $3 == "" THEN NULL ELSE $3 END
			)
	`
	res, err := s.tx.Exec(sqlStatement,
		v.RestaurantID,
		v.VisitDateTime,
		v.Note,
	)
	checkAndPanic(err)
	lastID, err := res.LastInsertId()
	checkAndPanic(err)
	return lastID
}

// AddVisitUser adds the given visit to the sqlite database. Must call Commit() to commit transaction
func (s Storage) AddVisitUser(vu adder.VisitUser) int64 {
	// We use case when to allow inserting nulls in the database
	sqlStatement := `
		INSERT INTO 
			visit_user(
				visit_id,
				user_id,
				rating
			)
		VALUES
			(
				$1,
				$2,
				CASE WHEN $3 == 0 THEN NULL ELSE $3 END
			)
		ON CONFLICT(visit_id, user_id) DO UPDATE
		SET			
			visit_id = $1,
			user_id = $2,
			rating = CASE WHEN $3 == 0 THEN NULL ELSE $3 END
	`
	res, err := s.tx.Exec(sqlStatement,
		vu.VisitID,
		vu.UserID,
		vu.Rating,
	)
	checkAndPanic(err)
	lastID, err := res.LastInsertId()
	checkAndPanic(err)
	return lastID
}

// GetUser queries the user table for a given user id. If the returned user has ID = 0 then it is not in the db.
func (s Storage) GetUser(id int64) lister.User {
	var u lister.User
	sqlStatement := `
		SELECT 
			id,
			first_name,
			last_name,
			email
		FROM
			user 
		WHERE 
			id = $1
	`
	row := s.db.QueryRow(sqlStatement, id)
	err := row.Scan(
		&u.ID,
		&u.FirstName,
		&u.LastName,
		&u.Email,
	)
	if err != sql.ErrNoRows {
		checkAndPanic(err)
	}
	return u
}

// GetUserBy queries the user table for a given user by field and value. Do not pass field arguments from untrusted
// sources. If the returned user has ID = 0 then it is not in the db.
func (s Storage) GetUserBy(field string, value string) lister.User {
	var u lister.User
	sqlStatement := `
		SELECT 
			id,
			first_name,
			last_name,
			email
		FROM
			user
		WHERE
			%s = $1
	`
	row := s.db.QueryRow(fmt.Sprintf(sqlStatement, field), value)
	err := row.Scan(
		&u.ID,
		&u.FirstName,
		&u.LastName,
		&u.Email,
	)
	if err != sql.ErrNoRows {
		checkAndPanic(err)
	}
	return u
}

// GetUserAuthByEmail returns the password and remember hashes of a given email. If the returned user has ID = 0 then it
// is not in the db.
func (s Storage) GetUserAuthByEmail(email string) auther.User {
	var uh auther.User
	sqlStatement := `
		SELECT
			id,
			password_hash,
			COALESCE(remember_token, "")
		FROM
			user
		WHERE
			email = $1
	`
	row := s.db.QueryRow(sqlStatement, email)
	err := row.Scan(
		&uh.ID,
		&uh.PasswordHash,
		&uh.RememberToken,
	)
	if err != sql.ErrNoRows {
		checkAndPanic(err)
	}
	return uh
}

// GetUserAuthByID returns the password and remember hashes of a given email. If the returned user has ID = 0 then it
// is not in the db.
func (s Storage) GetUserAuthByID(id int64) auther.User {
	var uh auther.User
	sqlStatement := `
		SELECT
			id,
			password_hash,
			COALESCE(remember_token, "")
		FROM
			user
		WHERE
			id = $1
	`
	row := s.db.QueryRow(sqlStatement, id)
	err := row.Scan(
		&uh.ID,
		&uh.PasswordHash,
		&uh.RememberToken,
	)
	if err != sql.ErrNoRows {
		checkAndPanic(err)
	}
	return uh
}

// GetUserCount returns the number of users in the db.
func (s Storage) GetUserCount() int64 {
	var userCount int64
	sqlStatement := `
		SELECT 
			count(id)
		FROM
			user
	`
	row := s.db.QueryRow(sqlStatement)
	err := row.Scan(
		&userCount,
	)
	if err != sql.ErrNoRows {
		checkAndPanic(err)
	}
	return userCount
}

// UpdateUser updates a given user, returns the rows affected. Caller must call Commit() to commit the transaction
func (s Storage) UpdateUser(u updater.User) int64 {
	sqlStatement := `
		UPDATE
			user
		SET
			first_name = $1,
			last_name = $2,
			email = $3
		WHERE
			id = $4
	`
	res, err := s.tx.Exec(sqlStatement,
		u.FirstName,
		u.LastName,
		u.Email,
		u.ID,
	)
	checkAndPanic(err)
	rowsAffected, err := res.RowsAffected()
	checkAndPanic(err)
	return rowsAffected
}

// UpdateUserPassword updates the password of the user with the given id. Caller must call Commit() to commit the
// transaction
func (s Storage) UpdateUserPassword(id int64, newPasswordHash string) int64 {
	sqlStatement := `
		UPDATE
			user
		SET
			password_hash = $1
		WHERE
			id = $2
	`
	res, err := s.tx.Exec(sqlStatement,
		newPasswordHash,
		id,
	)
	checkAndPanic(err)
	rowsAffected, err := res.RowsAffected()
	checkAndPanic(err)
	return rowsAffected
}

// UpdateVisit updates a given visit, returns the rows affected. Caller must call Commit() to commit the
// transaction
func (s Storage) UpdateVisit(v updater.Visit) int64 {
	// We use case when to allow updating to nulls in the database
	sqlStatement := `
		UPDATE
			visit
		SET
			restaurant_id = $1,
			visit_datetime = $2,
			note = CASE WHEN $3 == "" THEN NULL ELSE $3 END
		WHERE
			id = $4
	`

	res, err := s.tx.Exec(sqlStatement,
		v.RestaurantID,
		v.VisitDateTime,
		v.Note,
		v.ID,
	)
	checkAndPanic(err)
	rowsAffected, err := res.RowsAffected()
	checkAndPanic(err)
	return rowsAffected
}

// UpdateVisitUser updates a given visit_user, returns the rows affected. Caller must call Commit() to commit the
// transaction
func (s Storage) UpdateVisitUser(vu updater.VisitUser) int64 {
	// We use case when to allow updating to nulls in the database
	sqlStatement := `
		UPDATE
			visit_user
		SET
			visit_id = $1,
			user_id = $2,
			rating = CASE WHEN $3 == "" THEN NULL ELSE $3 END
		WHERE
			id = $4
	`

	res, err := s.tx.Exec(sqlStatement,
		vu.VisitID,
		vu.UserID,
		vu.Rating,
		vu.ID,
	)
	checkAndPanic(err)
	rowsAffected, err := res.RowsAffected()
	checkAndPanic(err)
	return rowsAffected
}

// RemoveVisit deletes a given visit and returns the number of rows affected. Caller must call Commit() to commit the
// transaction
func (s Storage) RemoveVisit(visitID int64) int64 {
	return s.removeRow("visit", visitID)
}

// RemoveVisitUser deletes a given visit_user and returns the number of rows affected. Caller must call Commit() to commit the
// transaction
func (s Storage) RemoveVisitUser(visitUserID int64) int64 {
	return s.removeRow("visit_user", visitUserID)
}

// AddUser adds a given user to the database and returns the new user id. Caller must call Commit() to commit the
// transaction.
func (s Storage) AddUser(u adder.User) int64 {
	// We use case when to allow inserting nulls in the database
	sqlStatement := `
		INSERT INTO 
			user(
				first_name,
				last_name,
				email,
				password_hash
			)
		VALUES
			(
				$1,
				$2,
				$3,
				$4
			)
	`
	res, err := s.tx.Exec(sqlStatement,
		u.FirstName,
		u.LastName,
		u.Email,
		u.PasswordHash,
	)
	checkAndPanic(err)
	lastID, err := res.LastInsertId()
	checkAndPanic(err)
	return lastID
}

// UpdateUserRememberToken updates a user's remember_hash and then returns the number of rows affected. Caller must call
// Commit() to commit the transaction.
func (s Storage) UpdateUserRememberToken(u auther.User) int64 {
	// We use case when to allow updating to nulls in the database
	sqlStatement := `
		UPDATE
			user
		SET
			remember_token = $1
		WHERE
			id = $2
	`

	res, err := s.tx.Exec(sqlStatement,
		u.RememberToken,
		u.ID,
	)
	checkAndPanic(err)
	rowsAffected, err := res.RowsAffected()
	checkAndPanic(err)
	return rowsAffected
}

// GetRestaurantAvgRatingByUser gets a given restaurants average rating group by user. If the returned value for a user
// is 0 then the restaurant has no ratings for that user.
func (s Storage) GetRestaurantAvgRatingByUser(restaurantID int64) []lister.AvgUserRating {
	var allRatings []lister.AvgUserRating
	var ar lister.AvgUserRating
	sqlStatement := `
		SELECT
			user_id,
			u.first_name,
			u.last_name,
			coalesce(avg(rating), 0) as avg_rating
		FROM
			visit_user as vu
			left join visit as v on v.id = vu.visit_id
			left join user as u on u.id = vu.user_id
		WHERE
			v.restaurant_id = $1
		GROUP BY
			u.first_name,
			u.last_name
	`
	dbRows, err := s.db.Query(sqlStatement, restaurantID)
	checkAndPanic(err)
	defer dbRows.Close()
	for dbRows.Next() {
		err = dbRows.Scan(
			&ar.ID,
			&ar.FirstName,
			&ar.LastName,
			&ar.AvgRating,
		)
		checkAndPanic(err)
		allRatings = append(allRatings, ar)
	}
	err = dbRows.Err()
	checkAndPanic(err)
	return allRatings
}

func checkAndPanic(err error) {
	if err != nil {
		log.Panicln(err)
	}
}
