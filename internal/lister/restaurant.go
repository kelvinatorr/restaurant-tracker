package lister

type Restaurant struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	Cuisine    string     `json:"cuisine"`
	Note       string     `json:"note"`
	Address    string     `json:"address"`
	CityState  cityState  `json:"city_state"`
	Zipcode    string     `json:"zipcode"`
	Latitude   float32    `json:"latitude"`
	Longitude  float32    `json:"longitude"`
	GmapsPlace gmapsPlace `json:"gmaps_place"`
}

type cityState struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

type gmapsPlace struct {
	ID                   int64   `json:"id"`
	LastUpdated          string  `json:"last_updated"`
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
}
