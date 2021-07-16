package bot

import (
	"errors"
	"fmt"
	"sort"

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"RacoBot/internal/db"
	"RacoBot/pkg/fibapi"
)

// Client represents a FIB API client initialized with a Telegram UserID
type Client struct {
	userID int64
	token  *oauth2.Token
	fibapi.Client
}

// errors
var (
	TokenNotFoundError = errors.New("user token not found")
)

// NewClient initializes a FIB API client with the given Telegram UserID
// if that UserID doesn't exist in the database, it will return nil and leave it for the later API caller to handle
// thus simplifies its usage to: `xxx, err := NewClient(userID).GetXXX()`
func NewClient(userID int64) *Client {
	token, err := db.GetToken(userID)
	if err != nil || token == nil {
		return nil
	}

	return &Client{
		userID: userID,
		token:  token,
		Client: *fibapi.NewClient(*token),
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

	if newToken.AccessToken != c.token.AccessToken {
		err = db.PutToken(c.userID, newToken)
		if err != nil {
			log.Error(err)
			return
		}
	}
}

// GetFullName gets the user's full name (`${firstName} ${lastName}`)
func (c *Client) GetFullName() (fullName string, err error) {
	if c == nil {
		err = TokenNotFoundError
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
		err = TokenNotFoundError
		return
	}
	defer c.updateToken()

	res, err := c.Client.GetNotices()
	if err != nil {
		return
	}

	for _, n := range res {
		ns = append(ns, NoticeMessage{n})
	}
	return
}

// GetNotice gets a specific notice message with the given ID
func (c *Client) GetNotice(ID int64) (n NoticeMessage, err error) {
	if c == nil {
		err = TokenNotFoundError
		return
	}
	defer c.updateToken()

	res, err := c.Client.GetNotice(ID)
	if err != nil {
		return
	}

	n = NoticeMessage{res}
	return
}

// GetNewNotices gets the user's new notice messages
func (c *Client) GetNewNotices() (ns []NoticeMessage, err error) {
	if c == nil {
		err = TokenNotFoundError
		return
	}
	defer c.updateToken()

	lastState, err := db.GetLastState(c.userID)
	if err != nil {
		return
	}

	res, noticesHash, err := c.Client.GetNoticesWithHash()
	if err != nil {
		return
	}

	if noticesHash == lastState.NoticesHash { // no change at all
		return
	}

	if len(res) == 0 {
		err = db.PutLastState(c.userID, db.LastState{
			NoticesHash: noticesHash,
		})
		return
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].ModifiedAt.Unix() < res[j].ModifiedAt.Unix()
	})

	if lastState.NoticesHash != "" && lastState.NoticeTimestamp != 0 { // if not a new user
		for _, n := range res {
			if n.ModifiedAt.Unix() > lastState.NoticeTimestamp { // posted or modified later than the last state
				ns = append(ns, NoticeMessage{n})
			}
		}
	}

	err = db.PutLastState(c.userID, db.LastState{
		NoticesHash:     noticesHash,
		NoticeTimestamp: res[len(res)-1].ModifiedAt.Unix(),
	})
	return
}

// Logout revokes the user's OAuth token and deletes it from the database
func (c *Client) Logout() (err error) {
	if c == nil {
		err = TokenNotFoundError
		return
	}

	err = c.Client.RevokeToken()
	if err != nil {
		return
	}

	err = db.DeleteToken(c.userID)
	return
}
