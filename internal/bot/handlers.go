package bot

import (
	"sort"
	"strconv"

	log "github.com/sirupsen/logrus"
	tb "gopkg.in/tucnak/telebot.v3"

	"RacoBot/internal/db"
)

// on command `/start`
// start replies a `/login` message
func start(c tb.Context) (err error) {
	return c.Send("/login to authorize Racó Bot") // TODO: make it nicer
}

// on command `/login`
// login replies a FIB API OAuth authorization link message for the user
func login(c tb.Context) (err error) {
	userID := int64(c.Sender().ID)
	token, err := db.GetToken(userID)
	if err != nil && err != db.TokenNotFoundError {
		log.Error(err)
		return
	}
	if token != nil && token.AccessToken != "" && token.RefreshToken != "" {
		// already logged-in user
		return c.Send("Already logged-in, check /whoami; or, /logout to revoke the authorization")
	}

	// new user
	session, err := db.NewLoginSession(userID)
	if err != nil {
		log.Error(err)
		return c.Send(InternalErrorMessage)
	}

	m, err := b.sendMessage(userID, LoginLinkMessage{session})
	if err != nil {
		log.Error(err)
		return c.Send(InternalErrorMessage)
	}
	session.MessageID = int64(m.ID)

	return db.PutLoginSession(session)
}

// on command `/whoami`
// whoami replies the user's full name
func whoami(c tb.Context) (err error) {
	userID := int64(c.Sender().ID)
	reply, err := NewClient(userID).GetFullName()
	if err != nil {
		return
	}

	return c.Send(reply)
}

// on command `/logout`
// logout revokes the user's FIB API OAuth token and deletes it from the database
func logout(c tb.Context) (err error) {
	userID := int64(c.Sender().ID)
	err = NewClient(userID).Logout()
	if err != nil {
		return
	}

	return c.Send("Logout successful")
}

// on command `/debug \d+`
// debug replies notice message with the given ID in payload
func debug(c tb.Context) (err error) {
	userID := int64(c.Sender().ID)
	messageID, err := strconv.ParseInt(c.Message().Payload, 10, 64)
	if err != nil {
		return c.Reply("Invalid payload")
	}

	notice, err := NewClient(userID).GetNotice(messageID)
	if err != nil {
		return
	}
	if notice.ID == 0 {
		// maybe don't exist, or not available to the user
		return c.Send("Notice unavailable")
	}

	return c.Send(notice.String(), sendNoticeMessageOption)
}

// on command `/test`
// test replies the latest one notice message
func test(c tb.Context) (err error) {
	userID := int64(c.Sender().ID)
	client := NewClient(userID)
	if client == nil {
		return TokenNotFoundError
	}
	notices, noticesHash, err := client.GetNoticesWithHash()
	if err != nil {
		return
	}
	if len(notices) == 0 {
		err = db.PutLastState(userID, db.LastState{
			NoticesHash: noticesHash,
		})
		if err != nil {
			return
		}

		return c.Send(NoNoticesAvailableMessage, SendHTMLMessageOption)
	}

	sort.Slice(notices, func(i, j int) bool {
		return notices[i].ModifiedAt.Unix() < notices[j].ModifiedAt.Unix()
	})

	err = db.PutLastState(userID, db.LastState{
		NoticesHash:     noticesHash,
		NoticeTimestamp: notices[len(notices)-1].ModifiedAt.Unix(),
	})
	if err != nil {
		return
	}

	return c.Send(NoticeMessage{notices[len(notices)-1]}.String(), sendNoticeMessageOption)
}
