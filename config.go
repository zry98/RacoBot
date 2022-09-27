package main

import (
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
	log "github.com/sirupsen/logrus"

	"RacoBot/internal/bot"
	"RacoBot/internal/db"
	"RacoBot/internal/jobs"
	"RacoBot/pkg/fibapi"
)

// Config represents a complete configuration
type Config struct {
	Host        string        `toml:"host,omitempty"`
	Port        uint16        `toml:"port,omitempty"`
	Log         LogConfig     `toml:"log,omitempty"`
	TLS         TLSConfig     `toml:"tls,omitempty"`
	Redis       db.Config     `toml:"redis"`
	TelegramBot bot.Config    `toml:"telegram_bot"`
	FIBAPI      fibapi.Config `toml:"fib_api"`
	JobsConfig  jobs.Config   `toml:"jobs,omitempty"`

	TelegramBotWebhookPath  string
	FIBAPIOAuthRedirectPath string
}

// LogConfig represents a configuration for the global logger
type LogConfig struct {
	Level string `toml:"level,omitempty"`
	Path  string `toml:"path,omitempty"`
}

// TLSConfig represents a configuration for TLS of HTTP server
type TLSConfig struct {
	ServerName      string `toml:"server_name,omitempty"`
	CertificatePath string `toml:"certificate_path,omitempty"`
	PrivateKeyPath  string `toml:"private_key_path,omitempty"`
}

// LoadConfig loads a configuration from the file ./config.toml
func LoadConfig(path string) (c Config) {
	f, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read config file: %v", err)
	}

	err = toml.Unmarshal(f, &c)
	if err != nil {
		log.Fatalf("failed to parse config file: %v", err)
	}

	if c.Host == "" {
		c.Host = "127.0.0.1"
	}
	if c.Port == 0 {
		c.Port = 8080
	}

	err = c.setupRoutingPaths()
	if err != nil {
		log.Fatalf("failed to setup routing paths: %v", err)
	}

	c.setupLogger()
	return
}

// setupRoutingPaths sets up the routing patterns configuration for HTTP server
func (c *Config) setupRoutingPaths() error {
	if c.TelegramBot.WebhookURL != "" {
		u, err := url.Parse(c.TelegramBot.WebhookURL)
		if err != nil {
			return err
		}
		c.TelegramBotWebhookPath = u.Path
	}

	if c.FIBAPI.OAuthRedirectURI == "" {
		log.Fatalf("missing FIB API OAuth redirect URI in config")
	}
	u, err := url.Parse(c.FIBAPI.OAuthRedirectURI)
	if err != nil {
		return err
	}
	c.FIBAPIOAuthRedirectPath = u.Path
	if c.TLS.ServerName == "" {
		c.TLS.ServerName = u.Hostname()
	}
	return nil
}

// setupLogger sets up the global logger configuration
func (c *Config) setupLogger() {
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})

	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	level, err := log.ParseLevel(c.Log.Level)
	if err != nil {
		log.Fatalf("failed to parse log level: %v", err)
	}
	log.SetLevel(level)
	log.Debugf("log level set to %s", strings.ToUpper(level.String()))
	if level >= log.DebugLevel {
		log.SetReportCaller(true)
	}

	if c.Log.Path != "" {
		f, e := os.OpenFile(c.Log.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if e != nil {
			log.Fatalf("failed to open log file: %v", e)
		}
		log.SetOutput(io.MultiWriter(os.Stdout, f))
	}
}
