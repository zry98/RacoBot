package fibapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

// Client represents a FIB API client initialized with a token
// the token may be expired, but the underlying client will try to refresh it in later API requests
type Client struct {
	*http.Client
}

// NewClient initializes a FIB API client with the given OAuth token
func NewClient(token oauth2.Token) *Client {
	ctx := context.Background()
	ts := oauthConf.TokenSource(ctx, &token)
	client := oauth2.NewClient(ctx, ts)
	return &Client{client}
}

// GetUserInfo gets the user's basic information (username, first name and last name only)
func (c *Client) GetUserInfo() (userInfo UserInfo, err error) {
	body, _, err := c.request(http.MethodGet, UserInfoURL)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &userInfo)
	if err != nil {
		err = fmt.Errorf("error parsing UserInfo response: %s\n%s", string(body), err)
	}
	return
}

// GetNotices gets the user's notices
func (c *Client) GetNotices() ([]Notice, error) {
	body, _, err := c.request(http.MethodGet, NoticesURL)
	if err != nil {
		return nil, err
	}

	var notices Notices
	err = json.Unmarshal(body, &notices)
	if err != nil {
		err = fmt.Errorf("error parsing Notices response: %s\n%s", string(body), err)
	}
	return notices.Results, nil
}

// GetNotice gets a specific notice with the given ID
// it returns an empty Notice with ID==0 when it's not available to the user
func (c *Client) GetNotice(ID int64) (notice Notice, err error) {
	notices, err := c.GetNotices()
	if err != nil {
		return
	}

	for _, n := range notices {
		if n.ID == ID {
			return n, nil
		}
	}
	return
}

// GetSubjects gets the user's subjects
func (c *Client) GetSubjects() ([]Subject, error) {
	body, _, err := c.request(http.MethodGet, SubjectsURL)
	if err != nil {
		return nil, err
	}

	var subjects Subjects
	err = json.Unmarshal(body, &subjects)
	if err != nil {
		err = fmt.Errorf("error parsing Subjects response: %s\n%s", string(body), err)
	}
	return subjects.Results, nil
}

const RevokeTokenRequestMimeType = "application/x-www-form-urlencoded"

// RevokeToken revokes the user's OAuth token
func (c *Client) RevokeToken() (err error) {
	token, err := c.Client.Transport.(*oauth2.Transport).Source.Token()
	if err != nil {
		return
	}

	params := url.Values{
		"client_id": {oauthConf.ClientID},
		"token":     {token.AccessToken},
	}
	_, err = c.Client.Post(OAuthRevokeURL, RevokeTokenRequestMimeType, strings.NewReader(params.Encode()))
	return
}

// GetAttachmentFileData gets the given attachment's file data
// Be careful: some attachments posted on Racó has copyright and should not be stored nor accessed by third-parties
func (c *Client) GetAttachmentFileData(a Attachment) (data io.Reader, err error) {
	// FIXME: waiting for the bug in FIB API to be fixed
	// the bug: attachments' URLs will have the `.json` suffixes if requested endpoint was `/jo/avisos.json`
	body, _, err := c.request(http.MethodGet, strings.TrimSuffix(a.URL, `.json`))
	if err != nil {
		return
	}

	data = bytes.NewReader(body)
	return
}

// request requests to FIB API with the given URL
func (c *Client) request(method, URL string) (body []byte, header http.Header, err error) {
	req, err := http.NewRequest(method, URL, nil)
	if err != nil {
		return
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	header = resp.Header
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest {
			// token has been revoked on server
			err = AuthorizationExpiredError
		} else {
			// TODO: handle more other errors
			err = UnknownError
		}
	}
	return
}
