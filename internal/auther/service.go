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
	CheckJWT(string) error
	generateJWT(string, string) (string, error)
	GetCookiePayload(string) (UserJWT, error)
}

// Repository provides access to User repository.
type Repository interface {
	Begin()
	Commit()
	Rollback()
	GetUserAuthByEmail(string) User
	UpdateUserRememberToken(User) int64
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
	foundUser := s.r.GetUserAuthByEmail(u.Email)
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

	if foundUser.RememberToken == "" {
		log.Printf("%s has no remember_hash. Generating...", u.Email)
		// Generate a new remember hash
		rT, err := rememberToken()
		if err != nil {
			err = fmt.Errorf("There was an error generating a remember token")
			return "", err
		}
		foundUser.RememberToken = rT
		// Save it to the database
		s.r.Begin()
		// Defer rollback just in case there is a problem.
		defer s.r.Rollback()
		recordsAffected := s.r.UpdateUserRememberToken(foundUser)
		log.Printf("%d user records affected.", recordsAffected)
		s.r.Commit()
	}

	// Generate new jwt
	jwt, err := s.generateJWT(u.Email, foundUser.RememberToken)

	// Return the jwt
	return jwt, err
}

// CheckJWT checks if a JWT is valid. First by checking that the signature matches and that the rememberToken is still
// in the db for that user. If there is any problem an error is returned.
func (s service) CheckJWT(jwt string) error {
	var err error

	jwtParts, err := splitJWT(jwt)
	if err != nil {
		return err
	}

	header := jwtParts[0]
	payload := jwtParts[1]
	signature := jwtParts[2]

	// Regenerate the sig
	regenSig := s.genJWTSignature(header, payload)

	// Check that the signature matches what was passed in
	if regenSig != signature {
		return fmt.Errorf("JWT has the wrong signature")
	}

	var uJWT UserJWT
	uJWT, err = decodeCookiePayload(payload)
	if err != nil {
		return err
	}
	// Check the token is still valid
	foundUser := s.r.GetUserAuthByEmail(strings.ToLower(uJWT.Email))
	if foundUser.ID == 0 {
		return fmt.Errorf("JWT has a non-existent user")
	}
	if foundUser.RememberToken != uJWT.RememberToken {
		return fmt.Errorf("JWT has an invalid remember token")
	}

	return err
}

func (s service) GetCookiePayload(jwt string) (UserJWT, error) {
	var uJWT UserJWT

	jwtParts, err := splitJWT(jwt)
	if err != nil {
		return uJWT, err
	}

	uJWT, err = decodeCookiePayload(jwtParts[1])
	if err != nil {
		return uJWT, err
	}

	return uJWT, nil
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

func decodeCookiePayload(payload string) (UserJWT, error) {
	var uJWT UserJWT
	pB, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return uJWT, fmt.Errorf("JWT payload had a base64 decoding error")
	}
	err = json.Unmarshal(pB, &uJWT)
	if err != nil {
		return uJWT, fmt.Errorf("JWT has the wrong payload format")
	}
	return uJWT, nil
}

func splitJWT(jwt string) ([]string, error) {
	// Split the string
	jwtParts := strings.Split(jwt, ".")
	if len(jwtParts) != 3 {
		return []string{}, fmt.Errorf("JWT has the wrong number of parts")
	}
	return jwtParts, nil
}

// generateJWT generates string tokens of rememberTokenBytes byte size
func (s service) generateJWT(email string, rememberToken string) (string, error) {
	// Make the header
	hS := header{Alg: "HS256", Typ: "JWT"}
	hB, err := json.Marshal(hS)
	if err != nil {
		return "", err
	}
	// base64url encode it
	h := base64.RawURLEncoding.EncodeToString(hB)

	// make the payload
	uJWTS := UserJWT{Email: email, RememberToken: rememberToken}
	uJWTB, err := json.Marshal(uJWTS)
	if err != nil {
		return "", err
	}
	// base64url encode it
	p := base64.RawURLEncoding.EncodeToString(uJWTB)

	// hmac it up
	sig := s.genJWTSignature(h, p)

	return h + "." + p + "." + sig, nil
}

func (s service) hash(str string) string {
	s.hmac.Reset()
	s.hmac.Write([]byte(str))
	b := s.hmac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s service) genJWTSignature(header string, payload string) string {
	return s.hash(header + "." + payload)
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
