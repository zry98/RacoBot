package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
	log "github.com/sirupsen/logrus"

	"RacoBot/internal/bot"
	"RacoBot/internal/db"
	"RacoBot/internal/job"
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
	JobsConfig  job.Config    `toml:"jobs,omitempty"`

	TelegramBotWebhookPath  string
	FIBAPIOAuthRedirectPath string
}

// LogConfig represents a configuration for the global logger
type LogConfig struct {
	Level string `toml:"level,omitempty"`
	Path  string `toml:"path,omitempty"`
}

// LoadConfig loads a configuration from the file ./config.toml
func LoadConfig(path string) (c Config) {
	f, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read config file: %v", err)
	}
	if err = toml.Unmarshal(f, &c); err != nil {
		log.Fatalf("failed to parse config file: %v", err)
	}

	c.setupLogger()
	if err = c.setupHTTPServer(); err != nil {
		log.Fatal(err)
	}
	return
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
		f, err := os.OpenFile(c.Log.Path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("failed to open log file: %v", err)
		}
		log.SetOutput(io.MultiWriter(os.Stdout, f))
	}
}

// TLSConfig represents a configuration for TLS of the HTTP server
type TLSConfig struct {
	ServerName      string `toml:"server_name,omitempty"`
	CertificatePath string `toml:"certificate_path,omitempty"`
	PrivateKeyPath  string `toml:"private_key_path,omitempty"`
}

// setupHTTPServer sets up the HTTP server configuration
func (c *Config) setupHTTPServer() error {
	if c.Host == "" {
		c.Host = "localhost"
	}
	if c.Port == 0 {
		c.Port = 8080
	}

	if err := c.setupRoutingPaths(); err != nil {
		return fmt.Errorf("failed to setup routing paths: %v", err)
	}

	if c.TLS.ServerName != "" { // TLS is enabled
		u, err := url.Parse(c.TLS.ServerName)
		if err != nil {
			return fmt.Errorf("failed to parse TLS server name: %v", err)
		}

		if c.TLS.CertificatePath != "" && c.TLS.PrivateKeyPath != "" {
			cert, err := tls.LoadX509KeyPair(c.TLS.CertificatePath, c.TLS.PrivateKeyPath)
			if err != nil {
				return fmt.Errorf("failed to load TLS certificate: %v", err)
			}

			srv.TLSConfig = &tls.Config{
				ServerName:   u.Hostname(),
				Certificates: []tls.Certificate{cert},
			}
		}
	}
	return nil
}

// setupRoutingPaths sets up the routing patterns configuration for HTTP server
func (c *Config) setupRoutingPaths() error {
	if c.TelegramBot.WebhookURL != "" {
		u, err := url.Parse(c.TelegramBot.WebhookURL)
		if err != nil {
			return fmt.Errorf("invalid Telegram bot webhook URL: %v", err)
		}
		c.TelegramBotWebhookPath = u.Path
	}

	if c.FIBAPI.OAuthRedirectURI == "" {
		return fmt.Errorf("missing FIB API OAuth redirect URI (`oauth_redirect_uri`) in config")
	}
	u, err := url.Parse(c.FIBAPI.OAuthRedirectURI)
	if err != nil {
		return fmt.Errorf("invalid FIB API OAuth redirect URI: %v", err)
	}
	c.FIBAPIOAuthRedirectPath = u.Path
	return nil
}
