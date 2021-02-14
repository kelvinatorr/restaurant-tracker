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

func getAllowedFields(object string) (map[string]string, error) {
	switch object {
	case "restaurant":
		return getAllowedRestaurantFields(), nil
	default:
		return make(map[string]string), fmt.Errorf("Unknown sort object %s", object)
	}

}

// checkSort checks that the sort params from the user are valid and prevents sql injections
func checkSort(object string, sortRequested url.Values) ([]SortOperation, error) {

	var result []SortOperation

	// If there is nothing to sort then just return
	if len(sortRequested) == 0 {
		return result, nil
	}

	allowedSortFields, err := getAllowedFields(object)
	if err != nil {
		return result, err
	}
	allowedDirections := getAllowedDirections()

	for k, v := range sortRequested {

		if k[:4] != "sort" {
			// skip this since it isn't a sort query param
			continue
		}

		sortField := strings.Trim(k[4:], "[]")

		if sf, check := allowedSortFields[sortField]; check {
			direction := v[0]
			if _, check := allowedDirections[direction]; check {
				sortOp := SortOperation{Field: sf, Direction: direction}
				result = append(result, sortOp)
			} else {
				err := fmt.Errorf("Bad sort direction")
				return result, err
			}
		} else {
			log.Printf("%s is not valid!", sortField)
			err := fmt.Errorf("%s is not a valid sort field", sortField)
			return result, err
		}
	}
	return result, nil
}
