package bot

import (
	"RacoBot/internal/locales"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	tb "gopkg.in/tucnak/telebot.v3"

	"RacoBot/internal/db"
	"RacoBot/pkg/fibapi"
)

// Update represents a Telegram bot Update
type Update struct {
	tb.Update
	// new fields of bot API v5.1, waiting for telebot to implement them...
	MyChatMember struct{} `json:"my_chat_member,omitempty"`
	ChatMember   struct{} `json:"chat_member,omitempty"`
}

// HandleUpdate handles a Telegram bot Update
func HandleUpdate(u Update) {
	b.ProcessUpdate(u.Update)
}

// Bot represents a Telegram bot
type Bot struct {
	tb.Bot
}

// sending options
var (
	sendNoticeMessageOption = &tb.SendOptions{ParseMode: tb.ModeHTML, DisableWebPagePreview: true}
	SendHTMLMessageOption   = &tb.SendOptions{ParseMode: tb.ModeHTML}
	SendSilentMessageOption = &tb.SendOptions{DisableNotification: true}
)

// BotConfig represents a configuration for Telegram bot
type BotConfig struct {
	Token      string `toml:"token"`
	WebhookURL string `toml:"webhook_URL"`
}

var b *Bot

func init() {
	setLanguageMenu.Inline(setLanguageMenu.Row(setLanguageButtonEN, setLanguageButtonES, setLanguageButtonCA))
}

// Init initializes the bot
func Init(config BotConfig) {
	_b, err := tb.NewBot(tb.Settings{
		Token:       config.Token,
		Synchronous: true,                             // for webhook mode
		Verbose:     log.GetLevel() >= log.DebugLevel, // for debugging only
	})
	if err != nil {
		log.Fatal(err)
	}
	b = &Bot{*_b}

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
	err = setWebhook(config.WebhookURL)
	if err != nil {
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

// sendMessage sends the given message to a Telegram User with the given ID
// it's meant to be used inside the package
func (bot *Bot) sendMessage(userID int64, message interface{}, opt ...interface{}) (*tb.Message, error) {
	switch m := message.(type) {
	case NoticeMessage:
		return bot.Send(&tb.Chat{ID: userID}, m.String(), sendNoticeMessageOption)
	case LoginLinkMessage:
		return bot.Send(&tb.Chat{ID: userID}, m.String(), SendHTMLMessageOption)
	case string:
		return bot.Send(&tb.Chat{ID: userID}, m, opt...)
	default:
		return nil, errors.New("message type is not sendable")
	}
}

// SendMessage sends the given message to a Telegram User with the given ID
// it's meant to be used outside the package
func SendMessage(userID int64, message interface{}, opt ...interface{}) {
	if _, err := b.sendMessage(userID, message, opt...); err != nil {
		log.Error(err)
	}
}

// DeleteLoginLinkMessage deletes the login link message of the given login session
func DeleteLoginLinkMessage(s db.LoginSession) {
	err := b.Delete(tb.StoredMessage{
		MessageID: strconv.FormatInt(s.LoginLinkMessageID, 10),
		ChatID:    s.UserID,
	})
	if err != nil {
		log.Error(err)
	}
}
