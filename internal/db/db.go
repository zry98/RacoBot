package db

import (
	"context"
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
	ctx         = context.Background() // TODO: use contexts properly
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
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatal(err)
	}

	RateLimiter = redis_rate.NewLimiter(rdb)
}

// Close closes the DB client
func Close() {
	if err := rdb.Close(); err != nil {
		log.Fatal(err)
	}
}
