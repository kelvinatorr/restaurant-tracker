package adder

type User struct {
	FirstName      string `json:"first_name" schema:"first,required"`
	LastName       string `json:"last_name" schema:"lastName,required"`
	Email          string `json:"email" schema:"email,required"`
	Password       string `json:"password" schema:"password,required"`
	RepeatPassword string `schema:"repeatPassword,required"`
	PasswordHash   string `schema:"-"`
}
