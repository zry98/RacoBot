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
	BotUpdateLimitPerSecond            = 2
	OAuthRedirectRequestLimitPerSecond = 10
)

// limit key prefixes
const (
	BotUpdateKeyPrefix            = "r:bot-update"
	OAuthRedirectRequestKeyPrefix = "r:oauth-redirect"
)

// OAuthRedirectRequestAllowed checks if an incoming OAuth redirect request from the given IP address is allowed to get processed
func OAuthRedirectRequestAllowed(ctx context.Context, IP string) bool {
	key := fmt.Sprintf("%s:%s", OAuthRedirectRequestKeyPrefix, IP)
	res, err := db.RateLimiter.Allow(ctx, key, redis_rate.PerSecond(OAuthRedirectRequestLimitPerSecond))
	if err != nil {
		return false
	}
	return res.Allowed != 0
}

// BotUpdateAllowed checks if an incoming Bot Update from a user with the given ID is allowed to get processed
func BotUpdateAllowed(ctx context.Context, userID int64) bool {
	key := fmt.Sprintf("%s:%d", BotUpdateKeyPrefix, userID)
	res, err := db.RateLimiter.Allow(ctx, key, redis_rate.PerSecond(BotUpdateLimitPerSecond))
	if err != nil {
		return false
	}
	return res.Allowed != 0
}
