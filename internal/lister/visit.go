package lister

type Visit struct {
	ID            int64       `json:"id"`
	RestaurantID  int64       `json:"restaurant_id"`
	VisitDateTime string      `json:"visit_datetime"`
	Note          string      `json:"note"`
	VisitUsers    []VisitUser `json:"visit_users"`
}

type user struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}
