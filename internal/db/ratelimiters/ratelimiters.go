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
	botUpdateKeyPrefix            = "r:bu"
	oauthRedirectRequestKeyPrefix = "r:or"
	loginCommandKeyPrefix         = "r:l"
)

// OAuthRedirectRequestAllowed checks if an incoming OAuth redirect request from the given IP address is allowed to get processed
func OAuthRedirectRequestAllowed(ctx context.Context, IP string) bool {
	key := fmt.Sprintf("%s:%s", oauthRedirectRequestKeyPrefix, IP)
	res, err := db.RateLimiter.Allow(ctx, key, redis_rate.PerMinute(oauthRedirectRequestLimitPerMinute))
	if err != nil {
		return false
	}
	return res.Allowed != 0
}

// BotUpdateAllowed checks if an incoming Bot Update from a user with the given ID is allowed to get processed
func BotUpdateAllowed(ctx context.Context, userID int64) bool {
	key := fmt.Sprintf("%s:%d", botUpdateKeyPrefix, userID)
	res, err := db.RateLimiter.Allow(ctx, key, redis_rate.PerSecond(botUpdateLimitPerSecond))
	if err != nil {
		return false
	}
	return res.Allowed != 0
}

// LoginCommandAllowed checks if an incoming /login command from a user with the given ID is allowed to get processed
func LoginCommandAllowed(userID int64) bool {
	key := fmt.Sprintf("%s:%d", loginCommandKeyPrefix, userID)
	res, err := db.RateLimiter.Allow(context.Background(), key, redis_rate.PerMinute(loginCommandLimitPerMinute))
	if err != nil {
		return false
	}
	return res.Allowed != 0
}
