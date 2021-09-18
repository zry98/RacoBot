package fibapi

import (
	"errors"
)

// URLs
const (
	// TODO: build URLs dynamically from BaseURL?
	BaseURL = "https://api.fib.upc.edu/v2"

	OAuthAuthURL   = "https://api.fib.upc.edu/v2/o/authorize/"
	OAuthTokenURL  = "https://api.fib.upc.edu/v2/o/token"
	OAuthRevokeURL = "https://api.fib.upc.edu/v2/o/revoke_token/"

	// use `.json` suffix to avoid setting an HTTP header when making requests
	UserInfoURL = "https://api.fib.upc.edu/v2/jo.json"
	NoticesURL  = "https://api.fib.upc.edu/v2/jo/avisos.json"
	SubjectsURL = "https://api.fib.upc.edu/v2/jo/assignatures.json"

	LoginRedirectBaseURL = "https://api.fib.upc.edu/v2/accounts/login/?next="
)

// the response text of an OAuth get token request when the Authorization Code is invalid
const OAuthInvalidAuthorizationCodeResponse = `{"error": "invalid_grant"}`

// errors
var (
	ErrInvalidAuthorizationCode = errors.New("fibapi: invalid authorization code")
	ErrAuthorizationExpired     = errors.New("fibapi: authorization expired")
	ErrNoticeNotFound           = errors.New("fibapi: notice not found")
	ErrUnknown                  = errors.New("fibapi: unknown error")
)
