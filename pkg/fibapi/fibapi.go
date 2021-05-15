/*
Package fibapi implements a simple library for getting information from the FIB API (https://api.fib.upc.edu/).
*/
package fibapi

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"
)

// FIBAPIConfig represents a configuration for FIB API OAuth application
type FIBAPIConfig struct {
	OAuthClientID     string `toml:"oauth_client_id"`
	OAuthClientSecret string `toml:"oauth_client_secret"`
	OAuthRedirectURI  string `toml:"oauth_redirect_URI"`
}

var oauthConf *oauth2.Config

// Init initializes the FIB API OAuth configuration instance
func Init(config FIBAPIConfig) {
	oauthConf = &oauth2.Config{
		ClientID:     config.OAuthClientID,
		ClientSecret: config.OAuthClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   OAuthAuthURL,
			TokenURL:  OAuthTokenURL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
		RedirectURL: config.OAuthRedirectURI,
		Scopes:      []string{"read"},
	}
}

// NewAuthorizationURL generates an authorization URL with the given state
func NewAuthorizationURL(state string) string {
	return oauthConf.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// Authorize tries to retrieve OAuth token with the given Authorization Code
func Authorize(authorizationCode string) (token *oauth2.Token, userInfo UserInfo, err error) {
	ctx := context.Background() // TODO: use context
	token, err = oauthConf.Exchange(ctx, authorizationCode)
	if err != nil {
		if errData, ok := err.(*oauth2.RetrieveError); ok && errData.Response.StatusCode == http.StatusBadRequest &&
			string(errData.Body) == OAuthInvalidAuthorizationCodeResponse {
			err = InvalidAuthorizationCodeError
		}
		return
	}

	// try to make an API call to get UserInfo, thus can check if the retrieved token is really valid
	userInfo, err = NewClient(*token).GetUserInfo()
	if err != nil {
		return
	}
	if userInfo.Username == "" {
		err = InvalidAuthorizationCodeError
	}
	return
}
