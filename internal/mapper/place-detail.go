package mapper

type addressComponent struct {
	LongName  string   `json:"long_name"`
	ShortName string   `json:"short_name"`
	Types     []string `json:"types"`
}

type geometry struct {
	Location location `json:"location"`
}

type location struct {
	Lat float32 `json:"lat"`
	Lng float32 `json:"lng"`
}

type placeDetailResult struct {
	PlaceID              string             `json:"place_id"`
	BusinessStatus       string             `json:"business_status"`
	FormattedPhoneNumber string             `json:"formatted_phone_number"`
	Name                 string             `json:"name"`
	PriceLevel           int                `json:"price_level"`
	Rating               float32            `json:"rating"`
	URL                  string             `json:"url"`
	UserRatingsTotal     int                `json:"user_ratings_total"`
	UTCOffset            int                `json:"utc_offset"`
	Website              string             `json:"website"`
	AddressComponents    []addressComponent `json:"address_components"`
	Geometry             geometry           `json:"geometry"`
	Address              string
	ZipCode              string
}

type PlaceDetail struct {
	Result placeDetailResult `json:"result"`
}
