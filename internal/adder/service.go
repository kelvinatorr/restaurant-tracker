package adder

import (
	"errors"
	"fmt"
	"log"

	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
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
}

type service struct {
	r Repository
}

func (s *service) AddRestaurant(r Restaurant) (int64, error) {
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
	// Add the restaurant
	newRestaurantID := s.r.AddRestaurant(r)

	// Only add gmaps place if we actually have it.
	if r.GmapsPlace.PlaceID != "" {
		r.GmapsPlace.RestaurantID = newRestaurantID
		// Add the GmapsPlace
		s.r.AddGmapsPlace(r.GmapsPlace)
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

// NewService creates an adding service with the necessary dependencies
func NewService(r Repository) Service {
	return &service{r}
}
