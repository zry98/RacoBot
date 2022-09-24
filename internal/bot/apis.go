package bot

import (
	"fmt"
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
	ErrUserNotFound = fmt.Errorf("user not found")
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
		log.Errorf("failed to extract token from user %d: %v", c.User.ID, err)
		return
	}

	if newToken.AccessToken == c.User.AccessToken && newToken.RefreshToken == c.User.RefreshToken {
		return
	}
	c.User.AccessToken = newToken.AccessToken
	c.User.RefreshToken = newToken.RefreshToken
	c.User.TokenExpiry = newToken.Expiry.Unix() - 10*60 // expire it 10 minutes in advance
	if err = db.PutUser(c.User); err != nil {
		log.Errorf("failed to put user %d: %v", c.User.ID, err)
		return
	}
}

// GetFullName gets the user's full name (as format of `${firstName} ${lastName}`)
func (c *Client) GetFullName() (fullName string, err error) {
	if c == nil {
		err = ErrUserNotFound
		return
	}
	defer c.updateToken()

	userInfo, err := c.PrivateClient.GetUserInfo()
	if err != nil {
		return
	}

	fullName = fmt.Sprintf("%s %s", userInfo.FirstName, userInfo.LastNames)
	return
}

// GetNotices gets the user's notice messages
func (c *Client) GetNotices() (messages []NoticeMessage, err error) {
	if c == nil {
		err = ErrUserNotFound
		return
	}
	defer c.updateToken()

	notices, err := c.PrivateClient.GetNotices()
	if err != nil {
		return
	}

	messages = make([]NoticeMessage, 0, len(notices))
	for _, notice := range notices {
		messages = append(messages, NoticeMessage{notice, c.User, getNoticeLinkURL(notice)})
	}
	return
}

// GetNotice gets a specific notice message with the given ID
func (c *Client) GetNotice(ID int32) (message NoticeMessage, err error) {
	if c == nil {
		err = ErrUserNotFound
		return
	}
	defer c.updateToken()

	notice, err := c.PrivateClient.GetNotice(ID)
	if err != nil {
		return
	}

	message = NoticeMessage{notice, c.User, getNoticeLinkURL(notice)}
	return
}

// GetNewNotices gets the user's new notice messages
func (c *Client) GetNewNotices() (messages []NoticeMessage, err error) {
	if c == nil {
		err = ErrUserNotFound
		return
	}
	defer c.updateToken()

	notices, noticesHash, err := c.PrivateClient.GetNoticesWithDigest()
	if err != nil {
		return
	}

	if noticesHash == c.User.LastNoticesDigest { // no change at all
		return
	}

	if len(notices) == 0 { // no available notices (mostly due to the new semester not started)
		c.User.LastNoticesDigest = noticesHash
		if err = db.PutUser(c.User); err != nil {
			err = fmt.Errorf("failed to put user: %w", err)
		}
		return
	}

	if c.User.LastNoticesDigest != "" && c.User.LastNoticeTimestamp != 0 { // if not a new user
		messages = make([]NoticeMessage, 0, len(notices))
		for _, n := range notices {
			if n.PublishedAt.Unix() > c.User.LastNoticeTimestamp {
				messages = append(messages, NoticeMessage{n, c.User, getNoticeLinkURL(n)})
			}
		}
	}

	c.User.LastNoticesDigest = noticesHash
	c.User.LastNoticeTimestamp = notices[len(notices)-1].PublishedAt.Unix()
	if err = db.PutUser(c.User); err != nil {
		err = fmt.Errorf("failed to put user: %w", err)
	}
	return
}

// Logout revokes the user's OAuth token and deletes it from the database
func (c *Client) Logout() error {
	if c == nil {
		return ErrUserNotFound
	}

	if err := c.PrivateClient.RevokeToken(); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	if err := db.DeleteUser(c.User.ID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

func getNoticeLinkURL(n fibapi.Notice) string {
	if strings.HasPrefix(n.SubjectCode, "#") {
		// special banner notice, not viewable on /avisos/veure.jsp
		return fmt.Sprintf("%s/#avis-%d", racoBaseURL, n.ID)
	}

	code, err := db.GetSubjectUPCCode(n.SubjectCode)
	if err != nil {
		if err == db.ErrSubjectNotFound {
			// not found in DB, try to get the code from FIB API
			subject, e := fibapi.GetPublicSubject(n.SubjectCode)
			if e != nil {
				log.Errorf("failed to get UPC code of %s from API: %v", n.SubjectCode, e)
				return racoBaseURL
			}

			code = subject.UPCCode
			// save it to DB by the way
			if e = db.PutSubjectUPCCode(n.SubjectCode, subject.UPCCode); e != nil {
				log.Errorf("failed to put UPC code of %s: %v", n.SubjectCode, e)
			}
		} else {
			// DB error
			log.Errorf("failed to get UPC code of %s: %v", n.SubjectCode, err)
			return racoBaseURL
		}
	}
	return fmt.Sprintf(racoNoticeURLTemplate, code, n.ID)
}
