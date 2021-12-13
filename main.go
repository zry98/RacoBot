package main

import (
	"crypto/tls"
	"flag"
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
	configPath := flag.String("config", "./config.toml", "Config file path (default: ./config.toml)")
	flag.Parse()
	config = LoadConfig(*configPath)

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

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      middleware(r),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if config.TLS.CertificatePath != "" && config.TLS.PrivateKeyPath != "" { // with HTTPS
		cert, err := tls.LoadX509KeyPair(config.TLS.CertificatePath, config.TLS.PrivateKeyPath)
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
	} else { // without HTTPS
		log.Fatal(srv.ListenAndServe())
	}
}
