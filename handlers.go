package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"

	"RacoBot/internal/bot"
	"RacoBot/internal/db"
	"RacoBot/internal/ratelimiters"
	"RacoBot/pkg/fibapi"
)

// response bodies
const (
	InternalErrorResponseBody  = "Internal error"
	RateLimitedResponseBody    = "Rate limited"
	InvalidRequestResponseBody = "Authorization failed (invalid request)"
	AuthorizedResponseBody     = "<h1>Authorized</h1>" // TODO: make it nicer
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

	if !ratelimiters.BotUpdateAllowed(r.Context(), int64(update.Message.Sender.ID)) {
		// TODO: handle rate limited users
		log.WithFields(log.Fields{
			"uid": update.Message.Sender.ID,
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
	if err != nil && err != db.LoginSessionNotFoundError {
		log.Error(err)

		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintln(w, InvalidRequestResponseBody)
		return
	}
	if err == db.LoginSessionNotFoundError || (&loginSession != nil && loginSession.UserID == 0 || loginSession.MessageID == 0) {
		log.WithFields(log.Fields{
			"ip": r.RemoteAddr,
			"s":  state,
			"c":  code,
		}).Info("Invalid redirect request (login session not found)")

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

	err = db.PutToken(loginSession.UserID, token)
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
	welcomeMessage := fmt.Sprintf("Hello, %s!", userInfo.FirstName)
	bot.SendMessage(loginSession.UserID, welcomeMessage)

	// TODO: make it nicer
	guideMessage := fmt.Sprintf("You can use /test to preview the latest one notice, \n/logout to stop receiving messages and revoke the authorization on server")
	bot.SendMessage(loginSession.UserID, guideMessage, bot.SendSilentMessageOption)

	fmt.Fprintln(w, AuthorizedResponseBody)
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
