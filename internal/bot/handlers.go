package bot

import (
	"sort"
	"strconv"

	log "github.com/sirupsen/logrus"
	tb "gopkg.in/tucnak/telebot.v3"

	"RacoBot/internal/db"
	"RacoBot/internal/locales"
	"RacoBot/internal/ratelimiters"
	"RacoBot/pkg/fibapi"
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

	loginLinkMessage, err := b.Send(&tb.Chat{ID: userID}, &LoginLinkMessage{session})
	//loginLinkMessage, err := b.sendMessage(userID, LoginLinkMessage{session})
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
// setPreferredLanguage replies the menu of preferred languages for the user to choose from, or sets the user's preferred language with the given callback
func setPreferredLanguage(c tb.Context) error {
	if c.Callback() == nil {
		// on command `/lang`, show menu of languages
		user, err := db.GetUser(int64(c.Sender().ID))
		if err != nil {
			return err
		}

		return c.Reply(locales.Get(user.LanguageCode).ChoosePreferredLanguageMenuText, setLanguageMenu)

	} else {
		// on callbacks, set language accordingly
		languageCode := c.Callback().Unique
		if languageCode == "" || (languageCode != "en" && languageCode != "es" && languageCode != "ca") {
			return c.Reply(locales.Get("").InternalErrorMessage)
		}

		userID := int64(c.Sender().ID)
		user, err := db.GetUser(userID)
		if err != nil {
			return err
		}

		user.LanguageCode = languageCode
		if err = db.PutUser(user); err != nil {
			return err
		}

		return c.Edit(locales.Get(languageCode).PreferredLanguageSetMessage)
	}
}
