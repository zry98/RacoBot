package db

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// LoginSession represents a session of login (FIB API OAuth authorization) procedure
type LoginSession struct {
	State              string
	UserID             int64  `json:"u"`
	LoginLinkMessageID int64  `json:"m"`
	UserLanguageCode   string `json:"l"`
}

// User represents a user's data
type User struct {
	ID                  int64  `json:"-"`
	AccessToken         string `json:"a"`
	RefreshToken        string `json:"r"`
	TokenExpiry         int64  `json:"e"`
	LanguageCode        string `json:"l,omitempty"`
	LastNoticesHash     string `json:"h,omitempty"`
	LastNoticeTimestamp int64  `json:"t,omitempty"`
}

// key prefixes
const (
	loginSessionKeyPrefix = "s"
	userKeyPrefix         = "u"
)

// errors
var (
	ErrLoginSessionNotFound = errors.New("db: login session not found")
	ErrUserNotFound         = errors.New("db: user not found")
)

// NewLoginSession creates a new login session for a user with the given ID
func NewLoginSession(userID int64, languageCode string) (s LoginSession, err error) {
	// make a random string as state
	buf := make([]byte, 16)
	_, err = rand.Read(buf)
	if err != nil {
		return
	}

	s = LoginSession{
		State:            base64.StdEncoding.EncodeToString(buf),
		UserID:           userID,
		UserLanguageCode: languageCode,
	}
	return
}

// GetLoginSession gets a login session with the given state
func GetLoginSession(state string) (s LoginSession, err error) {
	key := fmt.Sprintf("%s:%s", loginSessionKeyPrefix, state)
	value, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			err = ErrLoginSessionNotFound
		}
		return
	}

	err = json.Unmarshal([]byte(value), &s)
	return
}

// PutLoginSession puts the given login session
func PutLoginSession(s LoginSession) error {
	key := fmt.Sprintf("%s:%s", loginSessionKeyPrefix, s.State)
	value, err := json.Marshal(s)
	if err != nil {
		return err
	}

	return rdb.Set(ctx, key, value, 10*time.Minute).Err() // expires in 10 minutes
}

// DeleteLoginSession deletes a login session with the given state
func DeleteLoginSession(state string) error {
	key := fmt.Sprintf("%s:%s", loginSessionKeyPrefix, state)
	return rdb.Del(ctx, key).Err()
}

// GetUser gets the user with the given ID
func GetUser(userID int64) (user User, err error) {
	key := fmt.Sprintf("%s:%d", userKeyPrefix, userID)
	value, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			err = ErrUserNotFound
		}
		return
	}

	if err = json.Unmarshal([]byte(value), &user); err != nil {
		return
	}

	user.ID = userID
	return
}

// PutUser puts the given user
func PutUser(user User) error {
	value, err := json.Marshal(user)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s:%d", userKeyPrefix, user.ID)
	return rdb.Set(ctx, key, value, 0).Err()
}

// DeleteUser deletes the user with the given ID
func DeleteUser(userID int64) error {
	key := fmt.Sprintf("%s:%d", userKeyPrefix, userID)
	return rdb.Del(ctx, key).Err()
}

// GetUserIDs gets all users' IDs from keys
func GetUserIDs() (userIDs []int64, err error) {
	keys, err := rdb.Keys(ctx, fmt.Sprintf("%s:*", userKeyPrefix)).Result()
	if err != nil {
		return
	}

	var userID int64
	for _, key := range keys {
		userID, err = strconv.ParseInt(strings.Split(key, ":")[1], 10, 64)
		if err != nil {
			return
		}
		userIDs = append(userIDs, userID)
	}
	return
}
