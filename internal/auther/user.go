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

type UserJWT struct {
	ID            int64  `json:"id"`
	RememberToken string `json:"rememberToken"`
}

type UserChangePassword struct {
	ID int64
	CurrentPassword string `schema:"currentPassword,required"`
	NewPassword string `schema:"newPassword,required"`
	RepeatNewPassword string `schema:"repeatNewPassword,required"`
}