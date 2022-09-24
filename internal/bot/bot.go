package bot

import (
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	tb "gopkg.in/telebot.v3"

	"RacoBot/internal/db"
	"RacoBot/internal/locales"
	"RacoBot/pkg/fibapi"
)

var (
	b *tb.Bot

	SecretToken string
	Username    string
	adminUID    int64

	setLanguageMenu     = &tb.ReplyMarkup{OneTimeKeyboard: true}
	setLanguageButtonEN = setLanguageMenu.Data("English", "en")
	setLanguageButtonES = setLanguageMenu.Data("Castellano", "es")
	setLanguageButtonCA = setLanguageMenu.Data("Català", "ca")
)

type Update tb.Update

// HandleUpdate handles a Telegram bot update
func HandleUpdate(u Update) {
	b.ProcessUpdate(tb.Update(u))
}

// Config represents a configuration for Telegram bot
type Config struct {
	Token       string `toml:"token"`
	WebhookURL  string `toml:"webhook_url"`
	SecretToken string `toml:"secret_token"`
	AdminUID    int64  `toml:"admin_uid"`
}

// Init initializes the bot
func Init(config Config) {
	var err error
	b, err = tb.NewBot(tb.Settings{
		Token:       config.Token,
		Synchronous: true,                             // for webhook mode
		Verbose:     log.GetLevel() >= log.DebugLevel, // for debugging only
	})
	if err != nil {
		log.Fatalf("failed to initialize bot: %v", err)
	}

	// command handlers
	b.Use(errorInterceptor())
	b.Handle("/start", start)
	b.Handle("/login", login)
	b.Handle("/lang", setPreferredLanguage)
	b.Handle("/whoami", whoami)
	b.Handle("/test", test)
	b.Handle("/logout", logout)
	b.Handle("/debug", debug)
	b.Handle("/announce", publishAnnouncement, adminOnly())

	// initialize the menu for selecting preferred language
	setLanguageMenu.Inline(setLanguageMenu.Row(setLanguageButtonCA, setLanguageButtonES, setLanguageButtonEN))
	b.Handle(&setLanguageButtonCA, setPreferredLanguage)
	b.Handle(&setLanguageButtonES, setPreferredLanguage)
	b.Handle(&setLanguageButtonEN, setPreferredLanguage)

	// set command menus
	for _, languageCode := range locales.LanguageCodes {
		if err = setCommands(locales.Get(languageCode).CommandsMenu, languageCode); err != nil {
			log.Fatalf("failed to set commands menu for %s: %v", languageCode, err)
		}
	}

	// update webhook URL
	if err = setWebhook(config.WebhookURL, config.SecretToken); err != nil {
		log.Fatalf("failed to set webhook URL: %v", err)
	}

	// save secret token for webhook request authentication
	SecretToken = config.SecretToken

	// save the bot username for later use in login flow callback URL
	Username = b.Me.Username

	// save admin UID for later use in authorization middleware
	adminUID = config.AdminUID
}

// setWebhook sets the Telegram bot webhook URL to the given one
func setWebhook(URL string, secretToken string) error {
	params := struct {
		URL         string `json:"url"`
		SecretToken string `json:"secret_token"`
	}{
		URL,
		secretToken,
	}
	_, err := b.Raw("setWebhook", params)
	return err
}

// errorInterceptor is a middleware that intercepts and handles the error returned by the next handler
func errorInterceptor() tb.MiddlewareFunc {
	return func(next tb.HandlerFunc) tb.HandlerFunc {
		return func(c tb.Context) (err error) {
			err = next(c)
			if err != nil {
				errData, ok := err.(*oauth2.RetrieveError)
				if err == ErrUserNotFound || err == fibapi.ErrAuthorizationExpired ||
					(ok && errData.Response.StatusCode == http.StatusBadRequest && string(errData.Body) == fibapi.OAuthInvalidAuthorizationCodeResponse) {
					userID := c.Sender().ID
					log.WithField("UID", userID).Info(err)

					if err = db.DeleteUser(userID); err != nil {
						log.Error(err)
					}
					return c.Send(locales.Get(c.Sender().LanguageCode).FIBAPIAuthorizationExpiredMessage)
				}
				log.Error(err)
				return c.Send(locales.Get(c.Sender().LanguageCode).InternalErrorMessage,
					&tb.SendOptions{ParseMode: tb.ModeHTML})
			}
			return nil
		}
	}
}

// SendMessage sends the given message to a Telegram user with the given ID
// it's meant to be used outside the package
func SendMessage(userID int64, message interface{}, opt ...interface{}) *tb.Message {
	msg, err := b.Send(&tb.Chat{ID: userID}, message, opt...)
	if err != nil {
		log.Errorf("failed to send message to user %d: %s", userID, err)
		return nil
	}
	return msg
}

// DeleteLoginLinkMessage deletes the login link message of the given login session
func DeleteLoginLinkMessage(s db.LoginSession) {
	if err := b.Delete(tb.StoredMessage{
		MessageID: strconv.FormatInt(s.LoginLinkMessageID, 10),
		ChatID:    s.UserID,
	}); err != nil {
		log.Errorf("failed to delete login link message %d of session %s: %v", s.LoginLinkMessageID, s.State, err)
	}
}

// setCommands changes the list of the bot's commands
// FIXME: remove it when telebot implemented the language_code parameter
func setCommands(cmds []tb.Command, langCode string) error {
	params := struct {
		Commands     []tb.Command `json:"commands"`
		LanguageCode string       `json:"language_code"`
	}{
		cmds,
		langCode,
	}
	_, err := b.Raw("setMyCommands", params)
	return err
}

// adminOnly is a middleware that checks if the sender is admin
func adminOnly() tb.MiddlewareFunc {
	return func(next tb.HandlerFunc) tb.HandlerFunc {
		return func(c tb.Context) (err error) {
			if c.Sender().ID != adminUID {
				return nil
			}
			return next(c)
		}
	}
}
