package ratelimiter

import (
	"context"
	"fmt"

	"github.com/go-redis/redis_rate/v10"

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

// BotUpdateAllowed checks if an incoming Bot Update from a user with the given ID is allowed to get processed
func BotUpdateAllowed(ctx context.Context, userID int64) bool {
	key := fmt.Sprintf("%s:%d", keyPrefixBotUpdate, userID)
	res, err := db.RateLimiter.Allow(ctx, key, limitBotUpdate)
	if err != nil {
		panic(err)
	}
	return res.Allowed != 0
}

// OAuthRedirectRequestAllowed checks if an incoming OAuth redirect request from the given IP address is allowed to get processed
func OAuthRedirectRequestAllowed(ctx context.Context, IP string) bool {
	key := fmt.Sprintf("%s:%s", keyPrefixOAuthRedirectRequest, IP)
	res, err := db.RateLimiter.Allow(ctx, key, limitOAuthRedirectRequest)
	if err != nil {
		panic(err)
	}
	return res.Allowed != 0
}

// LoginCommandAllowed checks if an incoming /login command from a user with the given ID is allowed to get processed
func LoginCommandAllowed(userID int64) bool {
	key := fmt.Sprintf("%s:%d", keyPrefixLoginCommand, userID)
	res, err := db.RateLimiter.Allow(context.Background(), key, limitLoginCommand)
	if err != nil {
		panic(err)
	}
	return res.Allowed != 0
}
