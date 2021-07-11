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
	"golang.org/x/oauth2"
)

// LoginSession represents a session of login (FIB API OAuth authorization) procedure
type LoginSession struct {
	State     string
	UserID    int64 `json:"u"`
	MessageID int64 `json:"m"`
}

// User represents a user's info store in database
type User struct {
	ID                  int64  `db:"id"`
	AccessToken         string `db:"access_token"`
	RefreshToken        string `db:"refresh_token"`
	TokenExpiry         int64  `db:"token_expiry"`
	LastNoticeTimestamp int64  `db:"last_notice_timestamp"`
}

// LastState represents a user's last state
// including the FIB API notices response body's hash and the last one notice's `ModifiedAt` UNIX timestamp
type LastState struct {
	NoticesHash     string
	NoticeTimestamp int64
}

// key prefixes
const (
	loginSessionKeyPrefix = "s"
	userTokenKeyPrefix    = "t"
	lastStateKeyPrefix    = "l"
)

// errors
var (
	LoginSessionNotFoundError = errors.New("db: login session not found")
	TokenNotFoundError        = errors.New("db: token not found")
)

// NewLoginSession creates a new login session for a user with the given ID
func NewLoginSession(userID int64) (s LoginSession, err error) {
	// make a random string as state
	buf := make([]byte, 16)
	_, err = rand.Read(buf)
	if err != nil {
		return
	}

	s = LoginSession{
		State:  base64.StdEncoding.EncodeToString(buf),
		UserID: userID,
	}
	return
}

// GetLoginSession gets a login session with the given state
func GetLoginSession(state string) (s LoginSession, err error) {
	key := fmt.Sprintf("%s:%s", loginSessionKeyPrefix, state)
	value, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			err = LoginSessionNotFoundError
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

// GetToken gets the OAuth token of the user with the given ID from KV cache or SQL database
func GetToken(userID int64) (token *oauth2.Token, err error) {
	key := fmt.Sprintf("%s:%d", userTokenKeyPrefix, userID)
	value, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			err = TokenNotFoundError
		}
		return
	}

	fields := strings.Split(value, ",")
	if len(fields) != 3 {
		return nil, fmt.Errorf("db: token format error (%s)", "wrong fields number")
	}

	exp, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("db: token format error (%s)", err.Error())
	}

	token = &oauth2.Token{
		AccessToken:  fields[0],
		TokenType:    "Bearer",
		RefreshToken: fields[1],
		Expiry:       time.Unix(exp, 0),
	}
	return
}

// PutToken puts the given OAuth token of the user with the given ID
func PutToken(userID int64, token *oauth2.Token) error {
	value := fmt.Sprintf("%s,%s,%d", token.AccessToken, token.RefreshToken, token.Expiry.Unix())
	key := fmt.Sprintf("%s:%d", userTokenKeyPrefix, userID)
	return rdb.Set(ctx, key, value, 0).Err() // TODO: put only the access token and set a TTL with its expiry
}

// DeleteToken deletes the OAuth token of the user with the given ID
func DeleteToken(userID int64) error {
	key := fmt.Sprintf("%s:%d", userTokenKeyPrefix, userID)
	return rdb.Del(ctx, key).Err()
}

// GetUserIDs gets all users' IDs from token keys
// TODO: rewrite it to get all users' tokens at once using pipeline instead
func GetUserIDs() (userIDs []int64, err error) {
	keys, err := rdb.Keys(ctx, fmt.Sprintf("%s:*", userTokenKeyPrefix)).Result()
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

// GetLastState gets the last state of the user with the given ID
func GetLastState(userID int64) (lastState LastState, err error) {
	key := fmt.Sprintf("%s:%d", lastStateKeyPrefix, userID)
	value, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			err = nil // return empty last state without error
		}
		return
	}

	fields := strings.Split(value, ",")
	if len(fields) != 2 {
		return LastState{}, fmt.Errorf("db: last state format error (%s)", "wrong fields number")
	}

	lastState.NoticesHash = fields[0]
	lastState.NoticeTimestamp, err = strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return LastState{}, fmt.Errorf("db: last state format error (%s)", err.Error())
	}
	return
}

// PutLastState puts the given last state of the user with the given ID
func PutLastState(userID int64, lastState LastState) error {
	value := fmt.Sprintf("%s,%d", lastState.NoticesHash, lastState.NoticeTimestamp)
	key := fmt.Sprintf("%s:%d", lastStateKeyPrefix, userID)
	return rdb.Set(ctx, key, value, 0).Err()
}
