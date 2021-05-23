package updater

type Visit struct {
	ID            int64       `json:"id" schema:"id,required"`
	RestaurantID  int64       `json:"restaurant_id" schema:"restaurantID,required"`
	VisitDateTime string      `json:"visit_datetime" schema:"visitDateTime,required"`
	Note          string      `json:"note" schema:"note"`
	VisitUsers    []VisitUser `json:"visit_users" schema:"visitUsers"`
}
