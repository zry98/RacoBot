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

	fibapi.Init(config.FIBAPI)
	db.Init(config.Redis)
	bot.Init(config.TelegramBot)
}

func main() {
	defer db.Close()

	// cache all subject UPC codes
	jobs.CacheSubjectCodes()
	jobs.Init(config.JobsConfig)

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
			log.Fatalf("failed to load TLS certificate: %v", err)
			return
		}

		srv.TLSConfig = &tls.Config{
			ServerName:   config.TLS.ServerName,
			Certificates: []tls.Certificate{cert},
		}
		log.Fatal(srv.ListenAndServeTLS("", ""))
	} else { // without HTTPS
		log.Fatal(srv.ListenAndServe())
	}
}
