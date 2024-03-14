/*
Package fibapi implements a simple library for getting information from the FIB API (https://api.fib.upc.edu/).
*/
package fibapi

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
)

// Config represents a configuration for FIB API OAuth application
type Config struct {
	OAuthClientID     string `toml:"oauth_client_id"`
	OAuthClientSecret string `toml:"oauth_client_secret"`
	OAuthRedirectURI  string `toml:"oauth_redirect_uri"`
	PublicClientID    string `toml:"public_client_id"`
	ClientUserAgent   string `toml:"client_user_agent,omitempty"`
}

var (
	oauthConf *oauth2.Config

	// HTTP request headers to send
	requestHeaders = map[string]string{
		"Accept":          "application/json",
		"Accept-Language": "es-ES",
		"User-Agent":      "RacoBot/1.0 (https://github.com/zry98/RacoBot)",
	}
)

// Init initializes the FIB API clients
func Init(config Config) {
	u, err := url.Parse(BaseURL)
	if err != nil {
		panic(err)
	}
	if config.ClientUserAgent != "" {
		requestHeaders["User-Agent"] = config.ClientUserAgent
	}

	// private API OAuth config
	oauthConf = &oauth2.Config{
		ClientID:     config.OAuthClientID,
		ClientSecret: config.OAuthClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   oauthAuthURL,
			TokenURL:  oauthTokenURL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
		RedirectURL: config.OAuthRedirectURI,
		Scopes:      []string{"read"},
	}

	// private API client
	privateClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName: u.Hostname(),
				// TODO: change min and max version to TLS 1.3 when FIB API server supports it
				MinVersion: tls.VersionTLS12,
				MaxVersion: tls.VersionTLS12,
			},
			TLSHandshakeTimeout: tlsHandshakeTimeout,
			ForceAttemptHTTP2:   false,
		},
		Timeout: httpClientTimeout,
	}

	// public API client
	publicClientID = config.PublicClientID
	publicClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName: u.Hostname(),
				MinVersion: tls.VersionTLS12,
				MaxVersion: tls.VersionTLS12,
			},
			TLSHandshakeTimeout: tlsHandshakeTimeout,
			ForceAttemptHTTP2:   false,
		},
		Timeout: httpClientTimeout,
	}
}

// NewAuthorizationURL generates an authorization URL with the given state
func NewAuthorizationURL(state string) string {
	return oauthConf.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// Authorize tries to retrieve OAuth token with the given Authorization Code
func Authorize(authorizationCode string) (token *oauth2.Token, userInfo UserInfo, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	token, err = oauthConf.Exchange(ctx, authorizationCode)
	if err != nil {
		err = ProcessTokenError(err)
		return
	}

	// try to get UserInfo and check if the retrieved token is really valid
	userInfo, err = NewClient(*token).GetUserInfo()
	if err != nil {
		return
	}
	if userInfo.Username == "" {
		err = ErrInvalidAuthorizationCode
	}
	return
}

// ProcessTokenError returns a more specific error from the given error
func ProcessTokenError(err error) error {
	if rErr, ok := err.(*oauth2.RetrieveError); ok && rErr.Response.StatusCode == http.StatusBadRequest {
		var resp Response
		if err = json.Unmarshal(rErr.Body, &resp); err != nil {
			return fmt.Errorf("fibapi: error parsing response: %w", err)
		}
		if resp.Error == oauthInvalidAuthorizationCodeResponseErrorMessage {
			return ErrInvalidAuthorizationCode
		} else {
			if resp.Error == "" {
				resp.Error = "(no error message)"
			}
			return fmt.Errorf("fibapi: error retrieving token: %s", resp.Error)
		}
	} else {
		return fmt.Errorf("fibapi: error retrieving token: %w", err)
	}
}
