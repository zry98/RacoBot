package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"RacoBot/internal/bot"
	"RacoBot/internal/db"
	"RacoBot/internal/jobs"
	"RacoBot/pkg/fibapi"
)

var config Config

func init() {
	config = LoadConfig()
	db.Init(config.Redis)
	bot.Init(config.TelegramBot)
	fibapi.Init(config.FIBAPI)

	jobs.Init(config.JobsConfig)
}

func main() {
	defer db.Close()

	r := http.NewServeMux()
	r.HandleFunc(config.TelegramBotWebhookPath, HandleBotUpdate)      // Telegram Bot update
	r.HandleFunc(config.FIBAPIOAuthRedirectPath, HandleOAuthRedirect) // FIB API OAuth redirect

	//r.HandleFunc("/debug", HandleDebug)
	//r.HandleFunc("/debug/pprof/", pprof.Index)
	//r.HandleFunc("/debug/pprof/{action}", pprof.Index)
	//r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	//r.HandleFunc("/debug/pprof/profile", pprof.Profile)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      middleware(r),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if config.TLS.Certificate != "" && config.TLS.PrivateKey != "" {
		cert, err := tls.LoadX509KeyPair(config.TLS.Certificate, config.TLS.PrivateKey)
		if err != nil {
			log.Fatal(err)
			return
		}

		srv.TLSConfig = &tls.Config{
			MinVersion:               tls.VersionTLS12,
			Certificates:             []tls.Certificate{cert},
			CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			},
		}
		log.Fatal(srv.ListenAndServeTLS("", ""))
	} else {
		log.Fatal(srv.ListenAndServe())
	}
}
