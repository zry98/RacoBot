package bot

import (
	"reflect"
	"runtime"
	"strconv"

	log "github.com/sirupsen/logrus"
	tb "gopkg.in/telebot.v3"

	"RacoBot/internal/db"
	"RacoBot/internal/locale"
	"RacoBot/pkg/fibapi"
)

var (
	b                     *tb.Bot
	useLongPoller         bool
	adminUID              int64
	WebhookSecretToken    string
	Username              string
	MailtoLinkRedirectURL string
)

var (
	// menu keyboard for selecting preferred language
	setLanguageMenu     = &tb.ReplyMarkup{OneTimeKeyboard: true}
	setLanguageButtonEN = setLanguageMenu.Data("English", "en")
	setLanguageButtonES = setLanguageMenu.Data("Castellano", "es")
	setLanguageButtonCA = setLanguageMenu.Data("Català", "ca")
)

// HandleUpdate handles a Telegram bot update
func HandleUpdate(u tb.Update) {
	b.ProcessUpdate(u)
}

// Config represents a configuration for Telegram bot
type Config struct {
	AdminUID              int64  `toml:"admin_uid,omitempty"`
	Token                 string `toml:"token"`
	WebhookURL            string `toml:"webhook_url,omitempty"`
	WebhookSecretToken    string `toml:"webhook_secret_token,omitempty"`
	MailtoLinkRedirectURL string `toml:"mailto_link_redirect_url,omitempty"`
}

// Init initializes the bot
func Init(config Config) {
	var err error
	b, err = tb.NewBot(tb.Settings{
		Token:       config.Token,
		Synchronous: true,
		ParseMode:   tb.ModeHTML,
		Verbose:     log.GetLevel() > log.DebugLevel,
	})
	if err != nil {
		log.Fatalf("failed to initialize bot: %v", err)
	}

	// set handlers
	b.Use(errorInterceptor)
	b.Handle("/start", start)
	b.Handle("/help", help)
	b.Handle("/login", login)
	b.Handle("/lang", setPreferredLanguage)
	b.Handle("/toggle_mute_banner_notices", toggleMuteBannerNotices)
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
	for _, languageCode := range locale.LanguageCodes {
		if err = setCommands(locale.Get(languageCode).CommandsMenu, languageCode); err != nil {
			log.Fatalf("failed to set commands menu for %s: %v", languageCode, err)
		}
	}

	if config.WebhookURL == "" { // get updates via long polling if no webhook URL is given
		if err = b.RemoveWebhook(true); err != nil {
			log.Fatalf("failed to delete webhook: %v", err)
		}
		useLongPoller = true
		go b.Start()
	} else { // get updates from webhook HTTP handler instead of long poller
		if _, err = b.Raw("setWebhook", struct {
			URL         string `json:"url"`
			SecretToken string `json:"secret_token"`
		}{config.WebhookURL, config.WebhookSecretToken}); err != nil {
			log.Fatalf("failed to set webhook: %v", err)
		}
		useLongPoller = false
		// save secret token for webhook request authentication in HTTP handler
		WebhookSecretToken = config.WebhookSecretToken
	}

	// save the bot username for later use in login flow callback URL
	Username = b.Me.Username
	if Username == "" {
		log.Fatalf("failed to get bot username")
	}
	// save admin UID for later use in authorization middleware
	adminUID = config.AdminUID

	MailtoLinkRedirectURL = config.MailtoLinkRedirectURL

	log.Debug("bot started")
}

// Stop stops the bot
func Stop() {
	if b != nil {
		if useLongPoller {
			b.Stop()
		} else {
			if err := b.RemoveWebhook(false); err != nil {
				log.Errorf("failed to delete webhook: %v", err)
			}
		}
	}
	log.Debug("bot stopped")
}

// errorInterceptor is a middleware that intercepts and handles the error returned by the next handler
func errorInterceptor(next tb.HandlerFunc) tb.HandlerFunc {
	return func(c tb.Context) error {
		if err := next(c); err != nil {
			if err == ErrUserNotFound {
				return c.Send(locale.Get(c.Sender().LanguageCode).StartMessage)
			}
			if err == fibapi.ErrAuthorizationExpired {
				log.Infof("user %d authorization has expired", c.Sender().ID)
				if e := db.DelUser(c.Sender().ID); e != nil {
					log.Errorf("failed to delete user %d: %v", c.Sender().ID, e)
				}
				return c.Send(&ErrorMessage{locale.Get(c.Sender().LanguageCode).FIBAPIAuthorizationExpiredMessage})
			}
			if err != ErrInternal {
				handlerName := runtime.FuncForPC(reflect.ValueOf(next).Pointer()).Name()
				log.Errorf("error in handler %s: %v", handlerName, err)
			}
			return c.Send(&ErrorMessage{locale.Get(c.Sender().LanguageCode).InternalErrorMessage})
		}
		return nil
	}
}

// SendMessage sends the given message to a Telegram user with the given ID
// it's meant to be called from outside the package
func SendMessage(userID int64, message interface{}, opt ...interface{}) *tb.Message {
	msg, err := b.Send(tb.ChatID(userID), message, append(opt, tb.NoPreview)...)
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
