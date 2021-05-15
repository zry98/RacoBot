package db

import (
	"context"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redis_rate/v9"
	log "github.com/sirupsen/logrus"
)

// RedisConfig represents a configuration for redis connection
type RedisConfig struct {
	Addr     string `toml:"address"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	DB       int    `toml:"db"`
}

var (
	ctx         = context.Background() // TODO: use contexts properly
	rdb         *redis.Client
	RateLimiter *redis_rate.Limiter
)

// Init initializes a connection to redis server
func Init(config RedisConfig) {
	addrType := "tcp"
	if strings.HasPrefix(config.Addr, "/") { // for unix sockets
		addrType = "unix"
	}

	rdb = redis.NewClient(&redis.Options{
		Network:  addrType,
		Addr:     config.Addr,
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

// Close closes the connection to redis server
func Close() {
	err := rdb.Close()
	if err != nil {
		log.Fatal(err)
	}
}
