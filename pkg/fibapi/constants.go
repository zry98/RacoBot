package fibapi

import (
	"errors"
	"time"
)

// Response represents a response from the FIB API
type Response struct {
	Detail string `json:"detail,omitempty"`
	Error  string `json:"error,omitempty"`
}

// URLs
const (
	BaseURL        = "https://api.fib.upc.edu/"
	oauthAuthURL   = "https://api.fib.upc.edu/v2/o/authorize/"
	oauthTokenURL  = "https://api.fib.upc.edu/v2/o/token"
	oauthRevokeURL = "https://api.fib.upc.edu/v2/o/revoke_token/"

	// use `.json` suffix to avoid setting an HTTP header when making requests
	userInfoURL              = "https://api.fib.upc.edu/v2/jo.json"
	noticesURL               = "https://api.fib.upc.edu/v2/jo/avisos.json"
	subjectsURL              = "https://api.fib.upc.edu/v2/jo/assignatures.json"
	publicSubjectsURL        = "https://api.fib.upc.edu/v2/assignatures.json"
	publicSubjectURLTemplate = "https://api.fib.upc.edu/v2/assignatures/%s.json"
	loginRedirectBaseURL     = "https://api.fib.upc.edu/v2/accounts/login/?next="
)

const (
	OAuthAuthorizationCodeLength = 30

	oauthInvalidAuthorizationCodeResponseErrorMessage = "invalid_grant"
	resourceNotFoundResponseDetail                    = "Not found."
	publicAPIClientIDHeader                           = "client_id"
)

const (
	tlsHandshakeTimeout = 5 * time.Second
	httpClientTimeout   = 20 * time.Second
	requestTimeout      = 10 * time.Second
)

// errors
var (
	ErrInvalidAuthorizationCode = errors.New("fibapi: invalid authorization code")
	ErrAuthorizationExpired     = errors.New("fibapi: authorization has expired")
	ErrNoticeNotFound           = errors.New("fibapi: notice not found")
	ErrResourceNotFound         = errors.New("fibapi: resource not found")
)
