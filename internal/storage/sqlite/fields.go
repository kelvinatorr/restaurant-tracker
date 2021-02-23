package sqlite

import "github.com/kelvinatorr/restaurant-tracker/internal/lister"

func (s Storage) RestaurantSortFields() map[string]string {
	restaurantFields := make(map[string]string)
	restaurantFields["name"] = "res.name"
	restaurantFields["cuisine"] = "cuisine"
	restaurantFields["city"] = "city_name"
	restaurantFields["state"] = "state_name"
	restaurantFields["last_visit"] = "last_visit_datetime"
	restaurantFields["avg_rating"] = "avg_rating"
	return restaurantFields
}

func (s Storage) RestaurantFilterFields() map[string]lister.Field {
	restaurantFields := make(map[string]lister.Field)
	restaurantFields["name"] = lister.Field{Name: "res.name", Type: "TEXT"}
	restaurantFields["cuisine"] = lister.Field{Name: "cuisine", Type: "TEXT"}
	restaurantFields["city"] = lister.Field{Name: "city_name", Type: "TEXT"}
	restaurantFields["state"] = lister.Field{Name: "state_name", Type: "TEXT"}
	restaurantFields["last_visit"] = lister.Field{Name: "last_visit_datetime", Type: "TEXT"}
	restaurantFields["avg_rating"] = lister.Field{Name: "avg_rating", Type: "REAL"}
	return restaurantFields
}
