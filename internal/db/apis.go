package db

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// key names
const (
	keySubjectCodes = "subject_codes"
	keyUserIDs      = "user_ids"
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
	oauthStateLength           = 15                   // no padding
	OAuthStateHexEncodedLength = 2 * oauthStateLength // for use in HTTP handler check
	//OAuthStateBase64EncodedLength = ((4 * oauthStateLength / 3) + 3) & ^3 // for use in HTTP handler check
)

// NewLoginSession creates a login session for a user with the given ID and language code
func NewLoginSession(userID int64, languageCode string) (LoginSession, error) {
	// make a random string as state
	buf := make([]byte, oauthStateLength)
	if _, err := rand.Read(buf); err != nil {
		return LoginSession{}, err
	}

	return LoginSession{
		State:            hex.EncodeToString(buf),
		UserID:           userID,
		UserLanguageCode: languageCode,
	}, nil
}

// GetLoginSession gets a login session with the given state
func GetLoginSession(state string) (LoginSession, error) {
	key := fmt.Sprintf("%s:%s", keyPrefixLoginSession, state)
	value, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			err = ErrLoginSessionNotFound
		}
		return LoginSession{}, err
	}

	var s LoginSession
	if err = json.Unmarshal([]byte(value), &s); err != nil {
		return LoginSession{}, err
	}
	s.State = state
	return s, nil
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
func GetUser(userID int64) (User, error) {
	key := fmt.Sprintf("%s:%d", keyPrefixUser, userID)
	value, err := rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			err = ErrUserNotFound
		}
		return User{}, err
	}

	var u User
	if err = json.Unmarshal([]byte(value), &u); err != nil {
		return User{}, err
	}
	u.ID = userID
	return u, nil
}

// PutUser puts the given user and indexes its ID in the user IDs Set (atomically)
func PutUser(user User) error {
	key := fmt.Sprintf("%s:%d", keyPrefixUser, user.ID)
	value, err := json.Marshal(user)
	if err != nil {
		return err
	}
	_, err = rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, key, value, ttlUser)
		pipe.SAdd(ctx, keyUserIDs, user.ID)
		return nil
	})
	return err
}

// DelUser deletes a user with the given ID and removes it from the user IDs Set (atomically)
func DelUser(userID int64) error {
	key := fmt.Sprintf("%s:%d", keyPrefixUser, userID)
	_, err := rdb.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, key)
		pipe.SRem(ctx, keyUserIDs, userID)
		return nil
	})
	return err
}

// GetAllUserIDs gets all user IDs from the user IDs Set
func GetAllUserIDs() ([]int64, error) {
	values, err := rdb.SMembers(ctx, keyUserIDs).Result()
	if err != nil {
		return nil, err
	}
	userIDs := make([]int64, 0, len(values))
	for _, v := range values {
		ID, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
		userIDs = append(userIDs, ID)
	}
	return userIDs, nil
}

// reconcileUserIDIndex rebuilds the user IDs Set from the existing u:<id> keys.
// It migrates deployments created before the index existed and self-heals any drift
func reconcileUserIDIndex() error {
	var ids []any
	iter := rdb.Scan(ctx, 0, fmt.Sprintf("%s:*", keyPrefixUser), 100).Iterator()
	for iter.Next(ctx) {
		ID, err := strconv.ParseInt(strings.TrimPrefix(iter.Val(), keyPrefixUser+":"), 10, 64)
		if err != nil {
			return err
		}
		ids = append(ids, ID)
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	return rdb.SAdd(ctx, keyUserIDs, ids...).Err()
}

// GetSubjectUPCCode gets the UPC code of a subject with the given acronym
func GetSubjectUPCCode(acronym string) (uint32, error) {
	value, err := rdb.HGet(ctx, keySubjectCodes, acronym).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			err = ErrSubjectNotFound
		}
		return 0, err
	}

	i, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(i), nil
}

// PutSubjectUPCCode puts the given UPC code of a subject with the given acronym
func PutSubjectUPCCode(acronym string, code uint32) error {
	value := strconv.FormatUint(uint64(code), 10)
	return rdb.HSet(ctx, keySubjectCodes, acronym, value).Err()
}

// PutSubjectUPCCodes puts the given subject UPC codes in bulk
func PutSubjectUPCCodes(codes map[string]uint32) error {
	values := make(map[string]any, len(codes))
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
