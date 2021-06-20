package db

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"golang.org/x/oauth2"

	"RacoBot/pkg/fibapi"
)

// LoginSession represents a session of login (FIB API OAuth authorization) procedure
type LoginSession struct {
	State     string
	UserID    int64 `json:"u"`
	MessageID int64 `json:"m"`
}

// key prefixes
const (
	loginSessionKeyPrefix  = "login_session"
	userTokenKeyPrefix     = "token"
	lastNoticeIDKeyPrefix  = "last_nid"
	noticeKeyPrefix        = "n"
	userNoticeIDsKeyPrefix = "nids"
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

// PutToken puts the given OAuth token of the user with the given ID
func PutToken(userID int64, token *oauth2.Token) error {
	value, err := json.Marshal(token)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%d", userTokenKeyPrefix, userID)
	return rdb.Set(ctx, key, value, 0).Err()
}

// GetToken gets the OAuth token of the user with the given ID
func GetToken(userID int64) (token *oauth2.Token, err error) {
	key := fmt.Sprintf("%s:%d", userTokenKeyPrefix, userID)
	value, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			err = TokenNotFoundError
		}
		return
	}

	err = json.Unmarshal([]byte(value), &token)
	return

	// TODO: get rid of tokens' json serialization & deserialization?
	//refreshToken, err := rdb.Get(ctx, "fibapi:token:test:refresh_token").Result()
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//accessToken, err := rdb.Get(ctx, "fibapi:token:test:access_token").Result()
	//if err != nil {
	//	if err == redis.Nil {
	//		return &oauth2.Token{
	//			RefreshToken: refreshToken,
	//			Expiry:       time.Now().Add(-10 * time.Second),
	//		}
	//	}
	//	log.Fatal(err)
	//}
	//
	//accessTokenTTL, err := rdb.TTL(ctx, "fibapi:token:test:access_token").Result()
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//return &oauth2.Token{
	//	AccessToken:  accessToken,
	//	RefreshToken: refreshToken,
	//	Expiry:       time.Now().Add(accessTokenTTL),
	//}
}

// GetUserIDs gets all users' IDs
// TODO: rewrite it to get all users' tokens at once using pipeline instead
func GetUserIDs() (IDs []int64, err error) {
	values, err := rdb.Keys(ctx, fmt.Sprintf("%s:*", userTokenKeyPrefix)).Result()
	if err != nil {
		return
	}

	var userID int64
	for _, value := range values {
		userID, err = strconv.ParseInt(value[6:], 10, 64) // extract userID
		if err != nil {
			return
		}

		IDs = append(IDs, userID)
	}
	return
}

// GetNextUserID gets the next cursor and ID of the next user after the given cursor
// FIXME: is it reliable? it seems not guaranteed to return all keys
//func GetNextUserID(cursor uint64) (nextCursor uint64, userID int64, err error) {
//	values, nextCursor, err := rdb.Scan(ctx, cursor, "token:*", 1).Result()
//	if err != nil {
//		return
//	}
//	if len(values) == 0 {
//		return
//	}
//
//	userID, err = strconv.ParseInt(values[0][6:], 10, 64)
//	return
//}

// GetUserLastNoticeID gets the last sent notice's ID of the user with the given ID
func GetUserLastNoticeID(userID int64) (lastNoticeID int64, err error) {
	key := fmt.Sprintf("%s:%d", lastNoticeIDKeyPrefix, userID)
	value, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil || value == "" || value == "0" {
		// a newly logged-in user
		return 0, nil
	}
	if err != nil {
		return
	}

	lastNoticeID, err = strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, nil
	}
	return
}

// PutUserLastNoticeID puts the last sent notice's ID of the user with the given ID
func PutUserLastNoticeID(userID, lastNoticeID int64) error {
	key := fmt.Sprintf("%s:%d", lastNoticeIDKeyPrefix, userID)
	return rdb.Set(ctx, key, lastNoticeID, 0).Err()
}

// DeleteToken deletes the OAuth token of the user with the given ID
func DeleteToken(userID int64) error {
	key := fmt.Sprintf("%s:%d", userTokenKeyPrefix, userID)
	return rdb.Del(ctx, key).Err()
}

// GetNotice gets a notice with the given ID
func GetNotice(ID int64) (notice fibapi.Notice, err error) {
	key := fmt.Sprintf("%s:%d", noticeKeyPrefix, ID)
	value, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return
	}

	err = json.Unmarshal([]byte(value), &notice)
	return
}

// PutNotice puts the given notice
func PutNotice(notice fibapi.Notice) error {
	value, err := json.Marshal(notice)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s:%d", noticeKeyPrefix, notice.ID)
	return rdb.Set(ctx, key, value, 0).Err()
}

// PutNotices puts the given notices
func PutNotices(notices []fibapi.Notice) error {
	if len(notices) == 0 {
		return nil
	}
	pipe := rdb.TxPipeline()
	for _, notice := range notices {
		key := fmt.Sprintf("%s:%d", noticeKeyPrefix, notice.ID)
		value, err := json.Marshal(notice)
		if err != nil {
			return err
		}
		pipe.Set(ctx, key, string(value), 0) // TODO: expire it at its ExpiresAt
	}
	_, err := pipe.Exec(ctx)
	return err
}

// MSET alternative
//func PutNotices(notices []fibapi.Notice) error {
//	kvs := make([]string, len(notices)*2)
//	i := 0
//	for _, notice := range notices {
//		kvs[i] = fmt.Sprintf("%s:%d", noticeKeyPrefix, notice.ID)
//		value, err := json.Marshal(notice)
//		if err != nil {
//			return err
//		}
//		kvs[i+1] = string(value)
//		i += 2
//	}
//	return rdb.MSet(ctx, kvs).Err()
//}

// GetUserNoticeIDs gets all notice IDs of the user with the given ID
func GetUserNoticeIDs(userID int64) (noticeIDs []int64, err error) {
	key := fmt.Sprintf("%s:%d", userNoticeIDsKeyPrefix, userID)
	values, err := rdb.SMembers(ctx, key).Result()
	if err != nil {
		return
	}

	var ID int64
	for _, v := range values {
		ID, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return
		}
		noticeIDs = append(noticeIDs, ID)
	}
	return
}

// PutUserNoticeIDs puts the given notice IDs of the user with the given ID
func PutUserNoticeIDs(userID int64, noticeIDs []int64) error {
	if len(noticeIDs) == 0 {
		return nil
	}
	values := make([]string, len(noticeIDs))
	for i, ID := range noticeIDs {
		values[i] = strconv.FormatInt(ID, 10)
	}

	key := fmt.Sprintf("%s:%d", userNoticeIDsKeyPrefix, userID)
	return rdb.SAdd(ctx, key, values).Err()
}

// GetUserNotices gets all notices of the user with the given ID
func GetUserNotices(userID int64) (notices []fibapi.Notice, err error) {
	userNoticeIDs, err := GetUserNoticeIDs(userID)
	if err != nil {
		return
	}

	cmds := make([]*redis.StringCmd, len(userNoticeIDs))
	pipe := rdb.TxPipeline()
	for i, noticeID := range userNoticeIDs {
		key := fmt.Sprintf("%s:%d", noticeKeyPrefix, noticeID)
		cmds[i] = pipe.Get(ctx, key)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		return
	}

	var value string
	for _, cmd := range cmds {
		value, err = cmd.Result()
		if err != nil {
			return
		}

		var notice fibapi.Notice
		err = json.Unmarshal([]byte(value), &notice)
		if err != nil {
			return
		}
		notices = append(notices, notice)
	}
	return
}

// MGET alternative
//func GetUserNotices(userID int64) (notices []fibapi.Notice, err error) {
//	userNoticeIDs, err := GetUserNoticeIDs(userID)
//	if err != nil {
//		return
//	}
//
//	keys := make([]string, len(userNoticeIDs))
//	for i, ID := range userNoticeIDs {
//		keys[i] = fmt.Sprintf("%s:%d", noticeKeyPrefix, ID)
//	}
//
//	values, err := rdb.MGet(ctx, keys...).Result()
//	if err != nil {
//		return
//	}
//
//	for _, value := range values {
//		if value != nil {
//			var notice fibapi.Notice
//			err = json.Unmarshal([]byte(value.(string)), &notice)
//			if err != nil {
//				return
//			}
//			notices = append(notices, notice)
//		}
//	}
//	return
//}
