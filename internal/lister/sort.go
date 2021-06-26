package lister

import (
	"fmt"
	"log"
	"net/url"
	"strings"
)

type SortOperation struct {
	Field     string
	Direction string
}

func getAllowedDirections() map[string]bool {
	return map[string]bool{"asc": true, "desc": true}
}

func (s service) getAllowedSortFields(object string) (map[string]string, error) {
	switch object {
	case "restaurant":
		return s.r.RestaurantSortFields(), nil
	case "visit":
		return s.r.VisitSortFields(), nil
	default:
		return make(map[string]string), fmt.Errorf("Unknown sort object %s", object)
	}

}

// checkSort checks that the sort params from the user are valid and prevents sql injections
func (s service) checkSort(object string, sortRequested url.Values) ([]SortOperation, error) {

	var result []SortOperation

	// If there is nothing to sort then just return
	if len(sortRequested) == 0 {
		return result, nil
	}

	allowedSortFields, err := s.getAllowedSortFields(object)
	if err != nil {
		return result, err
	}
	allowedDirections := getAllowedDirections()

	for k, v := range sortRequested {

		sortOp := parseSortArg(k, v)
		if sortOp.Field == "" && sortOp.Direction == "" {
			// skip this since it isn't a sort query param. Use && instead of || so the user sees an error on a badly formed sort
			// query
			continue
		}

		if sf, check := allowedSortFields[sortOp.Field]; check {
			if _, check := allowedDirections[sortOp.Direction]; check {
				sortOp.Field = sf
				result = append(result, sortOp)
			} else {
				err := fmt.Errorf("Bad sort direction")
				return result, err
			}
		} else {
			log.Printf("%s is not valid!", sortOp.Field)
			err := fmt.Errorf("%s is not a valid sort field", sortOp.Field)
			return result, err
		}
	}
	return result, nil
}

func (s service) GetSortParam(object string, sortRequested url.Values) SortOperation {
	var result SortOperation

	for k, v := range sortRequested {
		s := parseSortArg(k, v)
		if s.Field == "" && s.Direction == "" {
			// skip this since it isn't a sort query param
			continue
		} else if s.Field == object {
			result = s
			return result
		}
	}
	return result
}

// parseSortArg just parses the sort[]= query param. It does not sanitize it.
func parseSortArg(keyArg string, valueArg []string) SortOperation {
	var result SortOperation

	if keyArg[:4] != "sort" {
		// return an empty string object
		return result
	}

	result.Field = strings.Trim(keyArg[4:], "[]")
	result.Direction = valueArg[0]
	return result
}
