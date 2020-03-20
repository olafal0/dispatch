package auth

import (
	"database/sql"
	"errors"
	"log"

	"github.com/olafal0/dispatch/kvstore"
	"golang.org/x/crypto/bcrypt"
)

var bcryptCost = 12

// LoginManager is an object made for managing user signin and authentication
// using the built-in token signing and key-value storage mechanisms (based on
// sqlite).
//
// The provided methods can easily be used with the dispatch API framework by
// adding routes for SignupUser and AuthenticateUser.
type LoginManager struct {
	DB    *kvstore.KeyValueDB
	Token *TokenSigner
}

// ErrorIncorrectLogin represents a failed login attempt.
var ErrorIncorrectLogin = errors.New("Invalid username or password")

// UserLogin stores the information needed for a login attempt.
type UserLogin struct {
	Username string
	Password string
}

// SavedUser represents the data stored for a signed-up user.
type SavedUser struct {
	Username       string
	HashedPassword []byte
}

// SignupUser creates and stores user information for the new user. Upon
// successful registration, the user is signed in and the new generated token
// is returned.
func (lm *LoginManager) SignupUser(login UserLogin) (token string, err error) {
	existing := SavedUser{}
	err = lm.DB.Table("users").GetObject(login.Username, &existing)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	if existing.Username != "" {
		log.Println("User already exists")
		return "", errors.New("User already exists")
	}

	hashed, err := GetHash(login.Password)
	if err != nil {
		return "", err
	}

	err = lm.DB.Table("users").SetObject(login.Username, SavedUser{login.Username, hashed})
	if err != nil {
		return "", err
	}

	return lm.AuthenticateUser(login)
}

// AuthenticateUser attempts to log in an existing user with the provided
// credentials, returning an access token if the credentials match.
func (lm *LoginManager) AuthenticateUser(login UserLogin) (token string, err error) {
	existing := SavedUser{}
	err = lm.DB.Table("users").GetObject(login.Username, &existing)
	if err != nil {
		return "", err
	}

	valid := CheckPassword(login.Password, existing.HashedPassword)
	if valid {
		// No error; login successful
		return lm.Token.CreateToken(login.Username)
	}

	return "", ErrorIncorrectLogin
}

// GetHash returns the bcrypt hash of the provided password.
func GetHash(password string) ([]byte, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return nil, err
	}
	return hashed, nil
}

// CheckPassword returns true if the provided password matches the hash; else false.
func CheckPassword(password string, hash []byte) bool {
	comparisonErr := bcrypt.CompareHashAndPassword(hash, []byte(password))
	if comparisonErr == nil {
		return true
	}
	return false
}
