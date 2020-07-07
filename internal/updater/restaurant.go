package updater

type Restaurant struct {
	ID           int64
	Name         string
	Cuisine      string
	Note         string
	Address      string
	CityState    CityState
	Zipcode      string
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
