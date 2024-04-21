package bot

import (
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"RacoBot/internal/db"
	"RacoBot/pkg/fibapi"
)

// Client represents a FIB API client initialized with a Telegram userID
type Client struct {
	fibapi.PrivateClient
	db.User
}

// errors
var (
	ErrUserNotFound = errors.New("user not found")
	ErrInternal     = errors.New("internal error")
)

// NewClient initializes a FIB API private client with the given Telegram userID
// if that userID doesn't exist in the database, it will return nil and leave it for the later API caller to handle
// thus simplifies its usage to: `xxx, err := NewClient(userID).GetXXX()`
func NewClient(userID int64) *Client {
	user, err := db.GetUser(userID)
	if err != nil {
		if !errors.Is(err, db.ErrUserNotFound) {
			log.Errorf("failed to get user %d: %v", userID, err)
		}
		return nil
	}
	if user.AccessToken == "" || user.RefreshToken == "" {
		return nil
	}

	return &Client{
		*fibapi.NewClient(user.AccessToken, user.RefreshToken, user.TokenExpiry),
		user,
	}
}

// updateToken updates the user's FIB API OAuth token in database if it has been refreshed by the underlying fibapi.PrivateClient
// it should be called (and should be deferred) in every API caller
func (c *Client) updateToken() {
	newToken, err := c.PrivateClient.Transport.(*oauth2.Transport).Source.Token()
	if err != nil {
		err = fibapi.ProcessTokenError(err)
		if errors.Is(err, fibapi.ErrInvalidAuthorizationCode) {
			log.Errorf("user %d authorization has expired", c.User.ID)
			if e := db.DelUser(c.User.ID); e != nil {
				log.Errorf("failed to delete user %d: %v", c.User.ID, e)
			}
		} else {
			log.Errorf("failed to extract token from user %d: %v", c.User.ID, err)
		}
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
	log.Debugf("user %d token has been updated, new expiry: %s", c.User.ID, newToken.Expiry.Format(time.RFC3339))
}

// GetFullName gets the user's full name (as format of `${firstName} ${lastName}`)
func (c *Client) GetFullName() (string, error) {
	if c == nil {
		return "", ErrUserNotFound
	}
	defer c.updateToken()

	userInfo, err := c.PrivateClient.GetUserInfo()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s %s", userInfo.FirstName, userInfo.LastNames), nil
}

// GetNotices gets the user's notice messages
func (c *Client) GetNotices() ([]NoticeMessage, error) {
	if c == nil {
		return nil, ErrUserNotFound
	}
	defer c.updateToken()

	notices, err := c.PrivateClient.GetNotices()
	if err != nil {
		return nil, err
	}

	msgs := make([]NoticeMessage, 0, len(notices))
	for _, notice := range notices {
		msgs = append(msgs, NoticeMessage{notice, c.User, getNoticeLinkURL(notice)})
	}
	return msgs, nil
}

// GetNotice gets a specific notice message with the given ID
func (c *Client) GetNotice(ID int32) (NoticeMessage, error) {
	if c == nil {
		return NoticeMessage{}, ErrUserNotFound
	}
	defer c.updateToken()

	notice, err := c.PrivateClient.GetNotice(ID)
	if err != nil {
		return NoticeMessage{}, err
	}
	return NoticeMessage{notice, c.User, getNoticeLinkURL(notice)}, nil
}

// GetNewNotices gets the user's new notice messages
func (c *Client) GetNewNotices() ([]NoticeMessage, error) {
	if c == nil {
		return nil, ErrUserNotFound
	}
	defer c.updateToken()

	ns, err := c.PrivateClient.GetNoticesSince(c.User.LastNoticeTimestamp)
	if err != nil {
		return nil, err
	}
	defer func() { // save the last notice's timestamp to DB
		if len(ns) > 0 {
			c.User.LastNoticeTimestamp = ns[len(ns)-1].PublishedAt.Unix()
			if e := db.PutUser(c.User); e != nil {
				log.Errorf("failed to put user %d: %v", c.User.ID, e)
			}
		}
	}()

	var msgs []NoticeMessage
	if c.User.LastNoticeTimestamp != 0 { // if not a new user, send the new notices
		msgs = make([]NoticeMessage, 0, len(ns))
		for _, n := range ns {
			if n.PublishedAt.Unix() > c.User.LastNoticeTimestamp {
				msgs = append(msgs, NoticeMessage{n, c.User, getNoticeLinkURL(n)})
			}
		}
	}
	return msgs, nil
}

// Logout revokes the user's OAuth token and deletes it from the database
func (c *Client) Logout() error {
	if c == nil {
		return ErrUserNotFound
	}

	defer func() { // delete user from DB no matter what
		if e := db.DelUser(c.User.ID); e != nil {
			log.Errorf("failed to delete user %d: %v", c.User.ID, e)
		}
	}()
	if err := c.PrivateClient.RevokeToken(); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
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
