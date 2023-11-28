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
	"RacoBot/internal/job"
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
	job.Stop()
	bot.Stop()
	db.Close()
}

func main() {
	defer cleanup()
	fibapi.Init(config.FIBAPI)
	db.Init(config.Redis)
	bot.Init(config.TelegramBot)
	job.CacheSubjectCodes()
	job.Init(config.JobsConfig)

	shutdown := make(chan struct{})
	go func() { // graceful shutdown
		s := make(chan os.Signal, 1)
		signal.Notify(s, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		<-s
		close(shutdown)
		cleanup()
		os.Exit(0)
	}()

	r := http.NewServeMux()
	r.HandleFunc(config.FIBAPIOAuthRedirectPath, HandleOAuthRedirect) // FIB API OAuth redirect
	if config.TelegramBotWebhookPath != "" {                          // Telegram Bot update by webhook
		r.HandleFunc(config.TelegramBotWebhookPath, HandleBotUpdate)
	}
	if config.MailtoLinkRedirectPath != "" { // mailto link redirect
		r.HandleFunc(config.MailtoLinkRedirectPath, HandleMailtoLinkRedirect)
	}

	srv = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Handler:      middleware(r),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	if log.GetLevel() >= log.DebugLevel {
		srv.ReadTimeout = 1 * time.Minute
		srv.WriteTimeout = 1 * time.Minute
	}
	if len(config.TLS.Certificates) > 0 { // with TLS
		srv.TLSConfig = &tls.Config{
			ServerName:   config.TLS.ServerName,
			Certificates: config.TLS.Certificates,
		}
		go func() {
			if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Errorf("failed to start HTTP server: %v", err)
				shutdown <- struct{}{}
			}
		}()
	} else { // without TLS
		go func() {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Errorf("failed to start HTTP server: %v", err)
				shutdown <- struct{}{}
			}
		}()
	}
	log.Debugf("started HTTP server listening on %s", srv.Addr)
	<-shutdown
}
