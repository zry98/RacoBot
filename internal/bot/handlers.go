package bot

import (
	"RacoBot/internal/ratelimiters"
	"RacoBot/pkg/fibapi"
	"sort"
	"strconv"

	log "github.com/sirupsen/logrus"
	tb "gopkg.in/tucnak/telebot.v3"

	"RacoBot/internal/db"
	"RacoBot/internal/locales"
)

// on command `/start`
// start replies a `/login` message
func start(c tb.Context) (err error) {
	return c.Send(locales.Get(c.Sender().LanguageCode).StartMessage) // TODO: make it nicer
}

// on command `/login`
// login replies a FIB API OAuth authorization link message for the User
func login(c tb.Context) (err error) {
	userID := int64(c.Sender().ID)
	if !ratelimiters.LoginCommandAllowed(userID) {
		log.WithField("uid", userID).Info("login command rate limited")
		return
	}

	user, err := db.GetUser(userID)
	if err != nil && err != db.UserNotFoundError {
		log.Error(err)
		return
	}
	if err == nil && user.AccessToken != "" && user.RefreshToken != "" {
		// already logged-in User
		return c.Send(locales.Get(user.LanguageCode).AlreadyLoggedInMessage)
	}

	// new User
	session, err := db.NewLoginSession(userID, c.Sender().LanguageCode)
	if err != nil {
		log.Error(err)
		return c.Send(locales.Get(c.Sender().LanguageCode).InternalErrorMessage)
	}

	loginLinkMessage, err := b.sendMessage(userID, LoginLinkMessage{session})
	if err != nil {
		log.Error(err)
		return c.Send(locales.Get(c.Sender().LanguageCode).InternalErrorMessage)
	}
	session.LoginLinkMessageID = int64(loginLinkMessage.ID)

	return db.PutLoginSession(session)
}

// on command `/whoami`
// whoami replies the User's full name
func whoami(c tb.Context) (err error) {
	userID := int64(c.Sender().ID)
	reply, err := NewClient(userID).GetFullName()
	if err != nil {
		return
	}

	return c.Send(reply)
}

// on command `/logout`
// logout revokes the User's FIB API OAuth token and deletes it from the database
func logout(c tb.Context) (err error) {
	userID := int64(c.Sender().ID)
	client := NewClient(userID)
	if client == nil {
		return UserNotFoundError
	}
	err = client.Logout()
	if err != nil {
		return
	}

	return c.Send(locales.Get(client.User.LanguageCode).LogoutSucceededMessage)
}

// on command `/debug \d+`
// debug replies notice message with the given ID in payload
func debug(c tb.Context) (err error) {
	userID := int64(c.Sender().ID)
	noticeID, err := strconv.ParseInt(c.Message().Payload, 10, 64)
	if err != nil {
		return c.Reply("Invalid payload")
	}

	client := NewClient(userID)
	if client == nil {
		return UserNotFoundError
	}
	notice, err := client.GetNotice(noticeID)
	if err == fibapi.NoticeNotFoundError || (err == nil && notice.ID == 0) {
		// maybe don't exist, or not available to the User
		return c.Send(locales.Get(client.User.LanguageCode).NoticeUnavailableErrorMessage, SendHTMLMessageOption)
	}
	if err != nil {
		return
	}

	return c.Send(notice.String(), sendNoticeMessageOption)
}

// on command `/test`
// test replies the latest one notice message
func test(c tb.Context) (err error) {
	userID := int64(c.Sender().ID)
	client := NewClient(userID)
	if client == nil {
		return UserNotFoundError
	}
	notices, noticesHash, err := client.GetNoticesWithHash()
	if err != nil {
		return
	}
	if len(notices) == 0 {
		client.User.LastNoticesHash = noticesHash
		err = db.PutUser(client.User)
		if err != nil {
			return
		}

		return c.Send(locales.Get(client.User.LanguageCode).NoNoticesAvailableErrorMessage, SendHTMLMessageOption)
	}

	sort.Slice(notices, func(i, j int) bool {
		return notices[i].ModifiedAt.Unix() < notices[j].ModifiedAt.Unix()
	})

	client.User.LastNoticesHash = noticesHash
	client.User.LastNoticeTimestamp = notices[len(notices)-1].ModifiedAt.Unix()
	if err = db.PutUser(client.User); err != nil {
		return
	}

	return c.Send(NoticeMessage{notices[len(notices)-1], client.User}.String(), sendNoticeMessageOption)
}
