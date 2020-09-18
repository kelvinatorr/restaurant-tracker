package auther

type UserSignIn struct {
	Email    string
	Password string
}

type User struct {
	ID int64
	PasswordHash string
	RememberHash string
}
