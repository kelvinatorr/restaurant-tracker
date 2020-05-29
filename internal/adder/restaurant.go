package adder

type cityState struct {
	id    int64
	city  string
	state string
}

type Restaurant struct {
	id           int64
	Name         string
	cuisine      string
	note         string
	address      string
	cityID       int64
	zipcode      string
	latitude     float32
	longitude    float32
	gmapsPlaceID int64
}
