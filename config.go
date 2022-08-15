package main

import (
	"io"
	"net/url"
	"os"

	"github.com/pelletier/go-toml/v2"
	log "github.com/sirupsen/logrus"

	"RacoBot/internal/bot"
	"RacoBot/internal/db"
	"RacoBot/internal/jobs"
	"RacoBot/pkg/fibapi"
)

// Config represents a complete configuration
type Config struct {
	Host        string        `toml:"host"`
	Port        uint16        `toml:"port"`
	Log         LogConfig     `toml:"log"`
	TLS         TLSConfig     `toml:"tls"`
	Redis       db.Config     `toml:"redis"`
	TelegramBot bot.Config    `toml:"telegram_bot"`
	FIBAPI      fibapi.Config `toml:"fib_api"`
	JobsConfig  jobs.Config   `toml:"jobs"`

	TelegramBotWebhookPath  string
	FIBAPIOAuthRedirectPath string
}

// LogConfig represents a configuration for the global logger
type LogConfig struct {
	Level string `toml:"level"`
	Path  string `toml:"path"`
}

// TLSConfig represents a configuration for TLS of HTTP server
type TLSConfig struct {
	CertificatePath string `toml:"certificate_path"`
	PrivateKeyPath  string `toml:"private_key_path"`
}

// LoadConfig loads a configuration from the file ./config.toml
func LoadConfig(path string) (c Config) {
	f, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	err = toml.Unmarshal(f, &c)
	if err != nil {
		log.Fatal(err)
	}

	err = c.setupRoutingPaths()
	if err != nil {
		log.Fatal(err)
	}

	c.setupLogger()
	return
}

// setupRoutingPaths sets up the routing patterns configuration for HTTP server
func (c *Config) setupRoutingPaths() (err error) {
	u, err := url.Parse(c.TelegramBot.WebhookURL)
	if err != nil {
		return
	}
	c.TelegramBotWebhookPath = u.Path

	u, err = url.Parse(c.FIBAPI.OAuthRedirectURI)
	if err != nil {
		return
	}
	c.FIBAPIOAuthRedirectPath = u.Path
	return
}

// setupLogger sets up the global logger configuration
func (c *Config) setupLogger() {
	log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})

	if c.Log.Level != "" {
		l, err := log.ParseLevel(c.Log.Level)
		if err != nil {
			log.Fatal("failed to parse log level string in config file: ", err)
		}

		log.SetLevel(l)
		log.Info("log level set to: ", l)
	}

	if c.Log.Path != "" {
		f, err := os.OpenFile(c.Log.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal("failed to open log file: ", err)
		}

		log.SetOutput(io.MultiWriter(os.Stdout, f))
	}
}
