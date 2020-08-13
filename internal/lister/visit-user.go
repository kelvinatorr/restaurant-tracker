package lister

type VisitUser struct {
	ID     int64  `json:"id"`
	User   User   `json:"user"`
	Rating string `json:"rating"`
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}
