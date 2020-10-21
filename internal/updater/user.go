package updater

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name" schema:"firstName,required"`
	LastName  string `json:"last_name" schema:"lastName,required"`
	Email     string `json:"email" schema:"email,required"`
}
