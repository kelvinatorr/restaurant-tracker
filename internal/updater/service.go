package updater

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/kelvinatorr/restaurant-tracker/internal/lister"

	"github.com/kelvinatorr/restaurant-tracker/internal/adder"
)

// Service provides listing operations.
type Service interface {
	UpdateRestaurant(Restaurant) (int64, error)
	UpdateVisit(Visit) (int64, error)
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
}

type service struct {
	r Repository
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
	// Update the restaurant.
	recordsAffected := s.r.UpdateRestaurant(r)
	if recordsAffected == 0 {
		// Rollback should occur because of the defer.
		return 0, fmt.Errorf("Restaurant id: %d was not found", r.ID)
	}

	// This restaurant did not have a GmapsPlace, but now has 1, so we insert it and get the id back.
	if r.GmapsPlace.ID == 0 && r.GmapsPlace.PlaceID != "" {
		log.Printf("Inserting new GmapsPlace with PlaceID %s\n", r.GmapsPlace.PlaceID)
		// Create a new GmapsPlace for adding
		newGmapsPlace := adder.GmapsPlace{
			PlaceID:              r.GmapsPlace.PlaceID,
			BusinessStatus:       r.GmapsPlace.BusinessStatus,
			FormattedPhoneNumber: r.GmapsPlace.FormattedPhoneNumber,
			Name:                 r.GmapsPlace.Name,
			PriceLevel:           r.GmapsPlace.PriceLevel,
			Rating:               r.GmapsPlace.Rating,
			URL:                  r.GmapsPlace.URL,
			UserRatingsTotal:     r.GmapsPlace.UserRatingsTotal,
			UTCOffset:            r.GmapsPlace.UTCOffset,
			Website:              r.GmapsPlace.Website,
			RestaurantID:         r.ID,
		}
		// Add the GmapsPlace
		s.r.AddGmapsPlace(newGmapsPlace)
	} else if r.GmapsPlace.ID != 0 {
		// This restaurant already has a GmapsPlace Record so we just update it.
		log.Printf("Updating GmapsPlace id: %d.\n", r.GmapsPlace.ID)
		// Timestamp this update.
		r.GmapsPlace.LastUpdated = time.Now().Format("2006-01-02T15:04:05Z")
		gmapsPlaceRecordsAffected := s.r.UpdateGmapsPlace(r.GmapsPlace)
		log.Printf("%d GmapsPlace records affected.\n", gmapsPlaceRecordsAffected)
	} else {
		log.Printf("Restaurant id: %d has no GmapsPlace record and update data has no GmapsPlace data.", r.ID)
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
func NewService(r Repository) Service {
	return service{r}
}
