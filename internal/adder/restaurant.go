package adder

type Restaurant struct {
	Name           string     `json:"name" schema:"name,required"`
	Cuisine        string     `json:"cuisine" schema:"cuisine,required"`
	BusinessStatus int        `json:"business_status" schema:"businessStatus"`
	Note           string     `json:"note" schema:"note"`
	Address        string     `json:"address" schema:"address"`
	Zipcode        string     `json:"zipcode" schema:"zipCode"`
	CityState      CityState  `json:"city_state" schema:"cityState"`
	Latitude       float32    `json:"latitude" schema:"latitude"`
	Longitude      float32    `json:"longitude" schema:"longitude"`
	GmapsPlace     GmapsPlace `json:"gmaps_place"`
	CityID         int64      `json:"city_id"`
}

type CityState struct {
	Name  string `json:"name" schema:"city"`
	State string `json:"state" schema:"state"`
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
