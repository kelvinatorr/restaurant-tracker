package adder

type Restaurant struct {
	Name       string     `json:"name"`
	Cuisine    string     `json:"cuisine"`
	Note       string     `json:"note"`
	Address    string     `json:"address"`
	Zipcode    string     `json:"zipcode"`
	CityState  CityState  `json:"city_state"`
	Latitude   float32    `json:"latitude"`
	Longitude  float32    `json:"longitude"`
	GmapsPlace GmapsPlace `json:"gmaps_place"`
	CityID     int64      `json:"city_id"`
}

type CityState struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

type GmapsPlace struct {
	PlaceID              string  `json:"place_id"`
	BusinessStatus       string  `json:"business_status"`
	FormattedPhoneNumber string  `json:"formatted_phone_number"`
	Name                 string  `json:"name"`
	PriceLevel           int     `json:"price_level"`
	Rating               float32 `json:"rating"`
	URL                  string  `json:"url"`
	UserRatingsTotal     int     `json:"user_ratings_total"`
	UTCOffset            int     `json:"utc_offset"`
	Website              string  `json:"website"`
	RestaurantID         int64   `json:"restaurant_id"`
}
