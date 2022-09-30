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

// start replies with a `/login` message
// on command `/start`
func start(c tb.Context) error {
	return c.Send(locales.Get(c.Sender().LanguageCode).StartMessage) // TODO: make it nicer
}

// login replies with a FIB API OAuth authorization link message for the user
// on command `/login`
func login(c tb.Context) error {
	userID := c.Sender().ID
	if !rl.LoginCommandAllowed(userID) {
		log.WithField("UID", userID).Info("login command rate limited")
		return nil
	}

	user, err := db.GetUser(userID)
	if err != nil && err != db.ErrUserNotFound {
		// db error
		log.Errorf("failed to get user %d: %v", userID, err)
		return ErrInternal
	}
	if err == nil && user.AccessToken != "" && user.RefreshToken != "" {
		// already logged-in user
		return c.Send(locales.Get(user.LanguageCode).AlreadyLoggedInMessage)
	}

	// new user
	session, err := db.NewLoginSession(userID, c.Sender().LanguageCode)
	if err != nil {
		log.Errorf("failed to create login session for user %d: %v", userID, err)
		return ErrInternal
	}

	loginLinkMessage, err := b.Send(tb.ChatID(userID), &LoginLinkMessage{session})
	if err != nil {
		log.Errorf("failed to send login link message to user %d: %v", userID, err)
		return ErrInternal
	}
	session.LoginLinkMessageID = int64(loginLinkMessage.ID)

	if err = db.PutLoginSession(session); err != nil {
		log.Errorf("failed to put login session %s: %v", session.State, err)
		return ErrInternal
	}
	return nil
}

// whoami replies with the user's full name
// on command `/whoami`
func whoami(c tb.Context) error {
	fullName, err := NewClient(c.Sender().ID).GetFullName()
	if err != nil {
		if err == ErrUserNotFound || err == fibapi.ErrAuthorizationExpired {
			return err
		}
		log.Errorf("failed to get full name of user %d: %v", c.Sender().ID, err)
		return ErrInternal
	}
	return c.Send(fullName)
}

// logout revokes the user's FIB API OAuth token and deletes it from the database
// on command `/logout`
func logout(c tb.Context) error {
	client := NewClient(c.Sender().ID)
	if client == nil {
		return ErrUserNotFound
	}
	if err := client.Logout(); err != nil {
		if err == fibapi.ErrAuthorizationExpired {
			return err
		}
		log.Errorf("failed to logout user %d: %v", c.Sender().ID, err)
		return c.Send(&ErrorMessage{locales.Get(client.User.LanguageCode).LogoutFailedMessage})
	}
	return c.Send(locales.Get(client.User.LanguageCode).LogoutSucceededMessage)
}

// debug replies with a notice with the given ID in payload
// on command `/debug <noticeID>`
func debug(c tb.Context) error {
	payload := c.Message().Payload
	if payload == "" {
		return nil
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
		return ErrInternal
	}
	return c.Send(&notice)
}

// test replies with the user's latest one notice
// on command `/test`
func test(c tb.Context) error {
	client := NewClient(c.Sender().ID)
	if client == nil {
		return ErrUserNotFound
	}
	notices, err := client.PrivateClient.GetNotices()
	if err != nil {
		if err == fibapi.ErrAuthorizationExpired {
			return err
		}
		log.Errorf("failed to get notices of user %d: %v", c.Sender().ID, err)
		return ErrInternal
	}

	defer func() { // save states to DB by the way
		if len(notices) > 0 {
			client.User.LastNoticeTimestamp = notices[len(notices)-1].PublishedAt.Unix()
			if e := db.PutUser(client.User); e != nil {
				log.Errorf("failed to put user %d: %v", c.Sender().ID, e)
			}
		}
	}()

	if len(notices) == 0 {
		return c.Send(&ErrorMessage{locales.Get(client.User.LanguageCode).NoAvailableNoticesErrorMessage})
	}
	latestNotice := notices[len(notices)-1]
	return c.Send(&NoticeMessage{latestNotice, client.User, getNoticeLinkURL(latestNotice)})
}

// setPreferredLanguage replies with the menu of supported languages for the user to select from on command `/lang`,
// or sets the user's preferred language when on callbacks &setLanguageButtonEN, &setLanguageButtonES, &setLanguageButtonCA
func setPreferredLanguage(c tb.Context) error {
	user, err := db.GetUser(c.Sender().ID)
	if err != nil {
		if err == db.ErrUserNotFound {
			return ErrUserNotFound
		}
		log.Errorf("failed to get user %d: %v", c.Sender().ID, err)
		return ErrInternal
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
	if e := db.PutUser(user); e != nil {
		log.Errorf("failed to put user %d: %v", c.Sender().ID, e)
		return ErrInternal
	}
	return c.Edit(locales.Get(langCode).PreferredLanguageSetMessage)
}

// publishAnnouncement publishes and pins the given announcement to all users in database
// on command `/announce`
func publishAnnouncement(c tb.Context) error {
	m := AnnouncementMessage{
		Text: strings.ReplaceAll(c.Message().Payload, "<br>", "\n"),
	}

	go func(announcement AnnouncementMessage) {
		logger := log.WithField("job", "PublishAnnouncement")
		count := 0

		userIDs, err := db.GetAllUserIDs()
		if err != nil {
			logger.Errorf("failed to get all user IDs: %v", err)
			return
		}
		logger.Infof("found %d users", len(userIDs))

		startTime := time.Now()
		for _, userID := range userIDs {
			if _, err = b.Send(tb.ChatID(userID), &announcement); err != nil {
				logger.Errorf("failed to send announcement to user %d: %v", userID, err)
				continue
			}
			count++
		}

		logger.Infof("sent announcement to %d/%d users in %v", count, len(userIDs), time.Since(startTime))
	}(m)

	return c.Send("Started publishing announcement")
}
