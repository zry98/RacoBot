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
	configPath := flag.String("config", "./config.toml", "Config file path (default: ./config.toml)")
	flag.Parse()
	config = LoadConfig(*configPath)
}

func cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if srv != nil {
		if err := srv.Shutdown(ctx); err != nil {
			log.Errorf("failed to shutdown HTTP server: %v", err)
		}
		log.Debug("HTTP server shutdown")
	}
	jobs.Stop()
	bot.Stop()
	db.Close()
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(err)
		}
	}()

	fibapi.Init(config.FIBAPI)
	db.Init(config.Redis)
	bot.Init(config.TelegramBot)
	defer cleanup()

	jobs.CacheSubjectCodes()
	jobs.Init(config.JobsConfig)

	r := http.NewServeMux()
	r.HandleFunc(config.FIBAPIOAuthRedirectPath, HandleOAuthRedirect) // FIB API OAuth redirect
	if config.TelegramBotWebhookPath != "" {
		r.HandleFunc(config.TelegramBotWebhookPath, HandleBotUpdate) // Telegram Bot update
	}

	srv = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      middleware(r),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	srvShutdown := make(chan struct{})
	go func() { // graceful shutdown
		s := make(chan os.Signal, 1)
		signal.Notify(s, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		<-s
		cleanup()
		close(srvShutdown)
		os.Exit(0)
	}()

	if config.TLS.CertificatePath != "" && config.TLS.PrivateKeyPath != "" { // with TLS
		cert, e := tls.LoadX509KeyPair(config.TLS.CertificatePath, config.TLS.PrivateKeyPath)
		if e != nil {
			log.Errorf("failed to load TLS certificate: %v", e)
			return
		}

		srv.TLSConfig = &tls.Config{
			ServerName:   config.TLS.ServerName,
			Certificates: []tls.Certificate{cert},
		}
		go func() {
			if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Errorf("failed to start HTTP server: %v", err)
				srvShutdown <- struct{}{}
			}
		}()
	} else { // without TLS
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Errorf("failed to start HTTP server: %v", err)
				srvShutdown <- struct{}{}
			}
		}()
	}
	log.Debugf("HTTP server started listening on %s", srv.Addr)
	<-srvShutdown
}
