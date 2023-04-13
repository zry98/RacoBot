package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redis_rate/v9"
	log "github.com/sirupsen/logrus"
)

// Config represents a configuration for redis connection
type Config struct {
	Address  string `toml:"address"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	DB       int    `toml:"db"`
}

var (
	ctx         = context.Background()
	rdb         *redis.Client
	RateLimiter *redis_rate.Limiter
)

// Init initializes the DB clients
func Init(config Config) {
	addrType := "tcp"
	if strings.HasPrefix(config.Address, "/") { // for unix sockets
		addrType = "unix"
	}

	rdb = redis.NewClient(&redis.Options{
		Network:  addrType,
		Addr:     config.Address,
		Username: config.Username,
		Password: config.Password,
		DB:       config.DB,
	})
	res, err := rdb.Ping(ctx).Result()
	if err != nil || res != "PONG" {
		panic(fmt.Errorf("failed to connect to DB: %v", err))
	}

	RateLimiter = redis_rate.NewLimiter(rdb)

	log.Debug("DB connected")
}

// Close closes the DB
func Close() {
	if rdb != nil {
		if err := rdb.Close(); err != nil {
			log.Errorf("failed to close DB: %v", err)
		}
	}
	log.Debug("DB closed")
}
