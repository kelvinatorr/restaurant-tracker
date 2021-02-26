package lister

import (
	"fmt"
	"net/url"
	"strings"
)

type FilterOption struct {
	Value    string
	Selected bool
}

type FilterOptions struct {
	Cuisine []FilterOption
	City    []FilterOption
	State   []FilterOption
}

type FilterOperation struct {
	Field     string
	FieldType string
	Operator  string
	Value     string
}

func getAllowedOperators() map[string]string {
	return map[string]string{
		"eq":   "=",
		"lt":   "<",
		"gt":   ">",
		"lteq": "<=",
		"gteq": ">=",
		"is":   "is",
	}
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

		filterOp, err := parseFilterArg(k, v)
		if err != nil {
			return result, err
		} else if filterOp.Field == "" && filterOp.Operator == "" {
			// skip this since it isn't a filter query param
			continue
		}

		// Check and sanitize the filter params
		if f, check := allowedFields[filterOp.Field]; check {
			filterOp.Field = f.Name
			if o, check := allowedOperators[filterOp.Operator]; check {
				filterOp.FieldType = f.Type
				filterOp.Operator = o
				result = append(result, filterOp)
			} else {
				err := fmt.Errorf("Bad filter operator: %s", filterOp.Operator)
				return result, err
			}
		} else {
			err := fmt.Errorf("%s is not a valid filter field", filterOp.Field)
			return result, err
		}
	}
	return result, nil
}

func (s service) GetFilterParam(object string, filterRequested url.Values) FilterOperation {
	var result FilterOperation

	for k, v := range filterRequested {
		f, _ := parseFilterArg(k, v) // ignore bad filter arguments
		if f.Field == "" && f.Operator == "" {
			// skip this since it isn't a filter query param
			continue
		} else if f.Field == object {
			result = f
			return result
		}
	}
	return result
}

// parseFilterArg just parses the filter[]= query param. It does not sanitize it.
func parseFilterArg(keyArg string, valueArg []string) (FilterOperation, error) {
	var result FilterOperation

	if keyArg[:6] != "filter" {
		// return an empty string array but no error
		return result, nil
	}

	fkey := strings.Split(strings.Trim(keyArg[6:], "[]"), "|")
	if len(fkey) != 2 {
		err := fmt.Errorf("Bad filter argument: %s", keyArg)
		return result, err
	}
	result.Field = fkey[0]
	result.Operator = fkey[1]
	result.Value = valueArg[0]
	return result, nil
}
