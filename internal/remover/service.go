package remover

import (
	"log"

	"github.com/kelvinatorr/restaurant-tracker/internal/lister"
)

// Service provides removing operations.
type Service interface {
	RemoveRestaurant(Restaurant) int64
}

// Repository provides access to restaurant repository.
type Repository interface {
	Begin()
	Commit()
	Rollback()
	RemoveRestaurant(Restaurant) int64
	GetRestaurantsByCity(int64) []lister.Restaurant
	RemoveCity(int64) int64
}

type service struct {
	r Repository
}

func (s service) RemoveRestaurant(r Restaurant) int64 {
	s.r.Begin()
	// Defer Rollback just in case thre is a problem.
	defer s.r.Rollback()
	// Remove the Restaurant
	restaurantRecordsAffected := s.r.RemoveRestaurant(r)
	log.Printf("Removed Restaurant id: %d. Records affected: %d\n", r.ID, restaurantRecordsAffected)

	// Remove city too.
	cityRecordsAffected := s.removeCity(r.CityID)
	s.r.Commit()
	// Return the total records affected
	return restaurantRecordsAffected + cityRecordsAffected
}

// removeCity removes a city if there are no longer any restaurants referencing it. Caller must call s.r.Commit()
// otherwise cities won't actually be removed!
func (s service) removeCity(cityID int64) int64 {
	var recordsAffected int64
	// Check if there are any restaurants with this cityID
	countOfRestaurants := len(s.r.GetRestaurantsByCity(cityID))
	// If there's 1 or none then remove it (the 1 is the restaurant we are currently deleting).
	if countOfRestaurants <= 1 {
		recordsAffected = s.r.RemoveCity(cityID)
		log.Printf("Removed City id: %d. Records affected: %d\n", cityID, recordsAffected)
	}
	// Return the number of records affected.
	return recordsAffected
}

// NewService returns a new remover.service
func NewService(r Repository) Service {
	return service{r}
}
