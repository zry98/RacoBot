package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"

	"RacoBot/internal/bot"
	"RacoBot/internal/db"
	rl "RacoBot/internal/db/ratelimiters"
	"RacoBot/internal/locales"
	"RacoBot/pkg/fibapi"
)

// response bodies
const (
	InternalErrorResponseBody       = "Internal error"
	RateLimitedResponseBody         = "Rate limited"
	TelegramRequestTokenHeader      = "X-Telegram-Bot-Api-Secret-Token"
	InvalidOAuthRequestResponseBody = "Authorization failed (invalid request)"
	AuthorizedResponseBodyTemplate  = "<!DOCTYPE html><html lang=\"%s\"><head><meta charset=\"utf-8\"><meta http-equiv=\"refresh\" content=\"0; url=tg://resolve?domain=%s\"><title>Racó Bot</title></head><body><h1>%s</h1><p>%s</p></body></html>\n"
)

// HandleBotUpdate handles an incoming Telegram Bot Update request
func HandleBotUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// check if request is legit from Telegram
	if bot.WebhookSecretToken != "" && r.Header.Get(TelegramRequestTokenHeader) != bot.WebhookSecretToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Errorf("failed to read request body: %v", err)
		return
	}
	if len(body) == 0 { // getting an additional empty request after normal update when using Cloudflare proxy, weird
		return
	}

	var update bot.Update
	if err = json.Unmarshal(body, &update); err != nil {
		log.WithFields(log.Fields{
			"IP": r.RemoteAddr,
		}).Errorf("invalid update: %v", err)
		return
	}

	var userID int64
	if update.Message != nil {
		userID = update.Message.Sender.ID
	} else if update.Callback != nil {
		userID = update.Callback.Sender.ID
	} else {
		log.WithFields(log.Fields{
			"IP": r.RemoteAddr,
		}).Error("invalid update: no message or callback")
		return
	}
	if userID != 0 && !rl.BotUpdateAllowed(r.Context(), userID) {
		log.WithFields(log.Fields{
			"UID": userID,
		}).Info("rate limited")
		return
	}

	// handle the update in the background and respond to the webhook request ASAP
	go bot.HandleUpdate(update)
}

// HandleOAuthRedirect handles an incoming FIB API OAuth redirect request
func HandleOAuthRedirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintln(w, InvalidOAuthRequestResponseBody)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" || state == "" || r.Header.Get("referer") != fibapi.BaseURL {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, InvalidOAuthRequestResponseBody)
		return
	}

	if !rl.OAuthRedirectRequestAllowed(r.Context(), r.RemoteAddr) {
		log.WithFields(log.Fields{
			"IP": r.RemoteAddr,
		}).Info("rate limited")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprintln(w, RateLimitedResponseBody)
		return
	}

	loginSession, err := db.GetLoginSession(state)
	if err != nil && err != db.ErrLoginSessionNotFound {
		log.WithFields(log.Fields{
			"IP":    r.RemoteAddr,
			"state": state,
		}).Errorf("failed to get login session: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, InternalErrorResponseBody)
		return
	}
	if err == db.ErrLoginSessionNotFound || loginSession.UserID == 0 || loginSession.LoginLinkMessageID == 0 {
		log.WithFields(log.Fields{
			"IP":    r.RemoteAddr,
			"state": state,
		}).Info("invalid OAuth redirect request: login session not found or invalid")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, InvalidOAuthRequestResponseBody)
		return
	}

	token, userInfo, err := fibapi.Authorize(code)
	if err != nil {
		logger := log.WithFields(log.Fields{
			"IP":    r.RemoteAddr,
			"UID":   loginSession.UserID,
			"state": state,
			"code":  code,
		})
		if err == fibapi.ErrInvalidAuthorizationCode {
			logger.Info("invalid OAuth redirect request: invalid authorization code")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, InvalidOAuthRequestResponseBody)
		} else {
			logger.Errorf("failed to authorize: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, InternalErrorResponseBody)
		}
		return
	}

	err = db.PutUser(db.User{
		ID:           loginSession.UserID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenExpiry:  token.Expiry.Unix() - 10*60, // expire it 10 minutes in advance
		LanguageCode: loginSession.UserLanguageCode,
	})
	if err != nil {
		log.Errorf("failed to put user %d: %v", loginSession.UserID, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, InternalErrorResponseBody)
		return
	}

	err = db.DeleteLoginSession(state)
	if err != nil {
		log.Errorf("failed to delete login session %s: %v", state, err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, InternalErrorResponseBody)
		return
	}
	bot.DeleteLoginLinkMessage(loginSession)

	log.Infof("user %d logged in", loginSession.UserID)
	greetingMessage := fmt.Sprintf("%s\n\n%s",
		fmt.Sprintf(locales.Get(loginSession.UserLanguageCode).GreetingMessage, userInfo.FirstName),
		locales.Get(loginSession.UserLanguageCode).HelpMessage)
	_ = bot.SendMessage(loginSession.UserID, greetingMessage)

	// respond HTML with authorized message
	// meanwhile make 301 redirect to let user back to the chat using Telegram URI scheme
	//fmt.Fprintf(w, AuthorizedResponseBodyTemplate,
	//	loginSession.UserLanguageCode,
	//	bot.Username,
	//	locales.Get(loginSession.UserLanguageCode).Authorized,
	//	locales.Get(loginSession.UserLanguageCode).AuthorizedResponseMessage)
	// TODO: check if 301 URI scheme redirect works well on all platforms
	http.Redirect(w, r, "tg://resolve?domain="+bot.Username, http.StatusMovedPermanently)
}

// middleware provides some useful middlewares for the server
func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { // returns an HTTP 500 response if the next handler got panicked
			if err := recover(); err != nil {
				log.Errorf("error recovered in request \"%s %s\": %v", r.Method, r.URL.Path, err)
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, InternalErrorResponseBody)
				return
			}
		}()

		// gets client's real IP if serving behind Cloudflare
		if ip := r.Header.Get("Cf-Connecting-Ip"); ip != "" {
			r.RemoteAddr = ip
		}

		next.ServeHTTP(w, r)
	})
}
