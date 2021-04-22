package mapper

type placeSearch struct {
	Candidates   []Candidate `json:"candidates"`
	Status       string      `json:"status"`
	ErrorMessage string      `json:"error_message"`
}

type Candidate struct {
	Name             string `json:"name"`
	PlaceID          string `json:"place_id"`
	FormattedAddress string `json:"formatted_address"`
}
