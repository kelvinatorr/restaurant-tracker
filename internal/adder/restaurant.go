package adder

type Restaurant struct {
	Name         string
	Cuisine      string
	Note         string
	Address      string
	Zipcode      string
	CityState    CityState
	Latitude     float32
	Longitude    float32
	GmapsPlace   GmapsPlace
	CityID       int64
	GmapsPlaceID int64
}

type CityState struct {
	Name  string
	State string
}

type GmapsPlace struct {
	PlaceID              string
	BusinessStatus       string
	FormattedPhoneNumber string
	Name                 string
	PriceLevel           int
	Rating               float32
	URL                  string
	UserRatingsTotal     int
	UTCOffset            int
	Website              string
}
