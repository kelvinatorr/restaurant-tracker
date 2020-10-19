package auther

type UserSignIn struct {
	Email    string
	Password string
}

type User struct {
	ID            int64
	PasswordHash  string
	RememberToken string
}

type userJWT struct {
	Email         string `json:"email"`
	RememberToken string `json:"rememberToken"`
}
