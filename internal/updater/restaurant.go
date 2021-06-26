package updater

type Restaurant struct {
	ID                int64      `json:"id" schema:"id,required"`
	Name              string     `json:"name" schema:"name,required"`
	Cuisine           string     `json:"cuisine" schema:"cuisine,required"`
	BusinessStatus    int        `json:"business_status" schema:"businessStatus"`
	Note              string     `json:"note" schema:"note"`
	Address           string     `json:"address" schema:"address"`
	CityState         CityState  `json:"city_state" schema:"cityState"`
	Zipcode           string     `json:"zipcode" schema:"zipCode"`
	Latitude          float32    `json:"latitude" schema:"latitude"`
	Longitude         float32    `json:"longitude" schema:"longitude"`
	GmapsPlace        GmapsPlace `json:"gmaps_place" schema:"gmapsPlace"`
	CityID            int64      `json:"city_id"`
	LastVisitDatetime string     `json:"last_visit_datetime"`
}

type CityState struct {
	Name  string `json:"name" schema:"city"`
	State string `json:"state" schema:"state"`
}

type GmapsPlace struct {
	ID                   int64   `json:"id" schema:"gmapsPlaceID"`
	LastUpdated          string  `json:"last_updated" schema:"lastUpdated"`
	PlaceID              string  `json:"place_id" schema:"placeID"`
	BusinessStatus       string  `json:"business_status" schema:"businessStatus"`
	FormattedPhoneNumber string  `json:"formatted_phone_number" schema:"phone"`
	Name                 string  `json:"name" schema:"gmapsName"`
	PriceLevel           int     `json:"price_level" schema:"priceLevel"`
	Rating               float32 `json:"rating" schema:"gmapsRating"`
	URL                  string  `json:"url" schema:"url"`
	UserRatingsTotal     int     `json:"user_ratings_total" schema:"nUserRatings"`
	UTCOffset            int     `json:"utc_offset" schema:"utcOffset"`
	Website              string  `json:"website" schema:"website"`
	RestaurantID         int64   `json:"restaurant_id"`
}
