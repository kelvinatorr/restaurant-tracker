package adder

import (
	"fmt"
	"log"
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
}

// Repository provides access to restaurant repository.
type Repository interface {
	// AddRestaurant saves a given restaurant to the repository.
	AddRestaurant(Restaurant) int64
	// IsDuplicateRestaurant checks if a restaurant with the same name in the same city and state is already in the db
	IsDuplicateRestaurant(Restaurant) bool
	// GetCityIDByNameAndState gets the id of a city with the same name and state from the database
	GetCityIDByNameAndState(Restaurant) int64
	AddCity(Restaurant) int64
}

type service struct {
	r Repository
}

func (s *service) AddRestaurant(r Restaurant) (int64, error) {
	fmt.Println(r.Name)
	// Check that there isn't a duplicate restaurant with the same name in the same city, state already
	if s.r.IsDuplicateRestaurant(r) {
		errorMsg := fmt.Sprintf("%s in %s, %s is already in the database.", r.Name, r.City, r.State)
		return 0, &ErrDuplicate{msg: errorMsg}
	}
	// Check if the city and state is already in the database, If it is, get the city id
	cityID := s.r.GetCityIDByNameAndState(r)
	if cityID == 0 {
		// If not, then add it to the city table and get the city id back
		log.Println(fmt.Sprintf("%s, %s not found, adding...", r.City, r.State))
		cityID = s.r.AddCity(r)
	}
	log.Println(fmt.Sprintf("%s, %s has cityID %d", r.City, r.State, cityID))
	// Add the city id to the restaurant object
	r.CityID = cityID
	// TODO: Get the gmaps place details
	// Add the restaurant
	newRestaurantID := s.r.AddRestaurant(r)
	return newRestaurantID, nil
}

// NewService creates an adding service with the necessary dependencies
func NewService(r Repository) Service {
	return &service{r}
}
