package lister

type VisitUser struct {
	ID     int64  `json:id`
	User   user   `json:user`
	Rating string `json:rating`
}
