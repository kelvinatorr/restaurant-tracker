package remover

import "log"

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
	RemoveGmapsPlace(int64) int64
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
	// Remove the GmapsPlace
	var gmapsPlaceRecordsAffected int64
	if r.GmapsPlaceID != 0 {
		gmapsPlaceRecordsAffected = s.r.RemoveGmapsPlace(r.GmapsPlaceID)
		log.Printf("Removed GmapsPlace id: %d. Records affected: %d\n", r.GmapsPlaceID, gmapsPlaceRecordsAffected)
	}
	// Return the total records affected
	// TODO: Remove city too.
	s.r.Commit()
	return restaurantRecordsAffected + gmapsPlaceRecordsAffected
}

// NewService returns a new remover.service
func NewService(r Repository) Service {
	return service{r}
}
