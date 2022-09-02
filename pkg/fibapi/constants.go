package fibapi

import (
	"errors"
)

// URLs
const (
	ServerName = "api.fib.upc.edu"
	// TODO: build URLs dynamically from BaseURL?
	BaseURL = "https://api.fib.upc.edu/v2"

	OAuthAuthURL   = "https://api.fib.upc.edu/v2/o/authorize/"
	OAuthTokenURL  = "https://api.fib.upc.edu/v2/o/token"
	OAuthRevokeURL = "https://api.fib.upc.edu/v2/o/revoke_token/"

	// use `.json` suffix to avoid setting an HTTP header when making requests
	UserInfoURL              = "https://api.fib.upc.edu/v2/jo.json"
	NoticesURL               = "https://api.fib.upc.edu/v2/jo/avisos.json"
	SubjectsURL              = "https://api.fib.upc.edu/v2/jo/assignatures.json"
	PublicSubjectURLTemplate = "https://api.fib.upc.edu/v2/assignatures/%s.json"

	LoginRedirectBaseURL = "https://api.fib.upc.edu/v2/accounts/login/?next="
)

const (
	// the response text of an OAuth get token request when the Authorization Code is invalid
	OAuthInvalidAuthorizationCodeResponse = `{"error": "invalid_grant"}`
	// the response text of a PublicSubject request when the subject is not found
	PublicSubjectNotFoundResponse = `{"detail": "Not found."}`
)

// errors
var (
	ErrInvalidAuthorizationCode = errors.New("fibapi: invalid authorization code")
	ErrAuthorizationExpired     = errors.New("fibapi: authorization expired")
	ErrNoticeNotFound           = errors.New("fibapi: notice not found")
	ErrResourceNotFound         = errors.New("fibapi: resource not found")
	ErrSubjectNotExists         = errors.New("fibapi: subject not exists")
	ErrUnknown                  = errors.New("fibapi: unknown error")
)
