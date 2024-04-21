package fibapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

var privateClient *http.Client

// PrivateClient represents a FIB API private app client initialized with a token, usually representing a user,
// the token may expire, but the underlying OAuth client will try to refresh it in later API requests
type PrivateClient struct {
	*http.Client
	ctx context.Context
}

// NewClient initializes a FIB API private client with the given OAuth token
func NewClient(accessToken string, refreshToken string, expiry int64) *PrivateClient {
	token := oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Expiry:       time.Unix(expiry, 0),
		TokenType:    "Bearer",
	}
	return NewClientFromToken(&token)
}

func NewClientFromToken(token *oauth2.Token) *PrivateClient {
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, privateClient)
	client := oauth2.NewClient(ctx, oauthConf.TokenSource(ctx, token))
	return &PrivateClient{client, ctx}
}

// GetUserInfo gets the user's basic information (username, first name and last name only)
func (c *PrivateClient) GetUserInfo() (UserInfo, error) {
	body, _, err := c.request(http.MethodGet, userInfoURL)
	if err != nil {
		return UserInfo{}, err
	}

	var userInfo UserInfo
	if err = json.Unmarshal(body, &userInfo); err != nil {
		return UserInfo{}, fmt.Errorf("fibapi: error parsing UserInfo: %w", err)
	}
	return userInfo, nil
}

// GetNotices gets the user's notices
func (c *PrivateClient) GetNotices() ([]Notice, error) {
	return c.GetNoticesSince(0)
}

// GetNoticesSince gets the user's notices published since the given timestamp
func (c *PrivateClient) GetNoticesSince(timestamp int64) ([]Notice, error) {
	body, _, err := c.request(http.MethodGet, noticesURL)
	if err != nil {
		return nil, err
	}
	var resp NoticesResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("fibapi: error parsing Notices: %w", err)
	}

	ns := make([]Notice, 0, len(resp.Results))
	for _, n := range resp.Results {
		if n.PublishedAt.Unix() > timestamp {
			ns = append(ns, n)
		}
	}
	sort.Slice(ns, func(i, j int) bool {
		// sort orders: PublishedAt, SubjectCode, Title
		if ns[i].PublishedAt.Unix() == ns[j].PublishedAt.Unix() {
			if ns[i].SubjectCode == ns[j].SubjectCode {
				return ns[i].Title < ns[j].Title // in line with raco web
			}
			return ns[i].SubjectCode < ns[j].SubjectCode
		}
		return ns[i].PublishedAt.Unix() < ns[j].PublishedAt.Unix()
	})
	return ns, nil
}

// GetNotice gets a specific notice with the given ID
func (c *PrivateClient) GetNotice(ID int32) (Notice, error) {
	notices, err := c.GetNotices()
	if err != nil {
		return Notice{}, err
	}

	for _, n := range notices {
		if n.ID == ID {
			return n, nil
		}
	}
	return Notice{}, ErrNoticeNotFound
}

// GetSubjects gets the user's subjects
func (c *PrivateClient) GetSubjects() ([]Subject, error) {
	body, _, err := c.request(http.MethodGet, subjectsURL)
	if err != nil {
		return nil, err
	}
	var resp SubjectsResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("fibapi: error parsing Subjects: %w", err)
	}
	return resp.Results, nil
}

// RevokeToken revokes the user's OAuth token
func (c *PrivateClient) RevokeToken() error {
	token, err := c.Client.Transport.(*oauth2.Transport).Source.Token()
	if err != nil {
		return fmt.Errorf("fibapi: error extracting token: %w", err)
	}

	if _, err = c.Client.PostForm(oauthRevokeURL, url.Values{
		"client_id": {oauthConf.ClientID},
		"token":     {token.AccessToken},
	}); err != nil {
		return fmt.Errorf("fibapi: error revoking token: %w", err)
	}
	return nil
}

// GetAttachmentFile gets the given Attachment's bytes
// BE CAREFUL: some attachments posted on racó are copyright-protected and should not be stored nor accessed by third-parties
func (c *PrivateClient) GetAttachmentFile(a Attachment) ([]byte, error) {
	body, _, err := c.request(http.MethodGet, strings.TrimSuffix(a.URL, `.json`))
	return body, err
}

// request makes a request to Private FIB API using the given HTTP method and URL
func (c *PrivateClient) request(method, URL string) ([]byte, http.Header, error) {
	ctx, cancel := context.WithTimeout(c.ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, URL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("fibapi: error creating request: %w", err)
	}
	req.Header = baseReqHeader.Clone()

	var body []byte
	resp, err := c.Client.Do(req)
	if err != nil {
		if rErr, ok := err.(*url.Error).Err.(*oauth2.RetrieveError); ok { // API error, pass it to later handling
			resp = rErr.Response
			body = rErr.Body
		} else {
			return nil, nil, fmt.Errorf("fibapi: error making request: %w", err)
		}
	} else {
		defer resp.Body.Close()
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("fibapi: error reading response body: %w", err)
		}
	}

	if resp.StatusCode != http.StatusOK {
		// API error handling
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest {
			// token has expired or has been revoked on server
			return body, resp.Header, ErrAuthorizationExpired
		} else {
			return body, resp.Header, fmt.Errorf("fibapi: bad response (HTTP %d): %s", resp.StatusCode, string(body))
		}
	}

	return body, resp.Header, nil // return with response header for future error handling
}
