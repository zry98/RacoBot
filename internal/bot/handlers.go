package bot

import (
	"sort"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	tb "gopkg.in/telebot.v3"

	"RacoBot/internal/db"
	rl "RacoBot/internal/db/ratelimiters"
	"RacoBot/internal/locales"
	"RacoBot/pkg/fibapi"
)

// on command `/start`
// start replies a `/login` message
func start(c tb.Context) (err error) {
	return c.Send(locales.Get(c.Sender().LanguageCode).StartMessage) // TODO: make it nicer
}

// on command `/login`
// login replies a FIB API OAuth authorization link message for the user
func login(c tb.Context) (err error) {
	userID := c.Sender().ID
	if !rl.LoginCommandAllowed(userID) {
		log.WithField("uid", userID).Info("login command rate limited")
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
		return c.Send(locales.Get(c.Sender().LanguageCode).InternalErrorMessage)
	}

	loginLinkMessage, err := b.Send(&tb.Chat{ID: userID}, &LoginLinkMessage{session})
	if err != nil {
		log.Error(err)
		return c.Send(locales.Get(c.Sender().LanguageCode).InternalErrorMessage)
	}
	session.LoginLinkMessageID = int64(loginLinkMessage.ID)

	return db.PutLoginSession(session)
}

// on command `/whoami`
// whoami replies the user's full name
func whoami(c tb.Context) (err error) {
	reply, err := NewClient(c.Sender().ID).GetFullName()
	if err != nil {
		return
	}

	return c.Send(reply)
}

// on command `/logout`
// logout revokes the user's FIB API OAuth token and deletes it from the database
func logout(c tb.Context) (err error) {
	client := NewClient(c.Sender().ID)
	if client == nil {
		return ErrUserNotFound
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
	noticeID, err := strconv.ParseInt(c.Message().Payload, 10, 64)
	if err != nil {
		return c.Reply("Invalid payload")
	}

	client := NewClient(c.Sender().ID)
	if client == nil {
		return ErrUserNotFound
	}
	notice, err := client.GetNotice(noticeID)
	if err == fibapi.ErrNoticeNotFound || (err == nil && notice.ID == 0) {
		// notice doesn't exist or isn't available to the user
		return c.Send(&ErrorMessage{locales.Get(client.User.LanguageCode).NoticeUnavailableErrorMessage})
	} else if err != nil {
		return
	}

	return c.Send(&notice)
}

// on command `/test`
// test replies the latest one notice message
func test(c tb.Context) (err error) {
	client := NewClient(c.Sender().ID)
	if client == nil {
		return ErrUserNotFound
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

		return c.Send(&ErrorMessage{locales.Get(client.User.LanguageCode).NoAvailableNoticesErrorMessage})
	}

	sort.Slice(notices, func(i, j int) bool {
		return notices[i].ModifiedAt.Unix() < notices[j].ModifiedAt.Unix()
	})

	client.User.LastNoticesHash = noticesHash
	client.User.LastNoticeTimestamp = notices[len(notices)-1].ModifiedAt.Unix()
	if err = db.PutUser(client.User); err != nil {
		return
	}

	return c.Send(&NoticeMessage{notices[len(notices)-1], client.User})
}

var (
	setLanguageMenu     = &tb.ReplyMarkup{OneTimeKeyboard: true}
	setLanguageButtonEN = setLanguageMenu.Data("English", "en")
	setLanguageButtonES = setLanguageMenu.Data("Castellano", "es")
	setLanguageButtonCA = setLanguageMenu.Data("Català", "ca")
)

// on command `/lang`, on callbacks &setLanguageButtonEN, &setLanguageButtonES, &setLanguageButtonCA
// setPreferredLanguage replies the menu of supported languages for the user to choose from, or sets the user's preferred language based on the given callback
func setPreferredLanguage(c tb.Context) error {
	// on command `/lang`, show menu for setting preferred language
	if c.Callback() == nil {
		user, err := db.GetUser(c.Sender().ID)
		if err != nil {
			return err
		}

		return c.Reply(locales.Get(user.LanguageCode).ChoosePreferredLanguageMenuText, setLanguageMenu)
	}

	// on callbacks, set the user's preferred language accordingly
	languageCode := c.Callback().Unique
	if languageCode == "" || (languageCode != "en" && languageCode != "es" && languageCode != "ca") {
		return c.Reply(locales.Get(c.Sender().LanguageCode).InternalErrorMessage)
	}

	user, err := db.GetUser(c.Sender().ID)
	if err != nil {
		return err
	}

	user.LanguageCode = languageCode
	if err = db.PutUser(user); err != nil {
		return err
	}

	return c.Edit(locales.Get(languageCode).PreferredLanguageSetMessage)
}

// on command `/announce`
// publishAnnouncement publishes and pins the given announcement to all users
func publishAnnouncement(c tb.Context) (err error) {
	announcementMessage := &AnnouncementMessage{
		Text: strings.ReplaceAll(c.Message().Payload, "<br>", "\n"),
	}

	userIDs, err := db.GetUserIDs()
	if err != nil {
		return
	}

	go func() {
		var message *tb.Message
		for _, userID := range userIDs {
			message, err = b.Send(&tb.Chat{ID: userID}, announcementMessage)
			if err != nil {
				log.Error(err)
				continue
			}

			err = b.Pin(message)
			if err != nil {
				log.Error(err)
				continue
			}
		}
	}()

	return nil
}
