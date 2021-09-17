package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"

	"RacoBot/internal/bot"
	"RacoBot/internal/db"
	"RacoBot/internal/locales"
	"RacoBot/internal/ratelimiters"
	"RacoBot/pkg/fibapi"
)

// response bodies
const (
	InternalErrorResponseBody  = "Internal error"
	RateLimitedResponseBody    = "Rate limited"
	InvalidRequestResponseBody = "Authorization failed (invalid request)"
)

// HandleBotUpdate handles an incoming Telegram Bot Update request
func HandleBotUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintln(w, "Invalid request")
		return
	}

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)

		fmt.Fprintln(w)
		return
	}

	var update bot.Update
	err = json.Unmarshal(body, &update)
	if err != nil {
		log.WithFields(log.Fields{
			"ip": r.RemoteAddr,
		}).Error(err)

		fmt.Fprintln(w)
		return
	}

	var userID int64
	if update.Message != nil {
		userID = update.Message.Sender.ID
	} else if update.Callback != nil {
		userID = update.Callback.Sender.ID
	}
	if userID != 0 && !ratelimiters.BotUpdateAllowed(r.Context(), userID) {
		// TODO: handle rate limited users
		log.WithFields(log.Fields{
			"uid": userID,
		}).Info("Rate limited")

		fmt.Fprintln(w)
		return
	}

	go bot.HandleUpdate(update) // use goroutine to handle the update, thus response to the webhook request asap

	fmt.Fprintln(w)
}

// HandleOAuthRedirect handles an incoming FIB API OAuth redirect request
func HandleOAuthRedirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintln(w, InvalidRequestResponseBody)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, InvalidRequestResponseBody)
		return
	}

	if !ratelimiters.OAuthRedirectRequestAllowed(r.Context(), r.RemoteAddr) {
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprintln(w, RateLimitedResponseBody)
		return
	}

	loginSession, err := db.GetLoginSession(state)
	if err == db.LoginSessionNotFoundError || loginSession.UserID == 0 || loginSession.LoginLinkMessageID == 0 {
		log.WithFields(log.Fields{
			"ip": r.RemoteAddr,
			"s":  state,
			"c":  code,
		}).Info("Invalid redirect request (login session not found)")

		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, InvalidRequestResponseBody)
		return
	}
	if err != nil {
		log.Error(err)

		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, InvalidRequestResponseBody)
		return
	}

	token, userInfo, err := fibapi.Authorize(code)
	if err != nil {
		if err == fibapi.InvalidAuthorizationCodeError {
			log.WithFields(log.Fields{
				"ip":  r.RemoteAddr,
				"s":   state,
				"c":   code,
				"uid": loginSession.UserID,
			}).Info("Invalid redirect request (invalid authorization code)")
		} else {
			// other internal errors
			log.Warn(err)
		}

		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, InvalidRequestResponseBody)
		return
	}

	err = db.PutUser(db.User{
		ID:           loginSession.UserID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenExpiry:  token.Expiry.Unix(),
		LanguageCode: loginSession.UserLanguageCode,
	})
	if err != nil {
		log.Error(err)

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, InternalErrorResponseBody)
		return
	}

	err = db.DeleteLoginSession(state)
	if err != nil {
		log.Error(err)

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, InternalErrorResponseBody)
		return
	}

	bot.DeleteLoginLinkMessage(loginSession)
	greetingMessage := fmt.Sprintf(locales.Get(loginSession.UserLanguageCode).GreetingMessage, userInfo.FirstName)
	bot.SendMessage(loginSession.UserID, greetingMessage)

	guideMessage := locales.Get(loginSession.UserLanguageCode).HelpMessage
	bot.SendMessage(loginSession.UserID, &bot.SilentMessage{Text: guideMessage})

	// use Telegram URI to redirect user to the chat
	http.Redirect(w, r, "tg://resolve?domain="+bot.BotUsername, 301)
	fmt.Fprintln(w, locales.Get(loginSession.UserLanguageCode).AuthorizedResponseBody)
}

// middleware provides some useful middlewares for the server
func middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// returns an HTTP 500 response if a handler got panicked
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, InternalErrorResponseBody)
				return
			}
		}()

		// gets client's real IP if serves behind Cloudflare
		if ip := r.Header.Get("Cf-Connecting-Ip"); ip != "" {
			r.RemoteAddr = ip
		}

		h.ServeHTTP(w, r)
	})
}
