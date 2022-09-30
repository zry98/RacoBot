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

	"golang.org/x/oauth2"
)

// PrivateClient represents a FIB API private app client initialized with a token
// the token may expire, but the underlying client will try to refresh it in later API requests
type PrivateClient struct {
	*http.Client
	ctx context.Context
}

// NewClient initializes a FIB API private client with the given OAuth token
func NewClient(token oauth2.Token) *PrivateClient {
	ctx := context.Background()
	ts := oauthConf.TokenSource(ctx, &token)
	client := oauth2.NewClient(ctx, ts)
	return &PrivateClient{client, ctx}
}

// GetUserInfo gets the user's basic information (username, first name and last name only)
func (c *PrivateClient) GetUserInfo() (userInfo UserInfo, err error) {
	body, _, err := c.request(http.MethodGet, userInfoURL)
	if err != nil {
		return
	}
	if err = json.Unmarshal(body, &userInfo); err != nil {
		err = fmt.Errorf("fibapi: error parsing UserInfo: %w", err)
	}
	return
}

// GetNotices gets the user's notices
func (c *PrivateClient) GetNotices() (notices []Notice, err error) {
	return c.GetNoticesSince(0)
}

// GetNoticesSince gets the user's notices published since the given timestamp
func (c *PrivateClient) GetNoticesSince(timestamp int64) (notices []Notice, err error) {
	body, _, err := c.request(http.MethodGet, noticesURL)
	if err != nil {
		return
	}
	var resp NoticesResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		err = fmt.Errorf("fibapi: error parsing Notices: %w", err)
		return
	}

	notices = make([]Notice, 0, len(resp.Results))
	for _, n := range resp.Results {
		if n.PublishedAt.Unix() > timestamp {
			notices = append(notices, n)
		}
	}
	sort.Slice(notices, func(i, j int) bool {
		return notices[i].PublishedAt.Unix() < notices[j].PublishedAt.Unix()
	})
	return
}

// GetNotice gets a specific notice with the given ID
func (c *PrivateClient) GetNotice(ID int32) (notice Notice, err error) {
	notices, err := c.GetNotices()
	if err != nil {
		return
	}

	for _, n := range notices {
		if n.ID == ID {
			return n, nil
		}
	}
	err = ErrNoticeNotFound
	return
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

	_, err = c.Client.PostForm(oauthRevokeURL, url.Values{
		"client_id": {oauthConf.ClientID},
		"token":     {token.AccessToken},
	})
	if err != nil {
		return fmt.Errorf("fibapi: error revoking token: %w", err)
	}
	return nil
}

// GetAttachmentFile gets the given Attachment's bytes
// BE CAREFUL: some attachments posted on racó are copyright-protected and should not be stored nor accessed by third-parties
func (c *PrivateClient) GetAttachmentFile(a Attachment) (body []byte, err error) {
	body, _, err = c.request(http.MethodGet, strings.TrimSuffix(a.URL, `.json`))
	return
}

// request makes a request to Private FIB API using the given HTTP method and URL
func (c *PrivateClient) request(method, URL string) (body []byte, header http.Header, err error) {
	ctx, cancel := context.WithTimeout(c.ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, URL, nil)
	if err != nil {
		err = fmt.Errorf("fibapi: error creating request: %w", err)
		return
	}
	for k, v := range requestHeaders {
		req.Header.Set(k, v)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		if rErr, ok := err.(*url.Error).Err.(*oauth2.RetrieveError); ok { // API error, pass it to later handling
			resp = rErr.Response
			body = rErr.Body
			err = nil
		} else {
			err = fmt.Errorf("fibapi: error making request: %w", err)
			return
		}
	} else {
		defer resp.Body.Close()
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			err = fmt.Errorf("fibapi: error reading response body: %w", err)
			return
		}
	}

	if resp.StatusCode != http.StatusOK {
		// API error handling
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest {
			// token has expired or has been revoked on server
			err = ErrAuthorizationExpired
		} else {
			err = fmt.Errorf("fibapi: bad response (%d): %s", resp.StatusCode, string(body))
		}
	}

	header = resp.Header // return response header for future error handling
	return
}
