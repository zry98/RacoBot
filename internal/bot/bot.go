package bot

import (
	"fmt"
	"net/url"
	"strconv"

	log "github.com/sirupsen/logrus"
	tb "gopkg.in/telebot.v3"

	"RacoBot/internal/db"
	"RacoBot/internal/locales"
	"RacoBot/pkg/fibapi"
)

var (
	b                  *tb.Bot
	useLongPoller      bool
	adminUID           int64
	WebhookSecretToken string
	Username           string
)

var (
	// menu keyboard for selecting preferred language
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
	AdminUID           int64  `toml:"admin_uid,omitempty"`
	Token              string `toml:"token"`
	WebhookURL         string `toml:"webhook_url,omitempty"`
	WebhookSecretToken string `toml:"webhook_secret_token,omitempty"`
}

// Init initializes the bot
func Init(config Config) {
	var err error
	b, err = tb.NewBot(tb.Settings{
		Token:       config.Token,
		Synchronous: true,
		Verbose:     log.GetLevel() > log.DebugLevel,
	})
	if err != nil {
		fatalf("failed to initialize bot: %v", err)
	}

	// command handlers
	b.Use(errorInterceptor)
	b.Handle("/start", start)
	b.Handle("/login", login)
	b.Handle("/lang", setPreferredLanguage)
	b.Handle("/whoami", whoami)
	b.Handle("/test", test)
	b.Handle("/logout", logout)
	b.Handle("/debug", debug)
	b.Handle("/announce", publishAnnouncement, adminOnly)

	// initialize the menu for selecting preferred language
	setLanguageMenu.Inline(setLanguageMenu.Row(setLanguageButtonCA, setLanguageButtonES, setLanguageButtonEN))
	b.Handle(&setLanguageButtonCA, setPreferredLanguage)
	b.Handle(&setLanguageButtonES, setPreferredLanguage)
	b.Handle(&setLanguageButtonEN, setPreferredLanguage)

	// set command menus
	for _, languageCode := range locales.LanguageCodes {
		if err = setCommands(locales.Get(languageCode).CommandsMenu, languageCode); err != nil {
			fatalf("failed to set commands menu for %s: %v", languageCode, err)
		}
	}

	if config.WebhookURL == "" {
		if err = b.RemoveWebhook(true); err != nil {
			fatalf("failed to delete webhook: %v", err)
		}
		useLongPoller = true
		go b.Start()
	} else { // get updates from webhook HTTP handler instead of long poller if a URL is provided
		_, err = url.Parse(config.WebhookURL)
		if err != nil {
			fatalf("failed to parse webhook URL: %v", err)
		}
		if err = setWebhook(config.WebhookURL, config.WebhookSecretToken); err != nil {
			fatalf("failed to set webhook: %v", err)
		}
		// waiting for telebot to implement webhook secret token (https://github.com/tucnak/telebot/pull/543)
		//if err = b.SetWebhook(&tb.Webhook{
		//	Listen:        config.WebhookURL,
		//	DropUpdates:   false,
		//	HasCustomCert: false,
		//	WebhookSecretToken:   config.WebhookSecretToken,
		//}); err != nil {
		//	fatalf("failed to set webhook: %v", err)
		//}

		// save secret token for webhook request authentication in HTTP handler
		WebhookSecretToken = config.WebhookSecretToken
		useLongPoller = false
	}

	// save the bot username for later use in login flow callback URL
	Username = b.Me.Username
	if Username == "" {
		fatalf("failed to get bot username")
	}

	// save admin UID for later use in authorization middleware
	adminUID = config.AdminUID

	log.Debug("bot initialized")
}

// Stop stops the bot
func Stop() {
	if useLongPoller {
		b.Stop()
	} else {
		if err := b.RemoveWebhook(false); err != nil {
			log.Errorf("failed to delete webhook: %v", err)
		}
	}
	log.Debug("bot stopped")
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
func errorInterceptor(next tb.HandlerFunc) tb.HandlerFunc {
	return func(c tb.Context) error {
		if err := next(c); err != nil {
			if err == ErrUserNotFound {
				return c.Send(locales.Get(c.Sender().LanguageCode).StartMessage)
			}
			if err == fibapi.ErrAuthorizationExpired {
				log.Infof("user %d authorization has expired", c.Sender().ID)
				if e := db.DeleteUser(c.Sender().ID); e != nil {
					log.Errorf("failed to delete user %d: %v", c.Sender().ID, e)
				}
				return c.Send(&ErrorMessage{locales.Get(c.Sender().LanguageCode).FIBAPIAuthorizationExpiredMessage})
			}
			if err != ErrInternal {
				log.Errorf("error in handler: %v", err)
			}
			return c.Send(&ErrorMessage{locales.Get(c.Sender().LanguageCode).InternalErrorMessage})
		}
		return nil
	}
}

// SendMessage sends the given message to a Telegram user with the given ID
// it's meant to be called from outside the package
func SendMessage(userID int64, message interface{}, opt ...interface{}) *tb.Message {
	msg, err := b.Send(tb.ChatID(userID), message, opt...)
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
		log.Errorf("failed to delete login link message %d of session \"%s\": %v", s.LoginLinkMessageID, s.State, err)
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
func adminOnly(next tb.HandlerFunc) tb.HandlerFunc {
	return func(c tb.Context) (err error) {
		if c.Sender().ID != adminUID {
			return nil
		}
		return next(c)
	}
}

// fatalf is a wrapper for panic a formatted error
func fatalf(f string, a ...any) {
	panic(fmt.Errorf(f, a...))
}
