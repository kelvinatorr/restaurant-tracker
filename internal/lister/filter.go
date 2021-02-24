package lister

import (
	"fmt"
	"net/url"
	"strings"
)

type FilterOptions struct {
	Cuisine []string
	City    []string
	State   []string
}

type FilterOperation struct {
	Field     string
	FieldType string
	Operator  string
	Value     string
}

type Filter struct {
	Cuisine string `schema:"cuisine"`
	City    string `schema:"city"`
	State   string `schema:"state"`
}

func getAllowedOperators() map[string]string {
	return map[string]string{"eq": "=", "lt": "<", "gt": ">", "lteq": "<=", "gteq": ">="}
}

func (s service) getAllowedFilterFields(object string) (map[string]Field, error) {
	switch object {
	case "restaurant":
		return s.r.RestaurantFilterFields(), nil
	default:
		return make(map[string]Field), fmt.Errorf("Unknown filter object %s", object)
	}

}

func (s service) checkFilter(object string, filterRequested url.Values) ([]FilterOperation, error) {

	var result []FilterOperation

	// If there are no filters requested then just return
	if len(filterRequested) == 0 {
		return result, nil
	}

	allowedFields, err := s.getAllowedFilterFields(object)
	if err != nil {
		return result, err
	}
	allowedOperators := getAllowedOperators()

	for k, v := range filterRequested {

		if k[:6] != "filter" {
			// skip this since it isn't a filter query param
			continue
		}

		fkey := strings.Split(strings.Trim(k[6:], "[]"), "|")
		if len(fkey) != 2 {
			err := fmt.Errorf("Bad filter argument: %s", k)
			return result, err
		}
		filterField := fkey[0]
		filterOperator := fkey[1]

		if f, check := allowedFields[filterField]; check {
			if o, check := allowedOperators[filterOperator]; check {
				value := v[0]
				filterOp := FilterOperation{Field: f.Name, FieldType: f.Type, Operator: o, Value: value}
				result = append(result, filterOp)
			} else {
				err := fmt.Errorf("Bad filter operator: %s", filterOperator)
				return result, err
			}
		} else {
			err := fmt.Errorf("%s is not a valid filter field", fkey)
			return result, err
		}
	}
	return result, nil
}
