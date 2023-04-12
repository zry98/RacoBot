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

// key names
const (
	keySubjectCodes = "subject_codes"
)

// key name prefixes
const (
	keyPrefixLoginSession = "l"
	keyPrefixUser         = "u"
)

// key expirations
const (
	ttlLoginSession = 10 * time.Minute     // 10 minutes
	ttlUser         = 0 * time.Second      // no expiration
	ttlSubjectCode  = time.Hour * 24 * 150 // 150 days
)

const (
	oauthStateLength              = 15                                    // no padding
	OAuthStateBase64EncodedLength = ((4 * oauthStateLength / 3) + 3) & ^3 // for use in HTTP handler
)

// NewLoginSession creates a login session for a user with the given ID and language code
func NewLoginSession(userID int64, languageCode string) (s LoginSession, err error) {
	// make a random string as state
	buf := make([]byte, oauthStateLength)
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
	key := fmt.Sprintf("%s:%s", keyPrefixLoginSession, state)
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
	key := fmt.Sprintf("%s:%s", keyPrefixLoginSession, s.State)
	value, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, key, value, ttlLoginSession).Err()
}

// DelLoginSession deletes a login session with the given state
func DelLoginSession(state string) error {
	key := fmt.Sprintf("%s:%s", keyPrefixLoginSession, state)
	return rdb.Del(ctx, key).Err()
}

// GetUser gets a user with the given ID
func GetUser(userID int64) (user User, err error) {
	key := fmt.Sprintf("%s:%d", keyPrefixUser, userID)
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
	key := fmt.Sprintf("%s:%d", keyPrefixUser, user.ID)
	value, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return rdb.Set(ctx, key, value, ttlUser).Err()
}

// DelUser deletes a user with the given ID
// TODO: add userIDs to a set?
func DelUser(userID int64) error {
	key := fmt.Sprintf("%s:%d", keyPrefixUser, userID)
	return rdb.Del(ctx, key).Err()
}

// GetAllUserIDs gets all user IDs
// TODO: get userIDs from a set?
func GetAllUserIDs() (userIDs []int64, err error) {
	keys, err := rdb.Keys(ctx, fmt.Sprintf("%s:*", keyPrefixUser)).Result()
	if err != nil {
		if err == redis.Nil {
			err = nil
		}
		return
	}

	userIDs = make([]int64, 0, len(keys))
	var userID int64
	for _, key := range keys {
		userID, err = strconv.ParseInt(strings.TrimPrefix(key, keyPrefixUser+":"), 10, 64)
		if err != nil {
			return
		}
		userIDs = append(userIDs, userID)
	}
	return
}

// GetSubjectUPCCode gets the UPC code of a subject with the given acronym
func GetSubjectUPCCode(acronym string) (code uint32, err error) {
	value, err := rdb.HGet(ctx, keySubjectCodes, acronym).Result()
	if err != nil {
		if err == redis.Nil {
			err = ErrSubjectNotFound
		}
		return
	}
	i, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return
	}
	code = uint32(i)
	return
}

// PutSubjectUPCCode puts the given UPC code of a subject with the given acronym
func PutSubjectUPCCode(acronym string, code uint32) error {
	value := strconv.FormatUint(uint64(code), 10)
	return rdb.HSet(ctx, keySubjectCodes, acronym, value).Err()
}

// PutSubjectUPCCodes puts the given subject UPC codes in bulk
func PutSubjectUPCCodes(codes map[string]uint32) error {
	values := make(map[string]interface{}, len(codes))
	for acronym, code := range codes {
		values[acronym] = strconv.FormatUint(uint64(code), 10)
	}
	if err := rdb.HSet(ctx, keySubjectCodes, values).Err(); err != nil {
		return err
	}
	return rdb.Expire(ctx, keySubjectCodes, ttlSubjectCode).Err()
}

// DelAllSubjectUPCCodes deletes all subject UPC codes
func DelAllSubjectUPCCodes() error {
	return rdb.Del(ctx, keySubjectCodes).Err()
}
