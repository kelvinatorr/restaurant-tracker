package updater

import (
	"fmt"
	"log"
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
	fmt.Println(r.CityID)
	// TODO: If the gmaps place id is 0 and PlaceID is not "" then insert it and get the id.
	// TODO: ElseIf if PlaceID is not "", update the gmaps place.
	// TODO: ElseIf if PlaceID is not "", update the gmaps place.
	// TODO: Handle errors, rollback?
	// Update the restaurant.
	rowsAffected := s.r.UpdateRestaurant(r)

	s.r.Commit()

	return rowsAffected
}

// NewService returns a new lister.service
func NewService(r Repository) Service {
	return service{r}
}
