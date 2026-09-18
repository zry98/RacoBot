package ratelimiter

import (
	"context"
	"fmt"

	"github.com/go-redis/redis_rate/v10"
	log "github.com/sirupsen/logrus"

	"RacoBot/internal/db"
)

// limits
var (
	limitBotUpdate            = redis_rate.PerSecond(2)
	limitOAuthRedirectRequest = redis_rate.PerMinute(3)
	limitLoginCommand         = redis_rate.PerMinute(3)
)

// limit key prefixes
const (
	keyPrefixBotUpdate            = "b"
	keyPrefixOAuthRedirectRequest = "o"
	keyPrefixLoginCommand         = "l"
)

// allowed reports whether a request identified by the given key is within the given limit
func allowed(ctx context.Context, key string, limit redis_rate.Limit) bool {
	res, err := db.RateLimiter.Allow(ctx, key, limit)
	if err != nil {
		log.Warnf("rate limiter error for key %q, allowing request: %v", key, err)
		return true
	}
	return res.Allowed != 0
}

// BotUpdateAllowed checks if an incoming Bot Update from a user with the given ID is allowed to get processed
func BotUpdateAllowed(ctx context.Context, userID int64) bool {
	return allowed(ctx, fmt.Sprintf("%s:%d", keyPrefixBotUpdate, userID), limitBotUpdate)
}

// OAuthRedirectRequestAllowed checks if an incoming OAuth redirect request from the given IP address is allowed to get processed
func OAuthRedirectRequestAllowed(ctx context.Context, IP string) bool {
	return allowed(ctx, fmt.Sprintf("%s:%s", keyPrefixOAuthRedirectRequest, IP), limitOAuthRedirectRequest)
}

// LoginCommandAllowed checks if an incoming /login command from a user with the given ID is allowed to get processed
func LoginCommandAllowed(userID int64) bool {
	return allowed(context.Background(), fmt.Sprintf("%s:%d", keyPrefixLoginCommand, userID), limitLoginCommand)
}
