package lister

type Restaurant struct {
	ID         int64
	Name       string
	Cuisine    string
	Note       string
	Address    string
	CityState  cityState
	Zipcode    string
	Latitude   float32
	Longitude  float32
	GmapsPlace gmapsPlace
}

type cityState struct {
	ID    int64
	Name  string
	State string
}

type gmapsPlace struct {
	ID                   int64
	LastUpdated          string
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
