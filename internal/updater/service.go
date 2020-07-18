package updater

import (
	"fmt"
	"log"

	"github.com/kelvinatorr/restaurant-tracker/internal/adder"
)

// Service provides listing operations.
type Service interface {
	UpdateRestaurant(Restaurant) int64
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
}

type service struct {
	r Repository
}

func (s service) UpdateRestaurant(r Restaurant) int64 {
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
		gmapsPlaceRecordsAffected := s.r.UpdateGmapsPlace(r.GmapsPlace)
		log.Printf("%d GmapsPlace records affected.\n", gmapsPlaceRecordsAffected)
	} else {
		log.Printf("Restaurant id: %d has no GmapsPlace record and update data has no GmapsPlace data.", r.ID)
	}

	s.r.Commit()

	return recordsAffected
}

// NewService returns a new updater.service
func NewService(r Repository) Service {
	return service{r}
}
