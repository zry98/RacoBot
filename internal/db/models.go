package db

import "errors"

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
	LastNoticeTimestamp int64  `json:"t,omitempty"`
	MuteBannerNotices   bool   `json:"i,omitempty"`
}

// errors
var (
	ErrLoginSessionNotFound = errors.New("db: login session not found")
	ErrUserNotFound         = errors.New("db: user not found")
	ErrSubjectNotFound      = errors.New("db: subject not found")
)
