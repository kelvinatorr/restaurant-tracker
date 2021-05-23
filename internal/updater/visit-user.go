package updater

type VisitUser struct {
	ID      int64 `json:"id schema:"id,required"`
	VisitID int64
	UserID  int64 `json:"user_id" schema:"userID,required"`
	Rating  int64 `json:"rating" schema:"rating"`
}
