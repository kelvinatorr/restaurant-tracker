package updater

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
	"github.com/kelvinatorr/restaurant-tracker/internal/mapper"

	"github.com/kelvinatorr/restaurant-tracker/internal/adder"
	"github.com/kelvinatorr/restaurant-tracker/internal/auther"
)

// Service provides listing operations.
type Service interface {
	UpdateRestaurant(Restaurant) (int64, error)
	UpdateVisit(Visit) (int64, error)
	UpdateUser(User) (int64, error)
	UpdateUserPassword(auther.UserChangePassword) (int64, error)
}

// Repository provides access to restaurant repository.
type Repository interface {
	Begin()
	Commit()
	Rollback()
	// UpdateRestaurant updates a given restaurant in the repository.
	UpdateRestaurant(Restaurant) int64
	GetCityIDByNameAndState(string, string) int64
	AddCity(string, string) int64
	AddGmapsPlace(adder.GmapsPlace) int64
	UpdateGmapsPlace(GmapsPlace) int64
	UpdateVisit(Visit) int64
	UpdateVisitUser(VisitUser) int64
	GetRestaurant(int64) lister.Restaurant
	GetUser(int64) lister.User
	AddVisitUser(adder.VisitUser) int64
	GetVisitUsersByVisitID(int64) []lister.VisitUser
	RemoveVisitUser(int64) int64
	GetUserBy(string, string) lister.User
	UpdateUser(User) int64
	UpdateUserPassword(int64, string) int64
	GetUserAuthByID(int64) auther.User
}

type Map interface {
	PlaceDetails(string) (mapper.PlaceDetail, error)
}

type service struct {
	r Repository
	m Map
}

func (s service) UpdateRestaurant(r Restaurant) (int64, error) {
	err := checkRestaurantData(r)
	if err != nil {
		return 0, err
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

	// This restaurant did not have a GmapsPlace, but now has 1, so we insert it and get the id back.
	if r.GmapsPlace.ID == 0 && r.GmapsPlace.PlaceID != "" {
		// Get the Place Details for this PlaceID
		pd, err := s.m.PlaceDetails(r.GmapsPlace.PlaceID)
		if err != nil {
			return 0, err
		}
		// Update the values in the restaurant struct.
		r.Latitude = pd.Result.Geometry.Location.Lat
		r.Longitude = pd.Result.Geometry.Location.Lng
		r.Zipcode = pd.Result.ZipCode
		r.Address = pd.Result.Address

		log.Printf("Inserting new GmapsPlace with PlaceID %s\n", r.GmapsPlace.PlaceID)
		// Create a new GmapsPlace for adding
		newGmapsPlace := adder.GmapsPlace{
			PlaceID:              pd.Result.PlaceID,
			BusinessStatus:       pd.Result.BusinessStatus,
			FormattedPhoneNumber: pd.Result.FormattedPhoneNumber,
			Name:                 pd.Result.Name,
			PriceLevel:           pd.Result.PriceLevel,
			Rating:               pd.Result.Rating,
			URL:                  pd.Result.URL,
			UserRatingsTotal:     pd.Result.UserRatingsTotal,
			UTCOffset:            pd.Result.UTCOffset,
			Website:              pd.Result.Website,
			RestaurantID:         r.ID,
		}
		// No need to set LastUpdated because it has a default to current timestamp in the repository
		// Add the GmapsPlace
		s.r.AddGmapsPlace(newGmapsPlace)
	} else if r.GmapsPlace.ID != 0 {
		// This restaurant already has a GmapsPlace Record so we just update it.
		log.Printf("Updating GmapsPlace id: %d.\n", r.GmapsPlace.ID)
		// Make the gmaps foreign key the restaurant id
		r.GmapsPlace.RestaurantID = r.ID
		// Parse the last updated date into the proper full format
		lastUpdated, err := time.Parse("2006-01-02", r.GmapsPlace.LastUpdated)
		if err != nil {
			return 0, err
		}
		r.GmapsPlace.LastUpdated = lastUpdated.Format(time.RFC3339)
		gmapsPlaceRecordsAffected := s.r.UpdateGmapsPlace(r.GmapsPlace)
		log.Printf("%d GmapsPlace records affected.\n", gmapsPlaceRecordsAffected)
	} else {
		log.Printf("Restaurant id: %d has no GmapsPlace record and update data has no GmapsPlace data.", r.ID)
	}

	// Update the restaurant.
	recordsAffected := s.r.UpdateRestaurant(r)
	if recordsAffected == 0 {
		// Rollback should occur because of the defer.
		return 0, fmt.Errorf("Restaurant id: %d was not found", r.ID)
	}

	s.r.Commit()

	return recordsAffected, nil
}

func (s service) UpdateVisit(v Visit) (int64, error) {
	// Check that the restaurant id is valid
	r := s.r.GetRestaurant(v.RestaurantID)
	if r.ID == 0 {
		errorMsg := fmt.Sprintf("There is no restaurant with id: %d", v.RestaurantID)
		return 0, errors.New(errorMsg)
	}
	// Check that the user id is valid and that there is only 1 entry per user id
	userIDs := make(map[int64]bool)
	for i, vu := range v.VisitUsers {
		u := s.r.GetUser(vu.UserID)
		if u.ID == 0 {
			errorMsg := fmt.Sprintf("There is no user with id: %d", vu.UserID)
			return 0, errors.New(errorMsg)
		}
		if _, ok := userIDs[vu.UserID]; ok {
			errorMsg := fmt.Sprintf("The data has multiple users with id: %d", vu.UserID)
			return 0, errors.New(errorMsg)
		}
		userIDs[vu.UserID] = true
		// Add the visit id to each VisitUser
		v.VisitUsers[i].VisitID = v.ID
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

	var visitUserRecordsAffected int64
	visitRecordsAffected := s.r.UpdateVisit(v)
	log.Printf("%d Visit records affected.\n", visitRecordsAffected)
	// Only update if the visit actually exists.
	if visitRecordsAffected > 0 {
		// Get the saved VisitUsers so we can remove anything that's not in this update.
		savedVisitUsers := s.r.GetVisitUsersByVisitID(v.ID)
		// Convert it to a map of ids
		visitUsersMap := make(map[int64]bool)
		for _, vu := range savedVisitUsers {
			visitUsersMap[vu.ID] = false
		}

		for _, vu := range v.VisitUsers {
			if vu.ID != 0 {
				visitUserRecordsAffected = visitUserRecordsAffected + s.r.UpdateVisitUser(vu)
				// Set this VisitUser to True in the map so it doesn't get deleted.
				visitUsersMap[vu.ID] = true
			} else {
				newVisit := adder.VisitUser{
					VisitID: vu.VisitID,
					UserID:  vu.UserID,
					Rating:  vu.Rating,
				}
				newVisitUserID := s.r.AddVisitUser(newVisit)
				log.Printf("Added User id: %d to Visit id: %d. New VisitUser id: %d", vu.UserID, vu.VisitID,
					newVisitUserID)
			}
		}

		// Now loop through the saved VisitUsers and delete anything we didn't see in this update. The user was removed
		// from the visit.
		for k, val := range visitUsersMap {
			if !val {
				s.r.RemoveVisitUser(k)
				log.Printf("Removed VisitUser id: %d from Visit id: %d", k, v.ID)
			}
		}
	}

	s.r.Commit()

	return visitRecordsAffected + visitUserRecordsAffected, nil
}

func (s service) UpdateUser(u User) (int64, error) {
	// Check that all the properties have values
	if u.FirstName == "" || u.LastName == "" {
		return 0, errors.New("First name and last name are required")
	}
	if u.Email == "" {
		return 0, errors.New("An email address is required")
	}

	// Check that the email is valid
	// Good enough validation: https://www.regextester.com/99632
	match, err := regexp.MatchString("[^@]+@[^\\.]+\\..+", u.Email)
	if err != nil {
		return 0, err
	} else if !match {
		return 0, errors.New("Invalid email address")
	}
	// Check that the email is not already in the database used by a different user
	existingUser := s.r.GetUserBy("email", u.Email)
	if existingUser.ID != 0 && existingUser.ID != u.ID {
		return 0, errors.New("A user with this email address already exists")
	}
	s.r.Begin()
	// Defer rollback just in case there is a problem.
	defer s.r.Rollback()
	// Update the user
	recordsAffected := s.r.UpdateUser(u)
	s.r.Commit()
	return recordsAffected, nil
}

func (s service) UpdateUserPassword(u auther.UserChangePassword) (int64, error) {
	// Check that the fields are populated
	if u.CurrentPassword == "" || u.NewPassword == "" || u.RepeatNewPassword == "" {
		return 0, errors.New("All the fields are required")
	}
	// Check that the new password and repeat password matches
	if u.NewPassword != u.RepeatNewPassword {
		return 0, errors.New("Passwords don't match")
	}

	// Check that the new password is not the same as the current password
	if u.NewPassword == u.CurrentPassword {
		return 0, errors.New("You can't set your password to be what you think your current one already is")
	}

	// Check that the current password is correct using the auther service
	// Check this id exists
	foundUser := s.r.GetUserAuthByID(u.ID)
	if foundUser.ID == 0 {
		return 0, errors.New("There is no user with this id")
	}

	err := auther.CheckPassword(foundUser.PasswordHash, u.CurrentPassword)
	if err != nil {
		return 0, errors.New("Wrong current password")
	}

	// Hash password using the auther service
	passwordHash, err := auther.HashPassword(u.NewPassword)
	if err != nil {
		return 0, err
	}

	// Clear password so it isn't inadvertently logged
	u.NewPassword = ""
	u.RepeatNewPassword = ""

	// Update the user's password
	s.r.Begin()
	// Defer rollback just in case there is a problem.
	defer s.r.Rollback()
	recordsAffected := s.r.UpdateUserPassword(u.ID, passwordHash)
	s.r.Commit()

	return recordsAffected, nil
}

func checkRestaurantData(r Restaurant) error {
	if r.ID == 0 {
		return errors.New("Update ID cannot be 0")
	}

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

// NewService returns a new updater.service
func NewService(r Repository, m Map) Service {
	return service{r, m}
}
