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
	GetVisit(int64) (Visit, error)
	GetVisitsByRestaurantID(int64) []Visit
}

// Repository provides access to restaurant repository.
type Repository interface {
	// GetRestaurant gets a given restaurant to the repository.
	GetRestaurant(int64) Restaurant
	GetRestaurants() []Restaurant
	GetVisit(int64) Visit
	GetVisitUsersByVisitID(int64) []VisitUser
	GetVisitsByRestaurantID(int64) []Visit
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

// GetVisit returns a visit with the given id
func (s service) GetVisit(id int64) (Visit, error) {
	var err error
	v := s.r.GetVisit(id)
	if v.ID == 0 {
		err = &ErrDoesNotExist{fmt.Sprintf("No visit with id: %d", id)}
	} else {
		// Get the users who were in this visit.
		v.VisitUsers = s.r.GetVisitUsersByVisitID(v.ID)
	}
	return v, err
}

func (s service) GetVisitsByRestaurantID(restaurantID int64) []Visit {
	allVisits := s.r.GetVisitsByRestaurantID(restaurantID)
	for i, v := range allVisits {
		// For each visit get the users who were there and their rating.
		allVisits[i].VisitUsers = s.r.GetVisitUsersByVisitID(v.ID)
	}
	return allVisits
}

// NewService returns a new lister.service
func NewService(r Repository) Service {
	return service{r}
}
