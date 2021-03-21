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
	PlaceDetails(string) (PlaceDetail, error)
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

func (s service) PlaceDetails(placeID string) (PlaceDetail, error) {
	// Get place details using Place Details Request
	pd := PlaceDetail{}

	v := url.Values{}
	v.Set("key", s.apiKey)
	v.Add("place_id", placeID)
	v.Add("fields", "name,place_id,business_status,formatted_phone_number,price_level,rating,url,user_ratings_total,utc_offset,website,address_components,geometry")
	getURL := fmt.Sprintf(baseGmapsURL, "details") + v.Encode()

	resp, err := http.Get(getURL)
	if err != nil {
		log.Println(err)
		return pd, fmt.Errorf("There was a problem querying for results")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return pd, fmt.Errorf("There was a problem reading the results")
	}
	getJSON(body, &pd)

	// Parse the address components into a simple address and zip code
	pd.Result.Address, pd.Result.ZipCode = parseAddress(pd.Result.AddressComponents)

	return pd, nil
}

// Takes a slice of addressComponents from Google's api response and returns a street address and zip code
func parseAddress(ad []addressComponent) (string, string) {
	var address, zipCode string
	var addressMap = make(map[string]string)
	for _, ac := range ad {
		t := ac.Types[0]
		switch t {
		case "street_number":
			addressMap["streetNumber"] = ac.LongName
		case "route":
			addressMap["route"] = ac.LongName
		case "postal_code":
			zipCode = ac.LongName
		case "subpremise":
			addressMap["subpremise"] = ac.LongName
		}
	}
	address = fmt.Sprintf("%s %s", addressMap["streetNumber"], addressMap["route"])
	if subpremise, ok := addressMap["subpremise"]; ok {
		address += " " + subpremise
	}
	return address, zipCode
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
