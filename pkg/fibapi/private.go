package fibapi

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
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
}

// NewClient initializes a FIB API private client with the given OAuth token
func NewClient(token oauth2.Token) *PrivateClient {
	ctx := context.Background()
	ts := oauthConf.TokenSource(ctx, &token)
	client := oauth2.NewClient(ctx, ts)
	return &PrivateClient{client}
}

// GetUserInfo gets the user's basic information (username, first name and last name only)
func (c *PrivateClient) GetUserInfo() (userInfo UserInfo, err error) {
	body, _, err := c.request(http.MethodGet, userInfoURL)
	if err != nil {
		err = fmt.Errorf("error getting UserInfo: %w", err)
		return
	}

	if err = json.Unmarshal(body, &userInfo); err != nil {
		err = fmt.Errorf("error parsing UserInfo: %w\n%s", err, string(body))
	}
	return
}

// GetNotices gets the user's notices
func (c *PrivateClient) GetNotices() (notices []Notice, err error) {
	notices, _, err = c.GetNoticesWithDigest()
	return
}

// GetNoticesWithDigest gets the user's notices with the response body's hash digest
func (c *PrivateClient) GetNoticesWithDigest() (notices []Notice, digest string, err error) {
	body, _, err := c.request(http.MethodGet, noticesURL)
	if err != nil {
		err = fmt.Errorf("error getting Notices: %w", err)
		return
	}

	var resp NoticesResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		err = fmt.Errorf("error parsing Notices: %w\n%s", err, string(body))
		return
	}

	notices = resp.Results
	sort.Slice(notices, func(i, j int) bool {
		return notices[i].PublishedAt.Unix() < notices[j].PublishedAt.Unix()
	})
	digest = fmt.Sprintf("%08x", crc32.ChecksumIEEE(body))
	return
}

// GetNotice gets a specific notice with the given ID
func (c *PrivateClient) GetNotice(ID int32) (notice Notice, err error) {
	notices, err := c.GetNotices()
	if err != nil {
		err = fmt.Errorf("error getting Notices: %w", err)
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
		return nil, fmt.Errorf("error getting Subjects: %w", err)
	}

	var resp SubjectsResponse
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parsing Subjects: %w\n%s", err, string(body))
	}

	return resp.Results, nil
}

// RevokeToken revokes the user's OAuth token
func (c *PrivateClient) RevokeToken() (err error) {
	token, err := c.Client.Transport.(*oauth2.Transport).Source.Token()
	if err != nil {
		return
	}

	_, err = c.Client.PostForm(oAuthRevokeURL, url.Values{
		"client_id": {oauthConf.ClientID},
		"token":     {token.AccessToken},
	})
	if err != nil {
		err = fmt.Errorf("error revoking token: %w", err)
	}
	return
}

// GetAttachmentFile gets the given Attachment's bytes
// BE CAREFUL: some attachments posted on racó are copyright-protected and should not be stored nor accessed by third-parties
func (c *PrivateClient) GetAttachmentFile(a Attachment) ([]byte, error) {
	body, _, err := c.request(http.MethodGet, strings.TrimSuffix(a.URL, `.json`))
	if err != nil {
		return nil, fmt.Errorf("error getting Attachment: %w", err)
	}

	return body, nil
}

// request makes a request to Private FIB API using the given HTTP method and URL
func (c *PrivateClient) request(method, URL string) (body []byte, header http.Header, err error) {
	req, err := http.NewRequest(method, URL, nil)
	if err != nil {
		return
	}
	for k, v := range requestHeaders {
		req.Header.Set(k, v)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		// API error handling
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest {
			// token has expired or has been revoked on server
			err = ErrAuthorizationExpired
		} else {
			// TODO: handle more other errors
			err = ErrUnknown
		}
	}

	header = resp.Header
	return
}
