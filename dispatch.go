package dispatch

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/olafal0/dispatch/auth"
)

// AuthorizerHook is a middleware hook that populates the context's Claims object
// with data from the request's authorization token. If there is no authorization
// token, or the token is invalid, it returns an error.
//
// This hook effectively acts as a requirement that the authorization token is correct.
func AuthorizerHook(input *EndpointInput, token *auth.TokenSigner) (*EndpointInput, error) {
	// Check for authorization header
	if input == nil {
		return nil, errors.New("Missing authorization token")
	}
	authToken := input.Ctx.Request.Header.Get("Authorization")
	if authToken == "" {
		return nil, errors.New("Missing authorization token")
	}

	claims, err := token.ParseToken(authToken)
	if err != nil {
		return nil, errors.New("Invalid authorization token")
	}
	input.Ctx.Claims = claims
	return input, nil
}

// GetHandler returns a handler function suitable for use in http.HandleFunc.
// For example:
//
//  	http.HandleFunc("/", api.GetHandler())
//  	log.Fatal(http.ListenAndServe(":8000", nil))
//
// The provided handler takes care of access control headers, CORS requests,
// JSON marshalling, and error handling.
func (api *API) GetHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "PUT, POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ctx := &Context{Request: r}
		output, err := api.Call(r.Method, r.URL.Path, ctx, data)
		if err != nil {
			if err == ErrorNotFound {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else if err == ErrorBadRequest {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		outBytes, err := json.Marshal(output)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(outBytes)
	}
}
