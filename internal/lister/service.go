package lister

import (
	"fmt"
)

// ErrDoesNotExist is used when a resturant does not exist in the repository
type ErrDoesNotExist struct {
	msg string
}

func (m *ErrDoesNotExist) Error() string {
	return m.msg
}

// Service provides listing operations.
type Service interface {
	GetRestaurant(int64) (Restaurant, error)
	GetRestaurants() []Restaurant
}

// Repository provides access to restaurant repository.
type Repository interface {
	// GetRestaurant gets a given restaurant to the repository.
	GetRestaurant(int64) Restaurant
	GetRestaurants() []Restaurant
}

type service struct {
	r Repository
}

// GetRestaurant returns a restaurant with the given id
func (s service) GetRestaurant(id int64) (Restaurant, error) {
	var err error
	r := s.r.GetRestaurant(id)
	if r.ID == 0 {
		err = &ErrDoesNotExist{fmt.Sprintf("No restaurant with id: %d", id)}
	}
	return r, err
}

// GetRestaurants returns all the restaurants in the storage
func (s service) GetRestaurants() []Restaurant {
	return s.r.GetRestaurants()
}

// NewService returns a new lister.service
func NewService(r Repository) Service {
	return service{r}
}
