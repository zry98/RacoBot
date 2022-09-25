package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"RacoBot/internal/bot"
	"RacoBot/internal/db"
	"RacoBot/internal/jobs"
	"RacoBot/pkg/fibapi"
)

var (
	config Config
	srv    *http.Server
)

func init() {
	defer func() {
		if err := recover(); err != nil {
			log.Fatalf("error recovered: %v", err)
		}
	}()

	sc := make(chan os.Signal)
	signal.Notify(sc, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sc
		log.Info("received SIGTERM, exiting")
		cleanup()
		os.Exit(0)
	}()

	configPath := flag.String("config", "./config.toml", "Config file path (default: ./config.toml)")
	flag.Parse()
	config = LoadConfig(*configPath)
}

func cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Errorf("failed to shutdown HTTP server: %v", err)
	}
	jobs.Close()
	//bot.Close()
	db.Close()
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Fatalf("error recovered: %v", err)
		}
	}()

	fibapi.Init(config.FIBAPI)
	db.Init(config.Redis)
	bot.Init(config.TelegramBot)
	defer cleanup()

	if log.GetLevel() < log.DebugLevel {
		jobs.CacheSubjectCodes()
	}
	jobs.Init(config.JobsConfig)

	r := http.NewServeMux()
	r.HandleFunc(config.TelegramBotWebhookPath, HandleBotUpdate)      // Telegram Bot update
	r.HandleFunc(config.FIBAPIOAuthRedirectPath, HandleOAuthRedirect) // FIB API OAuth redirect

	srv = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      middleware(r),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	var err error
	if config.TLS.CertificatePath != "" && config.TLS.PrivateKeyPath != "" { // with HTTPS
		cert, e := tls.LoadX509KeyPair(config.TLS.CertificatePath, config.TLS.PrivateKeyPath)
		if e != nil {
			log.Errorf("failed to load TLS certificate:", e)
			return
		}

		srv.TLSConfig = &tls.Config{
			ServerName:   config.TLS.ServerName,
			Certificates: []tls.Certificate{cert},
		}
		log.Infof("started listening on %s (HTTPS)", srv.Addr)
		err = srv.ListenAndServeTLS("", "")
	} else { // without HTTPS
		log.Infof("started listening on %s", srv.Addr)
		err = srv.ListenAndServe()
	}
	if err != nil {
		log.Errorf("error returned by HTTP server: %v", err)
		return
	}
}
