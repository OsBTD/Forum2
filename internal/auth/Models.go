package auth

import (
	"flag"
	"time"
)

type contextKey string

const (
	SessionCookieName            = "session_token01"
	UserKey           contextKey = "userID"
)

var secureCookie = flag.Bool("secure-cookie01", false, "Set secure cookie flag")

type users struct {
	userID     int
	storedHash string
	dbEmail    string
	username   string
}

type Credentials struct {
	Username string
	Email    string
	Password string
	Error    Error
}
type Error struct {
	Username string
	Email    string
	Password string
}

type Sessions struct {
	UserID    int
	ExpiresAt time.Time
	Username  string
}

type ContextUser struct {
	LoggedIn bool
	UserID   int
	Username string
}
