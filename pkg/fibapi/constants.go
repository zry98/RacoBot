package fibapi

import (
	"fmt"
)

// URLs
const (
	serverName     = "api.fib.upc.edu"
	baseURL        = "https://api.fib.upc.edu/v2"
	oAuthAuthURL   = "https://api.fib.upc.edu/v2/o/authorize/"
	oAuthTokenURL  = "https://api.fib.upc.edu/v2/o/token"
	oAuthRevokeURL = "https://api.fib.upc.edu/v2/o/revoke_token/"

	// use `.json` suffix to avoid setting an HTTP header when making requests
	userInfoURL              = "https://api.fib.upc.edu/v2/jo.json"
	noticesURL               = "https://api.fib.upc.edu/v2/jo/avisos.json"
	subjectsURL              = "https://api.fib.upc.edu/v2/jo/assignatures.json"
	publicSubjectURLTemplate = "https://api.fib.upc.edu/v2/assignatures/%s.json"
	publicSubjectsURL        = "https://api.fib.upc.edu/v2/assignatures.json"
	loginRedirectBaseURL     = "https://api.fib.upc.edu/v2/accounts/login/?next="
)

const (
	// the response text of an OAuth get token request when the Authorization Code is invalid
	OAuthInvalidAuthorizationCodeResponse = `{"error": "invalid_grant"}`
	// the response text of a PublicSubject request when the subject is not found
	publicSubjectNotFoundResponse = `{"detail": "Not found."}`
	publicAPIClientIDHeader       = "client_id"
)

// errors
var (
	ErrInvalidAuthorizationCode = fmt.Errorf("invalid authorization code")
	ErrAuthorizationExpired     = fmt.Errorf("authorization expired")
	ErrNoticeNotFound           = fmt.Errorf("notice not found")
	ErrResourceNotFound         = fmt.Errorf("resource not found")
	ErrSubjectNotExists         = fmt.Errorf("subject not exists")
	ErrUnknown                  = fmt.Errorf("unknown error")
)
