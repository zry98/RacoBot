package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
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

// predefined reply texts
const (
	FIBAPIOAuthAuthorizationExpiredErrorMessage = "Authorization expired, /login again"
	InternalErrorMessage                        = "Internal Error"
)

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

// Init initializes the bot
func Init(config BotConfig) {
	_b, err := tb.NewBot(tb.Settings{
		Token:       config.Token,
		Synchronous: true,                             // for webhook mode
		Verbose:     log.GetLevel() == log.DebugLevel, // for debugging only
	})
	if err != nil {
		log.Fatal(err)
	}
	b = &Bot{*_b}

	// on command `/start`, replies a `/login` message
	b.Handle("/start", func(c tb.Context) (err error) {
		return c.Send("/login to authorize Racó Bot") // TODO: make it nicer
	})

	// on command `/login`, replies a FIB API OAuth authorization link message for the user
	b.Handle("/login", func(c tb.Context) (err error) {
		userID := int64(c.Sender().ID)
		token, err := db.GetToken(userID)
		if err != nil && err != db.TokenNotFoundError {
			log.Error(err)
			return
		}
		if token != nil && token.AccessToken != "" && token.RefreshToken != "" {
			// already logged-in user
			return c.Send("Already logged-in, check /whoami; or, /logout to revoke the authorization")
		}

		// new user
		session, err := db.NewLoginSession(userID)
		if err != nil {
			log.Error(err)
			return c.Send(InternalErrorMessage)
		}

		m, err := b.sendMessage(userID, LoginLinkMessage{session})
		if err != nil {
			log.Error(err)
			return c.Send(InternalErrorMessage)
		}
		session.MessageID = int64(m.ID)

		return db.PutLoginSession(session)
	})

	// on command `/whoami`, replies the user's full name
	b.Handle("/whoami", middleware(func(c tb.Context) (err error) {
		userID := int64(c.Sender().ID)
		reply, err := NewClient(userID).GetFullName()
		if err != nil {
			return
		}

		return c.Send(reply)
	}))

	// on command `/logout`, revokes the user's FIB API OAuth token and deletes it from the database
	b.Handle("/logout", middleware(func(c tb.Context) (err error) {
		userID := int64(c.Sender().ID)
		err = NewClient(userID).Logout()
		if err != nil {
			return
		}

		return c.Send("Logout successful")
	}))

	// on command `/debug \d+`, replies notice message with the given ID in payload
	b.Handle("/debug", middleware(func(c tb.Context) (err error) {
		userID := int64(c.Sender().ID)
		messageID, err := strconv.ParseInt(c.Message().Payload, 10, 64)
		if err != nil {
			return c.Reply("Invalid payload")
		}

		notice, err := NewClient(userID).GetNotice(messageID)
		if err != nil {
			return
		}
		if notice.ID == 0 {
			// maybe don't exist, or not available to the user
			return c.Send("Notice unavailable")
		}

		return c.Send(notice.String(), sendNoticeMessageOption)
	}))

	// on command `/test`, replies the latest one notice message
	b.Handle("/test", middleware(func(c tb.Context) (err error) {
		userID := int64(c.Sender().ID)
		client := NewClient(userID)
		if client == nil {
			return TokenNotFoundError
		}
		notices, noticesHash, err := client.GetNoticesWithHash()
		if err != nil {
			return
		}
		if len(notices) == 0 {
			err = db.PutLastState(userID, db.LastState{
				NoticesHash: noticesHash,
			})
			if err != nil {
				return
			}

			return c.Send("No notices available")
		}

		sort.Slice(notices, func(i, j int) bool {
			return notices[i].ModifiedAt.Unix() < notices[j].ModifiedAt.Unix()
		})

		err = db.PutLastState(userID, db.LastState{
			NoticesHash:     noticesHash,
			NoticeTimestamp: notices[len(notices)-1].ModifiedAt.Unix(),
		})
		if err != nil {
			return
		}

		return c.Send(NoticeMessage{notices[len(notices)-1]}.String(), sendNoticeMessageOption)
	}))

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
			if err == TokenNotFoundError || err == fibapi.AuthorizationExpiredError ||
				(ok && errData.Response.StatusCode == http.StatusBadRequest && string(errData.Body) == fibapi.OAuthInvalidAuthorizationCodeResponse) {
				userID := int64(c.Sender().ID)
				log.WithField("uid", userID).Info(err)

				err = db.DeleteToken(userID)
				if err != nil {
					log.Error(err)
				}
				return c.Send(FIBAPIOAuthAuthorizationExpiredErrorMessage)
			}
		}
		return nil
	}
}

// sendMessage sends the given message to a Telegram user with the given ID
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

// SendMessage sends the given message to a Telegram user with the given ID
// it's meant to be used outside the package
func SendMessage(userID int64, message interface{}, opt ...interface{}) {
	if _, err := b.sendMessage(userID, message, opt...); err != nil {
		log.Error(err)
	}
}

// DeleteLoginLinkMessage deletes the login link message of the given login session
func DeleteLoginLinkMessage(s db.LoginSession) {
	err := b.Delete(tb.StoredMessage{
		MessageID: strconv.FormatInt(s.MessageID, 10),
		ChatID:    s.UserID,
	})
	if err != nil {
		log.Error(err)
	}
}
