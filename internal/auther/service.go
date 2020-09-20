package auther

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Service provides adding operations.
type Service interface {
	SignIn(UserSignIn) error
}

// Repository provides access to User repository.
type Repository interface {
	GetHashesByEmail(string) User
}

type service struct {
	r Repository
}

func (s service) SignIn(u UserSignIn) error {
	var err error
	// Lower case to normalize it.
	u.Email = strings.ToLower(u.Email)
	foundUser := s.r.GetHashesByEmail(u.Email)
	if foundUser.ID == 0 {
		err = fmt.Errorf("There is no user with this email address")
	} else {
		err = bcrypt.CompareHashAndPassword([]byte(foundUser.PasswordHash), []byte(u.Password))
		if err != nil {
			// Don't return too much detail about the error.
			err = fmt.Errorf("Incorrect password")
		}
	}
	return err
}

// HashPassword hashes a given password string using bcrypt with bcrypt DefaultCost
func HashPassword(password string) (string, error) {
	pwBytes := []byte(password)
	// bcrypt automatically includes salt.
	hashedBytes, err := bcrypt.GenerateFromPassword(pwBytes, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// NewService provides a new auth service
func NewService(r Repository) Service {
	return service{r}
}
