package adder

type Restaurant struct {
	Name         string
	Cuisine      string
	Note         string
	Address      string
	Zipcode      string
	CityState    CityState
	latitude     float32
	longitude    float32
	CityID       int64
	gmapsPlaceID int64
	GmapsPlace   GmapsPlace
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
