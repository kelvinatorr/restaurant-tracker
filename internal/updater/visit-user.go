package updater

type VisitUser struct {
	ID      int64 `json:"id"`
	VisitID int64
	UserID  int64 `json:"user_id"`
	Rating  int64 `json:"rating"`
}
