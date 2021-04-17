package adder

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/kelvinatorr/restaurant-tracker/internal/auther"
	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
	"github.com/kelvinatorr/restaurant-tracker/internal/mapper"
)

// ErrDuplicate is used when a resturant already exists.
type ErrDuplicate struct {
	msg string
}

func (m *ErrDuplicate) Error() string {
	return m.msg
}

// Service provides adding operations.
type Service interface {
	AddRestaurant(Restaurant) (int64, error)
	AddVisit(Visit) (int64, error)
	AddUser(User) (int64, error)
}

// Repository provides access to restaurant repository.
type Repository interface {
	Begin()
	Commit()
	Rollback()
	// AddRestaurant saves a given restaurant to the repository.
	AddRestaurant(Restaurant) int64
	// IsDuplicateRestaurant checks if a restaurant with the same name in the same city and state is already in the db
	IsDuplicateRestaurant(Restaurant) bool
	// GetCityIDByNameAndState gets the id of a city with the same name and state from the database
	GetCityIDByNameAndState(string, string) int64
	AddCity(string, string) int64
	AddGmapsPlace(GmapsPlace) int64
	AddVisit(Visit) int64
	AddVisitUser(VisitUser) int64
	GetRestaurant(int64) lister.Restaurant
	GetUser(int64) lister.User
	GetUserBy(string, string) lister.User
	AddUser(User) int64
}

type Map interface {
	PlaceDetails(string) (mapper.PlaceDetail, error)
}

type service struct {
	r Repository
	m Map
}

func (s *service) AddRestaurant(r Restaurant) (int64, error) {
	err := checkRestaurantData(r)
	if err != nil {
		return 0, err
	}

	// Check that there isn't a duplicate restaurant with the same name in the same city, state already
	if s.r.IsDuplicateRestaurant(r) {
		errorMsg := fmt.Sprintf("%s in %s, %s is already in the database.", r.Name, r.CityState.Name, r.CityState.State)
		return 0, &ErrDuplicate{msg: errorMsg}
	}
	// Check if the city and state is already in the database, If it is, get the city id
	cityID := s.r.GetCityIDByNameAndState(r.CityState.Name, r.CityState.State)
	s.r.Begin()
	// Defer rollback just in case there is a problem.
	defer s.r.Rollback()
	if cityID == 0 {
		// If not, then add it to the city table and get the city id back
		log.Println(fmt.Sprintf("%s, %s not found, adding...", r.CityState.Name, r.CityState.State))
		cityID = s.r.AddCity(r.CityState.Name, r.CityState.State)
	}
	log.Println(fmt.Sprintf("%s, %s has cityID %d", r.CityState.Name, r.CityState.State, cityID))
	// Add the city id to the restaurant object
	r.CityID = cityID

	var newRestaurantID int64
	// Only add gmaps place if we actually have it.
	if r.GmapsPlace.PlaceID != "" {
		// Get the Google Maps Details
		pd, err := s.m.PlaceDetails(r.GmapsPlace.PlaceID)
		if err != nil {
			return 0, err
		}
		// Update the values in the restaurant struct.
		r.Latitude = pd.Result.Geometry.Location.Lat
		r.Longitude = pd.Result.Geometry.Location.Lng
		r.Zipcode = pd.Result.ZipCode
		r.Address = pd.Result.Address

		r.GmapsPlace.BusinessStatus = pd.Result.BusinessStatus
		r.GmapsPlace.FormattedPhoneNumber = pd.Result.FormattedPhoneNumber
		r.GmapsPlace.Name = pd.Result.Name
		r.GmapsPlace.PriceLevel = pd.Result.PriceLevel
		r.GmapsPlace.Rating = pd.Result.Rating
		r.GmapsPlace.URL = pd.Result.URL
		r.GmapsPlace.UserRatingsTotal = pd.Result.UserRatingsTotal
		r.GmapsPlace.UTCOffset = pd.Result.UTCOffset
		r.GmapsPlace.Website = pd.Result.Website

		// First add the restaurant
		newRestaurantID = s.r.AddRestaurant(r)
		// Set the restaurant id on the GmapsPlace for foreign key relationships
		r.GmapsPlace.RestaurantID = newRestaurantID
		// Finally add the GmapsPlace
		s.r.AddGmapsPlace(r.GmapsPlace)
	} else {
		// Just add the restaurant because there is no Gmaps Place data
		newRestaurantID = s.r.AddRestaurant(r)
	}

	s.r.Commit()
	return newRestaurantID, nil
}

func (s *service) AddVisit(v Visit) (int64, error) {
	// Check that the restaurant id is valid
	r := s.r.GetRestaurant(v.RestaurantID)
	if r.ID == 0 {
		errorMsg := fmt.Sprintf("There is no restaurant with id: %d.", v.RestaurantID)
		return 0, errors.New(errorMsg)
	}
	// Check that the user id is valid and that there is only 1 entry per user id
	userIDs := make(map[int64]bool)
	for _, vu := range v.VisitUsers {
		u := s.r.GetUser(vu.UserID)
		if u.ID == 0 {
			errorMsg := fmt.Sprintf("There is no user with id: %d.", vu.UserID)
			return 0, errors.New(errorMsg)
		}
		if _, ok := userIDs[vu.UserID]; ok {
			errorMsg := fmt.Sprintf("The data has multiple users with id: %d.", vu.UserID)
			return 0, errors.New(errorMsg)
		}
		userIDs[vu.UserID] = true
	}

	visitDateTime, err := time.Parse("2006-01-02", v.VisitDateTime)
	if err != nil {
		log.Println(err)
		return 0, fmt.Errorf("Cannot format %s as date", v.VisitDateTime)
	}
	v.VisitDateTime = visitDateTime.Format(time.RFC3339)

	s.r.Begin()
	// Defer rollback just in case there is a problem.
	defer s.r.Rollback()

	visitID := s.r.AddVisit(v)
	for i := range v.VisitUsers {
		v.VisitUsers[i].VisitID = visitID
		s.r.AddVisitUser(v.VisitUsers[i])
	}

	s.r.Commit()

	return visitID, nil
}

func checkRestaurantData(r Restaurant) error {
	// Check that Name is not null
	if r.Name == "" {
		return errors.New("A name is required")
	}

	// Check that CityState is not null
	if r.CityState.Name == "" || r.CityState.State == "" {
		return fmt.Errorf("You must provide a city and state for %s", r.Name)
	}

	// Check that Cuisine is not null
	if r.Cuisine == "" {
		return errors.New("You must provide a cuisine")
	}

	return nil
}

func (s *service) AddUser(u User) (int64, error) {
	if err := checkUserData(u); err != nil {
		return 0, err
	}
	// Lower case it to normalize it.
	u.Email = strings.ToLower(u.Email)
	// Check email is not duplicate
	if existingUser := s.r.GetUserBy("email", u.Email); existingUser.ID != 0 {
		return 0, errors.New("This user already exists")
	}

	// Hash password using the auther service
	passwordHash, err := auther.HashPassword(u.Password)
	if err != nil {
		return 0, err
	}
	u.PasswordHash = passwordHash
	// Clear password so it isn't inadvertently logged
	u.Password = ""

	// Add the user
	s.r.Begin()
	// Defer rollback just in case there is a problem.
	defer s.r.Rollback()
	newUserID := s.r.AddUser(u)
	s.r.Commit()

	return newUserID, nil
}

func checkUserData(u User) error {
	// Check all fields are not empty
	if u.FirstName == "" || u.LastName == "" {
		return errors.New("First name and last name are required")
	}
	if u.Email == "" {
		return errors.New("An email address is required")
	}
	if u.Password == "" || u.RepeatPassword == "" {
		return errors.New("A password and repeatPassword are required")
	}

	// Check passwords are the same
	if u.Password != u.RepeatPassword {
		return errors.New("Passwords do not match")
	}

	// Check email is valid
	// Good enough validation: https://www.regextester.com/99632
	match, err := regexp.MatchString("[^@]+@[^\\.]+\\..+", u.Email)
	if err != nil {
		return err
	} else if !match {
		return errors.New("Invalid email address")
	}

	return nil
}

// NewService creates an adding service with the necessary dependencies
func NewService(r Repository, m Map) Service {
	return &service{r, m}
}
