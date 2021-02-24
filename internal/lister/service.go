package lister

import (
	"fmt"
	"net/url"
	"time"
)

// ErrDoesNotExist is used when a resturant does not exist in the repository
type ErrDoesNotExist struct {
	msg string
}

// Field is a repository field
type Field struct {
	Name string
	Type string
}

func (m *ErrDoesNotExist) Error() string {
	return m.msg
}

// Service provides listing operations.
type Service interface {
	GetRestaurant(int64) (Restaurant, error)
	GetRestaurants(url.Values) ([]Restaurant, error)
	GetVisit(int64) (Visit, error)
	GetVisitsByRestaurantID(int64) []Visit
	GetUserCount() int64
	GetUserByID(int64) User
	GetFilterOptions() FilterOptions
}

// Repository provides access to restaurant repository.
type Repository interface {
	// GetRestaurant gets a given restaurant to the repository.
	GetRestaurant(int64) Restaurant
	GetRestaurants([]SortOperation, []FilterOperation) []Restaurant
	GetVisit(int64) Visit
	GetVisitUsersByVisitID(int64) []VisitUser
	GetVisitsByRestaurantID(int64) []Visit
	GetUserCount() int64
	GetUser(int64) User
	GetRestaurantAvgRatingByUser(int64) []AvgUserRating
	GetDistinct(string, string) []string
	RestaurantSortFields() map[string]string
	RestaurantFilterFields() map[string]Field
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
func (s service) GetRestaurants(qp url.Values) ([]Restaurant, error) {
	var rs []Restaurant
	sops, err := s.checkSort("restaurant", qp)
	if err != nil {
		return rs, err
	}

	fops, err := s.checkFilter("restaurant", qp)
	if err != nil {
		return rs, err
	}

	rs = s.r.GetRestaurants(sops, fops)
	// Get ratings for each restaurant
	for i, r := range rs {
		rs[i].AvgUserRatings = s.r.GetRestaurantAvgRatingByUser(r.ID)
		var lastVisitHumanDate string = ""
		if r.LastVisitDatetime != "" {
			lastVisitDate, err := time.Parse(time.RFC3339, r.LastVisitDatetime)
			if err != nil {
				return rs, err
			}
			lastVisitHumanDate = lastVisitDate.Format("January 2, 2006")
		}
		// Add search value property
		rs[i].SearchValue = fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s", r.Name, r.Cuisine, r.CityState.Name, r.CityState.State,
			r.Note, r.LastVisitDatetime, lastVisitHumanDate)
	}
	return rs, nil
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

// GetFilterOptions returns a Filter
func (s service) GetFilterOptions() FilterOptions {
	return FilterOptions{
		Cuisine: s.r.GetDistinct("cuisine", "restaurant"),
		City:    s.r.GetDistinct("name", "city"),
		State:   s.r.GetDistinct("state", "city"),
	}
}

// NewService returns a new lister.service
func NewService(r Repository) Service {
	return service{r}
}
