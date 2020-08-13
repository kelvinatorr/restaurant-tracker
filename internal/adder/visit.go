package adder

type Visit struct {
	RestaurantID  int64       `json:"restaurant_id"`
	VisitDateTime string      `json:"visit_datetime"`
	Note          string      `json:"note"`
	VisitUsers    []VisitUser `json:"visit_users"`
}
