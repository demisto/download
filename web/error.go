package web

import (
	"encoding/json"
	"net/http"
)

// Errors is a list of errors
type Errors struct {
	Errors []*Error `json:"errors"`
}

// Error holds the info about a web error
type Error struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

func (e *Error) Error() string {
	return e.Title + ":" + e.Detail
}

// WriteError writes an error to the reply
func WriteError(w http.ResponseWriter, err *Error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(err)
}

var (
	// ErrBadRequest is a generic bad request
	ErrBadRequest = &Error{"bad_request", 400, "Bad request", "Request body is not well-formed. It must be JSON."}
	// ErrMissingPartRequest returns 400 if the request is missing some parts
	ErrMissingPartRequest = &Error{"missing_request", 400, "Bad request", "Request body is missing mandatory parts."}
	// ErrAuth if not authenticated
	ErrAuth = &Error{"unauthorized", 401, "Unauthorized", "The request requires authorization"}
	// ErrPermission if not authenticated
	ErrPermission = &Error{"forbidden", 403, "Forbidden", "The request requires the right permissions"}
	// ErrCredentials if there are missing / wrong credentials
	ErrCredentials = &Error{"invalid_credentials", 401, "Invalid credentials", "Invalid username or password"}
	// ErrNotAcceptable wrong accept header
	ErrNotAcceptable = &Error{"not_acceptable", 406, "Not Acceptable", "Accept header must be set to 'application/json'."}
	// ErrUnsupportedMediaType wrong media type
	ErrUnsupportedMediaType = &Error{"unsupported_media_type", 415, "Unsupported Media Type", "Content-Type header must be set to: 'application/json'."}
	// ErrCSRF missing CSRF cookie or parameter
	ErrCSRF = &Error{"forbidden", 403, "Forbidden", "Issue with CSRF code"}
	// ErrInternalServer if things go wrong on our side
	ErrInternalServer = &Error{"internal_server_error", 500, "Internal Server Error", "Something went wrong."}
)
