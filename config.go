package main

import (
	"io"
	"io/ioutil"
	"net/url"
	"os"

	"github.com/pelletier/go-toml"
	log "github.com/sirupsen/logrus"

	"RacoBot/internal/bot"
	"RacoBot/internal/db"
	"RacoBot/internal/jobs"
	"RacoBot/pkg/fibapi"
)

// Config represents a complete configuration
type Config struct {
	Host                    string              `toml:"host"`
	Port                    int                 `toml:"port"`
	Log                     LogConfig           `toml:"log"`
	TLS                     TLSConfig           `toml:"tls"`
	Redis                   db.RedisConfig      `toml:"redis"`
	TelegramBot             bot.BotConfig       `toml:"telegram_bot"`
	FIBAPI                  fibapi.FIBAPIConfig `toml:"fib_api"`
	JobsConfig              jobs.JobsConfig     `toml:"jobs"`
	TelegramBotWebhookPath  string
	FIBAPIOAuthRedirectPath string
}

// LogConfig represents a configuration for logger
type LogConfig struct {
	Level string `toml:"level"`
	File  string `toml:"file"`
}

// TLSConfig represents a configuration for TLS of HTTP server
type TLSConfig struct {
	Certificate string `toml:"certificate"`
	PrivateKey  string `toml:"private_key"`
}

// LoadConfig loads a configuration from the file ./config.toml
// TODO: implement config file path option
func LoadConfig() (c Config) {
	f, err := ioutil.ReadFile("./config.toml")
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
		log.Info("Log level set to: ", l)
	}

	if c.Log.File != "" {
		f, err := os.OpenFile(c.Log.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal("failed to open log file: ", err)
		}

		log.SetOutput(io.MultiWriter(os.Stdout, f))
	}
}
