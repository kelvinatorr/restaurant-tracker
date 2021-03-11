package mapper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

// Service provides mapping operations.
type Service interface {
	PlaceSearch(string) ([]Candidate, error)
	HaveGmapsKey() bool
}

type service struct {
	apiKey string
}

const baseGmapsURL string = "https://maps.googleapis.com/maps/api/place/%s/json?"

func (s service) HaveGmapsKey() bool {
	return s.apiKey != ""
}

func (s service) PlaceSearch(searchTerm string) ([]Candidate, error) {
	var result []Candidate

	v := url.Values{}
	v.Set("key", s.apiKey)
	v.Add("inputtype", "textquery")
	v.Add("input", searchTerm)
	v.Add("fields", "place_id,name,formatted_address")

	getURL := fmt.Sprintf(baseGmapsURL, "findplacefromtext") + v.Encode()

	log.Printf("Querying Google Maps Place search for: %s", searchTerm)
	resp, err := http.Get(getURL)
	if err != nil {
		log.Println(err)
		return result, fmt.Errorf("There was a problem querying for results")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return result, fmt.Errorf("There was a problem reading the results")
	}

	ps := placeSearch{}
	getJSON(body, &ps)

	// https://developers.google.com/maps/documentation/places/web-service/search#ErrorMessages
	if ps.Status != "OK" && ps.Status != "ZERO_RESULTS" {
		log.Printf("ERROR: %s %s", ps.Status, ps.ErrorMessage)
		return result, fmt.Errorf("There was a problem with the results")
	}

	result = ps.Candidates

	return result, nil
}

func getJSON(body []byte, v interface{}) {
	err := json.Unmarshal(body, v)
	if err != nil {
		log.Panicln(err)
	}
}

// NewService provides a new map service
func NewService(key string) Service {
	return service{
		apiKey: key,
	}
}
