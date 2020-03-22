package auth

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/olafal0/dispatch"
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
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// SavedUser represents the data stored for a signed-up user.
type SavedUser struct {
	Username       string
	HashedPassword []byte
}

// SignupUser creates and stores user information for the new user. Upon
// successful registration, the user is signed in and the new generated token
// is returned.
func (lm *LoginManager) SignupUser(login UserLogin, ctx *dispatch.Context) (err error) {
	existing := SavedUser{}
	err = lm.DB.Table("users").GetObject(login.Username, &existing)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if existing.Username != "" {
		log.Println("User already exists")
		return errors.New("User already exists")
	}

	hashed, err := GetHash(login.Password)
	if err != nil {
		return err
	}

	err = lm.DB.Table("users").SetObject(login.Username, SavedUser{login.Username, hashed})
	if err != nil {
		return err
	}

	return lm.AuthenticateUser(login, ctx)
}

// AuthenticateUser attempts to log in an existing user with the provided
// credentials, returning an access token if the credentials match.
func (lm *LoginManager) AuthenticateUser(login UserLogin, ctx *dispatch.Context) (err error) {
	existing := SavedUser{}
	err = lm.DB.Table("users").GetObject(login.Username, &existing)
	if err != nil {
		return err
	}

	valid := CheckPassword(login.Password, existing.HashedPassword)
	if valid {
		token, err := lm.Token.CreateToken(login.Username)
		// No error; login successful
		authCookie := &http.Cookie{
			Name:  "dispatch-auth",
			Value: token,
			// Secure: true,
			HttpOnly: true,
			SameSite: http.SameSiteDefaultMode,
			MaxAge:   int(time.Duration(time.Hour * 24 * 30).Seconds()), // 30 days
		}
		ctx.Writer.Header().Add("Set-Cookie", authCookie.String())
		loggedInCookie := &http.Cookie{
			Name:     "dispatch-logged-in",
			Value:    "true",
			Path:     "/",
			MaxAge:   int(time.Duration(time.Hour * 24 * 30).Seconds()), // 30 days
			SameSite: http.SameSiteNoneMode,
		}
		ctx.Writer.Header().Add("Set-Cookie", loggedInCookie.String())
		return err
	}

	return ErrorIncorrectLogin
}

// LogoutUser logs out a user by removing the session cookies containing their
// auth token.
func (lm *LoginManager) LogoutUser(ctx *dispatch.Context) {
	authCookie := &http.Cookie{
		Name:  "dispatch-auth",
		Value: "removed",
		// Secure: true,
		HttpOnly: true,
		SameSite: http.SameSiteDefaultMode,
		MaxAge:   -1,
	}
	ctx.Writer.Header().Add("Set-Cookie", authCookie.String())
	loggedInCookie := &http.Cookie{
		Name:  "dispatch-logged-in",
		Value: "false",
		Path:  "/",
	}
	ctx.Writer.Header().Add("Set-Cookie", loggedInCookie.String())
}

// AuthorizerHook is a middleware hook that populates the context's Claims object
// with data from the request's authorization token. If there is no authorization
// token, or the token is invalid, it returns an error.
//
// This hook effectively acts as a requirement that the authorization token is correct.
func AuthorizerHook(token *TokenSigner) dispatch.MiddlewareHook {
	return func(input *dispatch.EndpointInput) (*dispatch.EndpointInput, error) {
		// Check for authorization header
		if input == nil {
			return nil, errors.New("Missing authorization token")
		}
		authToken, err := input.Ctx.Request.Cookie("dispatch-auth")
		if authToken == nil || authToken.Value == "" {
			return nil, errors.New("Missing authorization token")
		}
		if err != nil {
			return nil, err
		}

		claims, err := token.ParseToken(authToken.Value)
		if err != nil {
			return nil, errors.New("Invalid authorization token")
		}
		input.Ctx.Claims = claims
		return input, nil
	}
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
