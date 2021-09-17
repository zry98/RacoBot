package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	tb "gopkg.in/tucnak/telebot.v3"

	"RacoBot/internal/db"
	"RacoBot/internal/locales"
	"RacoBot/pkg/fibapi"
)

var b *tb.Bot

type Update tb.Update

// HandleUpdate handles a Telegram bot update
func HandleUpdate(u Update) {
	b.ProcessUpdate(tb.Update(u))
}

// BotConfig represents a configuration for Telegram bot
type BotConfig struct {
	Token      string `toml:"token"`
	WebhookURL string `toml:"webhook_URL"`
}

func init() {
	// initialize set preferred language menu
	setLanguageMenu.Inline(setLanguageMenu.Row(setLanguageButtonEN, setLanguageButtonES, setLanguageButtonCA))
}

// Init initializes the bot
func Init(config BotConfig) {
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
	b.Handle("/start", start)
	b.Handle("/login", login)
	b.Handle("/whoami", middleware(whoami))
	b.Handle("/logout", middleware(logout))
	b.Handle("/debug", middleware(debug))
	b.Handle("/test", middleware(test))
	b.Handle("/lang", middleware(setPreferredLanguage))

	// inline keyboard button handlers
	b.Handle(&setLanguageButtonEN, middleware(setPreferredLanguage))
	b.Handle(&setLanguageButtonES, middleware(setPreferredLanguage))
	b.Handle(&setLanguageButtonCA, middleware(setPreferredLanguage))

	// update webhook URL
	if err = setWebhook(config.WebhookURL); err != nil {
		log.Fatal(err)
		return
	}

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

// middleware intercepts and handles the error returned by the next handler
func middleware(next tb.HandlerFunc) tb.HandlerFunc {
	return func(c tb.Context) (err error) {
		defer func() {
			if err != nil {
				log.Error(err)
				err = nil
			}
		}()

		err = next(c)
		if err != nil {
			errData, ok := err.(*oauth2.RetrieveError)
			if err == UserNotFoundError || err == fibapi.AuthorizationExpiredError ||
				(ok && errData.Response.StatusCode == http.StatusBadRequest && string(errData.Body) == fibapi.OAuthInvalidAuthorizationCodeResponse) {
				userID := int64(c.Sender().ID)
				log.WithField("uid", userID).Info(err)

				if err = db.DeleteUser(userID); err != nil {
					log.Error(err)
				}
				return c.Send(locales.Get(c.Sender().LanguageCode).FIBAPIAuthorizationExpiredErrorMessage)
			}
		}
		return nil
	}
}

// SendMessage sends the given message to a Telegram User with the given ID
// it's meant to be used outside the package
func SendMessage(userID int64, message interface{}, opt ...interface{}) {
	if _, err := b.Send(&tb.Chat{ID: userID}, message, opt...); err != nil {
		log.Error(err)
	}
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
