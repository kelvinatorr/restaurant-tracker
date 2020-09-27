package auther

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"log"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Service provides authing operations.
type Service interface {
	SignIn(UserSignIn) (string, error)
	generateJWT(string, string) (string, error)
}

// Repository provides access to User repository.
type Repository interface {
	Begin()
	Commit()
	Rollback()
	GetHashesByEmail(string) User
	UpdateUserRememberHash(User) int64
}

type service struct {
	r    Repository
	hmac hash.Hash
}

const rememberTokenBytes int = 32

func (s service) SignIn(u UserSignIn) (string, error) {
	var err error
	// Lower case to normalize it.
	u.Email = strings.ToLower(u.Email)
	// Check this email exists
	foundUser := s.r.GetHashesByEmail(u.Email)
	if foundUser.ID == 0 {
		err = fmt.Errorf("There is no user with this email address")
		return "", err
	}

	// Check the password
	if err = bcrypt.CompareHashAndPassword([]byte(foundUser.PasswordHash), []byte(u.Password)); err != nil {
		// Don't return too much detail about the error.
		err = fmt.Errorf("Incorrect password")
		return "", err
	}

	if foundUser.RememberHash == "" {
		log.Printf("%s has no remember_hash. Generating...", u.Email)
		// Generate a new remember hash
		rT, err := rememberToken()
		if err != nil {
			err = fmt.Errorf("There was an error generating a remember token")
			return "", err
		}
		foundUser.RememberHash = s.hash(rT)
		// Save it to the database
		s.r.Begin()
		// Defer rollback just in case there is a problem.
		defer s.r.Rollback()
		recordsAffected := s.r.UpdateUserRememberHash(foundUser)
		log.Printf("%d user records affected.", recordsAffected)
		s.r.Commit()
	}

	// Generate new jwt
	jwt, err := s.generateJWT(u.Email, foundUser.RememberHash)

	// Return the jwt
	return jwt, err
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

// generateJWT generates string tokens of rememberTokenBytes byte size
func (s service) generateJWT(email string, rememberHash string) (string, error) {
	// Make the header
	hS := header{Alg: "HS256", Typ: "JWT"}
	hB, err := json.Marshal(hS)
	if err != nil {
		return "", err
	}
	// base64url encode it
	h := base64.RawURLEncoding.EncodeToString(hB)

	// make the payload
	uJWTS := userJWT{Email: email, RememberHash: rememberHash}
	uJWTB, err := json.Marshal(uJWTS)
	if err != nil {
		return "", err
	}
	// base64url encode it
	p := base64.RawURLEncoding.EncodeToString(uJWTB)

	// hmac it up
	sig := s.hash(h + "." + p)

	return h + "." + p + "." + sig, nil
}

func (s service) hash(str string) string {
	s.hmac.Reset()
	s.hmac.Write([]byte(str))
	b := s.hmac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(b)
}

func genRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// genRandomBytes will first generate a byte slice of size nBytes and then return a string that is the
// base64.RawURLEncoding() encoded version of that byte slice
func genRandomString(nBytes int) (string, error) {
	b, err := genRandomBytes(nBytes)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// rememberToken generates remember tokens of a predetermined byte size.
func rememberToken() (string, error) {
	return genRandomString(rememberTokenBytes)
}

// NewService provides a new auth service
func NewService(r Repository, key string) Service {
	h := hmac.New(sha256.New, []byte(key))
	return service{
		r:    r,
		hmac: h,
	}
}
