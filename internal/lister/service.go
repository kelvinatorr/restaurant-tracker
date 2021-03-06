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
	GetVisit(int64, int64) (Visit, error)
	GetVisitsByRestaurantID(int64, url.Values) ([]Visit, error)
	GetUserCount() int64
	GetUserByID(int64) User
	GetFilterOptions(url.Values) FilterOptions
	GetFilterParam(string, url.Values) FilterOperation
	GetSortParam(string, url.Values) SortOperation
	GetUsers() []User
	GetDistinct(string, string) []string
}

// Repository provides access to restaurant repository.
type Repository interface {
	// GetRestaurant gets a given restaurant to the repository.
	GetRestaurant(int64) Restaurant
	GetRestaurants([]SortOperation, []FilterOperation) []Restaurant
	GetVisit(int64, int64) Visit
	GetVisitUsersByVisitID(int64) []VisitUser
	GetVisitsByRestaurantID(int64, []SortOperation) []Visit
	GetUserCount() int64
	GetUser(int64) User
	GetRestaurantAvgRatingByUser(int64) []AvgUserRating
	GetDistinct(string, string) []string
	RestaurantSortFields() map[string]string
	RestaurantFilterFields() map[string]Field
	VisitSortFields() map[string]string
	GetUsers() []User
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

	dateFormat := "2006-01-02"
	if r.LastVisitDatetime != "" {
		lastVisitDate, err := time.Parse(time.RFC3339, r.LastVisitDatetime)
		if err != nil {
			return r, err
		}
		r.LastVisitDatetime = lastVisitDate.Format(dateFormat)
	}
	if r.GmapsPlace.LastUpdated != "" {
		lastUpdated, err := time.Parse(time.RFC3339, r.GmapsPlace.LastUpdated)
		if err != nil {
			return r, err
		}
		r.GmapsPlace.LastUpdated = lastUpdated.Format(dateFormat)
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
	for i, r := range rs {
		// // Get ratings for each restaurant
		rs[i].AvgUserRatings = s.r.GetRestaurantAvgRatingByUser(r.ID)
		var lastVisitHumanDate string = ""
		if r.LastVisitDatetime != "" {
			lastVisitDate, err := time.Parse(time.RFC3339, r.LastVisitDatetime)
			if err != nil {
				return rs, err
			}
			lastVisitHumanDate = lastVisitDate.Format("January 2, 2006")

			// Format the last visit to just the date
			rs[i].LastVisitDatetime = lastVisitDate.Format("2006-01-02")
		}
		// Add search value property
		rs[i].SearchValue = fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s", r.Name, r.Cuisine, r.CityState.Name, r.CityState.State,
			r.Note, r.LastVisitDatetime, lastVisitHumanDate)
	}
	return rs, nil
}

// GetVisit returns a visit with the given id and restaurant id
func (s service) GetVisit(id int64, resID int64) (Visit, error) {
	var err error
	v := s.r.GetVisit(id, resID)
	if v.ID == 0 {
		err = &ErrDoesNotExist{fmt.Sprintf("No visit with id: %d for restaurant: %d", id, resID)}
		return v, err
	} else {
		// Get the users who were in this visit.
		v.VisitUsers = s.r.GetVisitUsersByVisitID(v.ID)
	}
	dateFormat := "2006-01-02"
	visitDateTime, err := time.Parse(time.RFC3339, v.VisitDateTime)
	if err != nil {
		return v, err
	}
	v.VisitDateTime = visitDateTime.Format(dateFormat)

	return v, err
}

func (s service) GetVisitsByRestaurantID(restaurantID int64, qp url.Values) ([]Visit, error) {
	var allVisits []Visit
	sops, err := s.checkSort("visit", qp)
	if err != nil {
		return allVisits, err
	}

	allVisits = s.r.GetVisitsByRestaurantID(restaurantID, sops)
	for i, v := range allVisits {
		// For each visit get the users who were there and their rating.
		allVisits[i].VisitUsers = s.r.GetVisitUsersByVisitID(v.ID)

		// Format the last visit to just the date
		visitDateTime, err := time.Parse(time.RFC3339, v.VisitDateTime)
		if err != nil {
			return allVisits, err
		}
		allVisits[i].VisitDateTime = visitDateTime.Format("2006-01-02")
	}
	return allVisits, nil
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
func (s service) GetFilterOptions(qp url.Values) FilterOptions {
	return FilterOptions{
		Cuisine: generateFilterOptions(s.r.GetDistinct("cuisine", "restaurant"), s.GetFilterParam("cuisine", qp).Value),
		City:    generateFilterOptions(s.r.GetDistinct("name", "city"), s.GetFilterParam("city", qp).Value),
		State:   generateFilterOptions(s.r.GetDistinct("state", "city"), s.GetFilterParam("state", qp).Value),
	}
}

// GetUsers gets all the users in storage
func (s service) GetUsers() []User {
	return s.r.GetUsers()
}

func (s service) GetDistinct(field string, obj string) []string {
	return s.r.GetDistinct(field, obj)
}

func generateFilterOptions(distinctSlice []string, selectedValue string) []FilterOption {
	var cuisine []FilterOption
	for _, o := range distinctSlice {
		selected := o == selectedValue
		cuisine = append(cuisine, FilterOption{Value: o, Selected: selected})
	}
	return cuisine
}

// NewService returns a new lister.service
func NewService(r Repository) Service {
	return service{r}
}
