package bot

import (
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	tb "gopkg.in/telebot.v3"

	"RacoBot/internal/db"
	rl "RacoBot/internal/db/ratelimiters"
	"RacoBot/internal/locales"
	"RacoBot/pkg/fibapi"
)

// on command `/start`
// start replies with a `/login` message
func start(c tb.Context) (err error) {
	return c.Send(locales.Get(c.Sender().LanguageCode).StartMessage) // TODO: make it nicer
}

// on command `/login`
// login replies with a FIB API OAuth authorization link message for the user
func login(c tb.Context) (err error) {
	userID := c.Sender().ID
	if !rl.LoginCommandAllowed(userID) {
		log.WithField("UID", userID).Info("login command rate-limited")
		return
	}

	user, err := db.GetUser(userID)
	if err != nil && err != db.ErrUserNotFound {
		log.Error(err)
		return
	}
	if err == nil && user.AccessToken != "" && user.RefreshToken != "" {
		// already logged-in user
		return c.Send(locales.Get(user.LanguageCode).AlreadyLoggedInMessage)
	}

	// new user
	session, err := db.NewLoginSession(userID, c.Sender().LanguageCode)
	if err != nil {
		log.Error(err)
		return
	}

	loginLinkMessage, err := b.Send(&tb.Chat{ID: userID}, &LoginLinkMessage{session})
	if err != nil {
		log.Error(err)
		return
	}
	session.LoginLinkMessageID = int64(loginLinkMessage.ID)

	if err = db.PutLoginSession(session); err != nil {
		log.Error(err)
		return
	}
	return nil
}

// on command `/whoami`
// whoami replies with the user's full name
func whoami(c tb.Context) (err error) {
	fullName, err := NewClient(c.Sender().ID).GetFullName()
	if err != nil {
		if err == ErrUserNotFound {
			return
		}
		log.Errorf("failed to get full name of user %d: %v", c.Sender().ID, err)
		return
	}
	return c.Send(fullName)
}

// on command `/logout`
// logout revokes the user's FIB API OAuth token and deletes it from the database
func logout(c tb.Context) (err error) {
	client := NewClient(c.Sender().ID)
	if client == nil {
		return ErrUserNotFound
	}
	if err = client.Logout(); err != nil {
		log.Errorf("failed to logout user %d: %v", c.Sender().ID, err)
		return c.Send(&ErrorMessage{locales.Get(client.User.LanguageCode).LogoutFailedMessage})
	}
	return c.Send(locales.Get(client.User.LanguageCode).LogoutSucceededMessage)
}

// on command `/debug <noticeID>`
// debug replies with a notice with the given ID in payload
func debug(c tb.Context) (err error) {
	payload := c.Message().Payload
	if payload == "" {
		return
	}
	noticeID, err := strconv.ParseInt(payload, 10, 32)
	if err != nil {
		log.Debugf("failed to parse notice ID %s: %v", payload, err)
		return c.Reply("Invalid payload (/debug <noticeID>)")
	}

	client := NewClient(c.Sender().ID)
	if client == nil {
		return ErrUserNotFound
	}
	notice, err := client.GetNotice(int32(noticeID))
	if err == fibapi.ErrNoticeNotFound || (err == nil && notice.ID == 0) {
		// notice doesn't exist or isn't available to the user
		return c.Send(&ErrorMessage{locales.Get(client.User.LanguageCode).NoticeUnavailableErrorMessage})
	}
	if err != nil {
		log.Errorf("failed to get notice %d: %v", noticeID, err)
		return
	}
	return c.Send(&notice)
}

// on command `/test`
// test replies with the user's latest one notice
func test(c tb.Context) (err error) {
	client := NewClient(c.Sender().ID)
	if client == nil {
		return ErrUserNotFound
	}
	notices, digest, err := client.GetNoticesWithDigest()
	if err != nil {
		log.Errorf("failed to get notices of user %d: %v", c.Sender().ID, err)
		return
	}
	defer func() { // save states to DB by the way
		client.User.LastNoticesDigest = digest
		if len(notices) > 0 {
			client.User.LastNoticeTimestamp = notices[len(notices)-1].PublishedAt.Unix()
		}
		if err = db.PutUser(client.User); err != nil {
			log.Errorf("failed to put user %d: %v", c.Sender().ID, err)
		}
	}()

	if len(notices) == 0 {
		return c.Send(&ErrorMessage{locales.Get(client.User.LanguageCode).NoAvailableNoticesErrorMessage})
	}
	latestNotice := notices[len(notices)-1]
	return c.Send(&NoticeMessage{latestNotice, client.User, getNoticeLinkURL(latestNotice)})
}

// on command `/lang` and on callbacks &setLanguageButtonEN, &setLanguageButtonES, &setLanguageButtonCA
// setPreferredLanguage replies with the menu of supported languages for the user to select from on command,
// or sets the user's preferred language when on callbacks
func setPreferredLanguage(c tb.Context) (err error) {
	user, e := db.GetUser(c.Sender().ID)
	if e != nil {
		if e == db.ErrUserNotFound {
			return
		}
		log.Fatalf("failed to get user %d: %v", c.Sender().ID, e)
		return
	}

	// on command `/lang`, show the menu for selecting language
	if c.Callback() == nil {
		return c.Reply(locales.Get(user.LanguageCode).SelectPreferredLanguageMenuText, setLanguageMenu)
	}

	// on callbacks, set the user's preferred language with the given button data
	langCode := c.Callback().Unique
	if langCode == "" || (langCode != "en" && langCode != "es" && langCode != "ca") {
		return c.Reply(&ErrorMessage{locales.Get(c.Sender().LanguageCode).LanguageUnavailableErrorMessage})
	}
	user.LanguageCode = langCode
	if err = db.PutUser(user); err != nil {
		log.Errorf("failed to put user %d: %v", c.Sender().ID, err)
		return
	}
	return c.Edit(locales.Get(langCode).PreferredLanguageSetMessage)
}

// on command `/announce`
// publishAnnouncement publishes and pins the given announcement to all users in database
func publishAnnouncement(c tb.Context) (err error) {
	m := AnnouncementMessage{
		Text: strings.ReplaceAll(c.Message().Payload, "<br>", "\n"),
	}

	go func(announcement AnnouncementMessage) {
		logger := log.WithField("job", "PublishAnnouncement")
		count := 0

		userIDs, e := db.GetAllUserIDs()
		if e != nil {
			logger.Errorf("failed to get all user IDs: %v", e)
			return
		}
		logger.Infof("found %d users", len(userIDs))

		startTime := time.Now()
		for _, userID := range userIDs {
			if _, e = b.Send(tb.ChatID(userID), &announcement); e != nil {
				logger.Errorf("failed to send announcement to user %d: %v", userID, e)
				continue
			}
			count++
		}

		logger.Infof("sent announcement to %d/%d users in %v", count, len(userIDs), time.Since(startTime))
	}(m)

	return c.Send("Started publishing announcement")
}
