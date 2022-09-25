package ratelimiters

import (
	"context"
	"fmt"

	"github.com/go-redis/redis_rate/v9"

	"RacoBot/internal/db"
)

// limits
const (
	// TODO: tune the limits
	botUpdateLimitPerSecond            = 2
	oauthRedirectRequestLimitPerMinute = 3
	loginCommandLimitPerMinute         = 3
)

// limit key prefixes
const (
	botUpdateKeyPrefix            = "b"
	oauthRedirectRequestKeyPrefix = "o"
	loginCommandKeyPrefix         = "l"
)

var (
	botUpdateLimit            = redis_rate.PerSecond(botUpdateLimitPerSecond)
	oauthRedirectRequestLimit = redis_rate.PerMinute(oauthRedirectRequestLimitPerMinute)
	loginCommandLimit         = redis_rate.PerMinute(loginCommandLimitPerMinute)
)

// BotUpdateAllowed checks if an incoming Bot Update from a user with the given ID is allowed to get processed
func BotUpdateAllowed(ctx context.Context, userID int64) bool {
	key := fmt.Sprintf("%s:%d", botUpdateKeyPrefix, userID)
	res, err := db.RateLimiter.Allow(ctx, key, botUpdateLimit)
	if err != nil {
		return false
	}
	return res.Allowed != 0
}

// OAuthRedirectRequestAllowed checks if an incoming OAuth redirect request from the given IP address is allowed to get processed
func OAuthRedirectRequestAllowed(ctx context.Context, IP string) bool {
	key := fmt.Sprintf("%s:%s", oauthRedirectRequestKeyPrefix, IP)
	res, err := db.RateLimiter.Allow(ctx, key, oauthRedirectRequestLimit)
	if err != nil {
		return false
	}
	return res.Allowed != 0
}

// LoginCommandAllowed checks if an incoming /login command from a user with the given ID is allowed to get processed
func LoginCommandAllowed(userID int64) bool {
	key := fmt.Sprintf("%s:%d", loginCommandKeyPrefix, userID)
	res, err := db.RateLimiter.Allow(context.Background(), key, loginCommandLimit)
	if err != nil {
		return false
	}
	return res.Allowed != 0
}
