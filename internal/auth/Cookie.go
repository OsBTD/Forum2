package auth

import (
	"log"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
)

func GenerateSessionID() string {
	id, err := uuid.NewV4()
	if err != nil {
		log.Printf("UUID generation error: %v", err)
		return ""
	}
	return id.String()
}

func setSessionCookie(w http.ResponseWriter, sessionID string, expireAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Expires:  expireAt,
		HttpOnly: true,
		Secure:   *secureCookie,        // set true by a flag in https
		SameSite: http.SameSiteLaxMode, // CSRF ATTACK prevention
		Path:     "/",
	})
}
