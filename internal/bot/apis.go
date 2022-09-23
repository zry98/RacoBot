package bot

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"RacoBot/internal/db"
	"RacoBot/pkg/fibapi"
)

// Client represents a FIB API client initialized with a Telegram UserID
type Client struct {
	fibapi.PrivateClient
	db.User
}

// errors
var (
	ErrUserNotFound = errors.New("user not found")
)

// NewClient initializes a FIB API private client with the given Telegram UserID
// if that UserID doesn't exist in the database, it will return nil and leave it for the later API caller to handle
// thus simplifies its usage to: `xxx, err := NewClient(userID).GetXXX()`
func NewClient(userID int64) *Client {
	user, err := db.GetUser(userID)
	if err != nil || user.AccessToken == "" || user.RefreshToken == "" {
		return nil
	}

	return &Client{
		*fibapi.NewClient(oauth2.Token{
			AccessToken:  user.AccessToken,
			RefreshToken: user.RefreshToken,
			Expiry:       time.Unix(user.TokenExpiry, 0),
			TokenType:    "Bearer",
		}),
		user,
	}
}

// updateToken updates the user's FIB API OAuth token in database if it has been refreshed by the underlying fibapi.PrivateClient
// it should be called (and should be deferred) in every API caller
func (c *Client) updateToken() {
	newToken, err := c.PrivateClient.Transport.(*oauth2.Transport).Source.Token()
	if err != nil {
		log.Error(err)
		return
	}

	if newToken.AccessToken != c.User.AccessToken {
		c.User.AccessToken = newToken.AccessToken
		c.User.RefreshToken = newToken.RefreshToken
		c.User.TokenExpiry = newToken.Expiry.Unix() - 10*60 // expire it 10 minutes in advance
		if err = db.PutUser(c.User); err != nil {
			log.Error(err)
			return
		}
	}
}

// GetFullName gets the user's full name (as format of `${firstName} ${lastName}`)
func (c *Client) GetFullName() (fullName string, err error) {
	if c == nil {
		err = ErrUserNotFound
		return
	}
	defer c.updateToken()

	res, err := c.PrivateClient.GetUserInfo()
	if err != nil {
		return
	}

	fullName = fmt.Sprintf("%s %s", res.FirstName, res.LastNames)
	return
}

// GetNotices gets the user's notice messages
func (c *Client) GetNotices() (ns []NoticeMessage, err error) {
	if c == nil {
		err = ErrUserNotFound
		return
	}
	defer c.updateToken()

	res, err := c.PrivateClient.GetNotices()
	if err != nil {
		return
	}

	for _, n := range res {
		ns = append(ns, NoticeMessage{n, c.User, getNoticeLinkURL(n)})
	}
	return
}

// GetNotice gets a specific notice message with the given ID
func (c *Client) GetNotice(ID int32) (n NoticeMessage, err error) {
	if c == nil {
		err = ErrUserNotFound
		return
	}
	defer c.updateToken()

	res, err := c.PrivateClient.GetNotice(ID)
	if err != nil {
		return
	}

	n = NoticeMessage{res, c.User, getNoticeLinkURL(res)}
	return
}

// GetNewNotices gets the user's new notice messages
func (c *Client) GetNewNotices() (ns []NoticeMessage, err error) {
	if c == nil {
		err = ErrUserNotFound
		return
	}
	defer c.updateToken()

	res, noticesHash, err := c.PrivateClient.GetNoticesWithHash()
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
				ns = append(ns, NoticeMessage{n, c.User, getNoticeLinkURL(n)})
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
		err = ErrUserNotFound
		return
	}

	err = c.PrivateClient.RevokeToken()
	if err != nil {
		return
	}

	err = db.DeleteUser(c.User.ID)
	return
}

func getNoticeLinkURL(n fibapi.Notice) string {
	if strings.HasPrefix(n.SubjectCode, "#") { // special banner notice, not viewable on /avisos/veure.jsp
		return fmt.Sprintf("%s/#avis-%d", racoBaseURL, n.ID)
	}

	code, err := db.GetSubjectUPCCode(n.SubjectCode)
	if err == db.ErrSubjectNotFound {
		var subject fibapi.PublicSubject
		subject, err = fibapi.GetPublicSubject(n.SubjectCode)
		if err != nil {
			log.Errorf("failed to get subject %s's UPC code: %s", n.SubjectCode, err)
			return racoBaseURL
		}
		code = subject.UPCCode

		err = db.PutSubjectUPCCode(n.SubjectCode, subject.UPCCode)
		if err != nil {
			log.Errorf("failed to put subject %s's UPC code %d to DB: %s", n.SubjectCode, subject.UPCCode, err)
		}
	}
	return fmt.Sprintf(racoNoticeURLTemplate, code, n.ID)
}
