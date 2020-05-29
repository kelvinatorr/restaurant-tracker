package adder

import "fmt"

// Service provides beer adding operations.
type Service interface {
	AddInitRestaurants(string)
}

// Repository provides access to restaurant repository.
type Repository interface {
	// AddRestaurant saves a given restaurant to the repository.
	AddRestaurant(Restaurant)
	// GetAllRestaurants returns all restaurants saved in storage.
	// GetAllRestaurants() []lister.Restaurant
}

type service struct {
	r Repository
}

func (s *service) AddRestaurant(r []Restaurant) {

}

func (s *service) AddInitRestaurants(r string) {
	fmt.Println(r)
}

func main() {
	fmt.Println("Hello from adder!")
}

// NewService creates an adding service with the necessary dependencies
func NewService(r Repository) Service {
	return &service{r}
}
