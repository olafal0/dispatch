package dispatch

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/olafal0/dispatch/auth"
)

// AuthorizerHook is a middleware hook that populates the context's Claims object
// with data from the request's authorization token. If there is no authorization
// token, or the token is invalid, it returns an error.
//
// This hook effectively acts as a requirement that the authorization token is correct.
func AuthorizerHook(token *auth.TokenSigner) MiddlewareHook {
	return func(input *EndpointInput) (*EndpointInput, error) {
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
		wroteHeader := 200
		wroteStatus := http.StatusText(200)
		startTime := time.Now()
		defer func() {
			log.Printf("%v %s%s - %d %s", time.Since(startTime), r.Method, r.URL.Path, wroteHeader, wroteStatus)
		}()
		writeError := func(w http.ResponseWriter, error string, code int) {
			wroteHeader = code
			wroteStatus = http.StatusText(code)
			http.Error(w, error, code)
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "PUT, POST, GET, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ctx := &Context{Request: r}
		output, err := api.Call(r.Method, r.URL.Path, ctx, data)
		if err != nil {
			if err == ErrorNotFound {
				writeError(w, err.Error(), http.StatusNotFound)
			} else if err == ErrorBadRequest {
				writeError(w, err.Error(), http.StatusBadRequest)
			} else {
				writeError(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		outBytes, err := json.Marshal(output)
		if err != nil {
			writeError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(outBytes)
	}
}
