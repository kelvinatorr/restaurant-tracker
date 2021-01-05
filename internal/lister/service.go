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
	GetUserCount() int64
	GetUserByID(int64) User
}

// Repository provides access to restaurant repository.
type Repository interface {
	// GetRestaurant gets a given restaurant to the repository.
	GetRestaurant(int64) Restaurant
	GetRestaurants() []Restaurant
	GetVisit(int64) Visit
	GetVisitUsersByVisitID(int64) []VisitUser
	GetVisitsByRestaurantID(int64) []Visit
	GetUserCount() int64
	GetUser(int64) User
	GetRestaurantAvgRating(int64) float32
	GetRestaurantAvgRatingByUser(int64) []AvgUserRating
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
	rs := s.r.GetRestaurants()
	// Get ratings for each restaurant
	for i, r := range rs {
		rs[i].AvgUserRatings = s.r.GetRestaurantAvgRatingByUser(r.ID)
		// Get the average rating overall
		rs[i].AvgRating = s.r.GetRestaurantAvgRating(r.ID)
	}
	return rs
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

// GetUserCount returns the number of users in the repository.
func (s service) GetUserCount() int64 {
	return s.r.GetUserCount()
}

// GetUserByID returns the User for a given id
func (s service) GetUserByID(id int64) User {
	return s.r.GetUser(id)
}

// NewService returns a new lister.service
func NewService(r Repository) Service {
	return service{r}
}
