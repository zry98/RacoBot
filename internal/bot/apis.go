package bot

import (
	"errors"
	"fmt"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"RacoBot/internal/db"
	"RacoBot/pkg/fibapi"
)

// Client represents a FIB API client initialized with a Telegram UserID
type Client struct {
	User db.User
	fibapi.Client
}

// errors
var (
	UserNotFoundError = errors.New("user not found")
)

// NewClient initializes a FIB API client with the given Telegram UserID
// if that UserID doesn't exist in the database, it will return nil and leave it for the later API caller to handle
// thus simplifies its usage to: `xxx, err := NewClient(userID).GetXXX()`
func NewClient(userID int64) *Client {
	user, err := db.GetUser(userID)
	if err != nil || user.AccessToken == "" || user.RefreshToken == "" {
		return nil
	}

	return &Client{
		User: user,
		Client: *fibapi.NewClient(oauth2.Token{
			AccessToken:  user.AccessToken,
			RefreshToken: user.RefreshToken,
			Expiry:       time.Unix(user.TokenExpiry, 0),
			TokenType:    "Bearer",
		}),
	}
}

// updateToken updates the user's FIB API OAuth token in database if it has been refreshed by the underlying fibapi.Client
// it should be called (and should be deferred) in every API caller
func (c *Client) updateToken() {
	newToken, err := c.Client.Transport.(*oauth2.Transport).Source.Token()
	if err != nil {
		log.Error(err)
		return
	}

	if newToken.AccessToken != c.User.AccessToken {
		c.User.AccessToken = newToken.AccessToken
		c.User.RefreshToken = newToken.RefreshToken
		c.User.TokenExpiry = newToken.Expiry.Unix() - 60  // TODO: tune the precaution seconds
		if err = db.PutUser(c.User); err != nil {
			log.Error(err)
			return
		}
	}
}

// GetFullName gets the user's full name (as format of `${firstName} ${lastName}`)
func (c *Client) GetFullName() (fullName string, err error) {
	if c == nil {
		err = UserNotFoundError
		return
	}
	defer c.updateToken()

	res, err := c.Client.GetUserInfo()
	if err != nil {
		return
	}

	fullName = fmt.Sprintf("%s %s", res.FirstName, res.LastNames)
	return
}

// GetNotices gets the user's notice messages
func (c *Client) GetNotices() (ns []NoticeMessage, err error) {
	if c == nil {
		err = UserNotFoundError
		return
	}
	defer c.updateToken()

	res, err := c.Client.GetNotices()
	if err != nil {
		return
	}

	for _, n := range res {
		ns = append(ns, NoticeMessage{n, c.User})
	}
	return
}

// GetNotice gets a specific notice message with the given ID
func (c *Client) GetNotice(ID int64) (n NoticeMessage, err error) {
	if c == nil {
		err = UserNotFoundError
		return
	}
	defer c.updateToken()

	res, err := c.Client.GetNotice(ID)
	if err != nil {
		return
	}

	n = NoticeMessage{res, c.User}
	return
}

// GetNewNotices gets the user's new notice messages
func (c *Client) GetNewNotices() (ns []NoticeMessage, err error) {
	if c == nil {
		err = UserNotFoundError
		return
	}
	defer c.updateToken()

	res, noticesHash, err := c.Client.GetNoticesWithHash()
	if err != nil {
		return
	}

	if noticesHash == c.User.LastNoticesHash { // no change at all
		return
	}

	if len(res) == 0 { // no available notices (mostly due to the new semester not started)
		c.User.LastNoticesHash = noticesHash
		err = db.PutUser(c.User)
		return
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].PublishedAt.Before(res[j].PublishedAt.Time)
	})

	if c.User.LastNoticesHash != "" && c.User.LastNoticeTimestamp != 0 { // if not a new user
		for _, n := range res {
			if n.PublishedAt.Unix() > c.User.LastNoticeTimestamp {
				ns = append(ns, NoticeMessage{n, c.User})
			}
		}
	}

	c.User.LastNoticesHash = noticesHash
	c.User.LastNoticeTimestamp = res[len(res)-1].PublishedAt.Unix()
	err = db.PutUser(c.User)
	return
}

// Logout revokes the user's OAuth token and deletes it from the database
func (c *Client) Logout() (err error) {
	if c == nil {
		err = UserNotFoundError
		return
	}

	err = c.Client.RevokeToken()
	if err != nil {
		return
	}

	err = db.DeleteUser(c.User.ID)
	return
}
