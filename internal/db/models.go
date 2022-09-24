package db

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// LoginSession represents a session of login (FIB API OAuth authorization) procedure
type LoginSession struct {
	UserID             int64  `json:"u"`
	LoginLinkMessageID int64  `json:"m"`
	State              string `json:"-"`
	UserLanguageCode   string `json:"l"`
}

// User represents a user's data
type User struct {
	ID                  int64  `json:"-"`
	TokenExpiry         int64  `json:"e"`
	AccessToken         string `json:"a"`
	RefreshToken        string `json:"r"`
	LanguageCode        string `json:"l,omitempty"`
	LastNoticesDigest   string `json:"d,omitempty"`
	LastNoticeTimestamp int64  `json:"t,omitempty"`
}

// key prefixes
const (
	loginSessionKeyPrefix = "l"
	userKeyPrefix         = "u"
	subjectKeyPrefix      = "s"
)

// key expirations
const (
	subjectKeyExpiration = time.Hour * 24 * 150 // 150 days
)

// errors
var (
	ErrLoginSessionNotFound = fmt.Errorf("login session not found")
	ErrUserNotFound         = fmt.Errorf("user not found")
	ErrSubjectNotFound      = fmt.Errorf("subject not found")
)

// NewLoginSession creates a login session for a user with the given ID and language code
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
	if err = json.Unmarshal([]byte(value), &s); err != nil {
		return
	}
	s.State = state
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

// GetUser gets a user with the given ID
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
	key := fmt.Sprintf("%s:%d", userKeyPrefix, user.ID)
	value, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, key, value, 0).Err()
}

// DeleteUser deletes a user with the given ID
func DeleteUser(userID int64) error {
	key := fmt.Sprintf("%s:%d", userKeyPrefix, userID)
	return rdb.Del(ctx, key).Err()
}

// GetAllUserIDs gets all user IDs
func GetAllUserIDs() (userIDs []int64, err error) {
	keys, err := rdb.Keys(ctx, fmt.Sprintf("%s:*", userKeyPrefix)).Result()
	if err != nil {
		if err == redis.Nil {
			err = nil
		}
		return
	}

	userIDs = make([]int64, 0, len(keys))
	var userID int64
	for _, key := range keys {
		userID, err = strconv.ParseInt(strings.TrimPrefix(key, userKeyPrefix+":"), 10, 64)
		if err != nil {
			return
		}
		userIDs = append(userIDs, userID)
	}
	return
}

// GetSubjectUPCCode gets the UPC code of a subject with the given acronym
func GetSubjectUPCCode(acronym string) (code uint32, err error) {
	key := fmt.Sprintf("%s:%s", subjectKeyPrefix, acronym)
	value, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			err = ErrSubjectNotFound
		}
		return
	}
	i, err := strconv.ParseUint(value, 10, 32)
	code = uint32(i)
	return
}

// PutSubjectUPCCode puts the given UPC code of a subject with the given acronym
func PutSubjectUPCCode(acronym string, code uint32) error {
	key := fmt.Sprintf("%s:%s", subjectKeyPrefix, acronym)
	value := strconv.FormatInt(int64(code), 10)
	return rdb.Set(ctx, key, value, subjectKeyExpiration).Err()
}

// PutSubjectUPCCodes puts the given subject UPC codes in bulk
func PutSubjectUPCCodes(codes map[string]uint32) error {
	_, err := rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		for acronym, code := range codes {
			key := fmt.Sprintf("%s:%s", subjectKeyPrefix, acronym)
			value := strconv.FormatInt(int64(code), 10)
			pipe.Set(ctx, key, value, subjectKeyExpiration)
		}
		return nil
	})
	return err
}
