package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	tb "gopkg.in/telebot.v3"

	"RacoBot/internal/db"
	"RacoBot/internal/locales"
	"RacoBot/pkg/fibapi"
)

var b *tb.Bot
var (
	BotUsername string
	AdminUID    int64
)

type Update tb.Update

// HandleUpdate handles a Telegram bot update
func HandleUpdate(u Update) {
	b.ProcessUpdate(tb.Update(u))
}

// Config represents a configuration for Telegram bot
type Config struct {
	Token      string `toml:"token"`
	WebhookURL string `toml:"webhook_URL"`
	AdminUID   int64  `toml:"admin_UID"`
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
		log.Fatal(err)
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

	// initialize menu for setting preferred language
	setLanguageMenu.Inline(setLanguageMenu.Row(setLanguageButtonCA, setLanguageButtonES, setLanguageButtonEN))
	b.Handle(&setLanguageButtonCA, setPreferredLanguage)
	b.Handle(&setLanguageButtonES, setPreferredLanguage)
	b.Handle(&setLanguageButtonEN, setPreferredLanguage)

	// set command menus
	for _, languageCode := range []string{"ca", "es", "en"} {
		if err = setCommands(locales.Get(languageCode).CommandsMenu, languageCode); err != nil {
			log.Fatal(err)
			return
		}
	}

	// update webhook URL
	if err = setWebhook(config.WebhookURL); err != nil {
		log.Fatal(err)
		return
	}

	// save bot username for later use in callback URL
	BotUsername = b.Me.Username

	// save admin UserID for later use in authorization middleware
	AdminUID = config.AdminUID

	log.Info("Bot OK") // all done, start serving
}

// setWebhook sets the Telegram bot webhook URL to the given one
func setWebhook(URL string) error {
	resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/bot%s/setWebhook?url=%s", b.Token, URL))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error setting webhook (HTTP %d)", resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var r struct {
		OK          bool   `json:"ok"`
		Result      bool   `json:"result,omitempty"`
		ErrorCode   int    `json:"error_code,omitempty"`
		Description string `json:"description"`
	}
	if err = json.Unmarshal(body, &r); err != nil {
		return err
	}
	if r.OK && r.Result && (r.Description == "Webhook was set" || r.Description == "Webhook is already set") {
		return nil
	}
	return fmt.Errorf("error setting webhook: (code %d) %s", r.ErrorCode, r.Description)
}

// errorInterceptor is a middleware that intercepts and handles the error returned by the next handler
func errorInterceptor() tb.MiddlewareFunc {
	return func(next tb.HandlerFunc) tb.HandlerFunc {
		return func(c tb.Context) (err error) {
			defer func() {
				// if there is still an error remaining, log it
				if err != nil {
					log.Error(err)
					err = nil
				}
			}()

			err = next(c)
			if err != nil {
				errData, ok := err.(*oauth2.RetrieveError)
				if err == ErrUserNotFound || err == fibapi.ErrAuthorizationExpired ||
					(ok && errData.Response.StatusCode == http.StatusBadRequest && string(errData.Body) == fibapi.OAuthInvalidAuthorizationCodeResponse) {
					userID := c.Sender().ID
					log.WithField("uid", userID).Info(err)

					if err = db.DeleteUser(userID); err != nil {
						log.Error(err)
					}
					return c.Send(locales.Get(c.Sender().LanguageCode).FIBAPIAuthorizationExpiredErrorMessage)
				}
			}
			return c.Send(locales.Get(c.Sender().LanguageCode).InternalErrorMessage)
		}
	}
}

// SendMessage sends the given message to a Telegram user with the given ID
// it's meant to be used outside the package
func SendMessage(userID int64, message interface{}, opt ...interface{}) *tb.Message {
	sent, err := b.Send(&tb.Chat{ID: userID}, message, opt...)
	if err != nil {
		log.Errorf("error sending message to user %d: %s", userID, err)
	}
	return sent
}

// DeleteLoginLinkMessage deletes the login link message of the given login session
func DeleteLoginLinkMessage(s db.LoginSession) {
	if err := b.Delete(tb.StoredMessage{
		MessageID: strconv.FormatInt(s.LoginLinkMessageID, 10),
		ChatID:    s.UserID,
	}); err != nil {
		log.Error(err)
	}
}

// FIXME: remove it when telebot implemented the language_code parameter
// setCommands changes the list of the bot's commands
func setCommands(cmds []tb.Command, langCode string) error {
	params := struct {
		Commands     []tb.Command `json:"commands"`
		LanguageCode string       `json:"language_code"`
	}{
		Commands:     cmds,
		LanguageCode: langCode,
	}
	_, err := b.Raw("setMyCommands", params)
	return err
}

// adminOnly is a middleware that checks if the sender is admin
func adminOnly() tb.MiddlewareFunc {
	return func(next tb.HandlerFunc) tb.HandlerFunc {
		return func(c tb.Context) (err error) {
			if c.Sender().ID != AdminUID {
				return nil
			}
			return next(c)
		}
	}
}
