package adder

type VisitUser struct {
	VisitID int64
	UserID  int64 `json:"user_id"`
	Rating  int64 `json:"rating"`
}
