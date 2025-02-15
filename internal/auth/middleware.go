package auth

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	db "forum/internal/database"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Default context with no user
		ctx := r.Context()
		userData := ContextUser{
			LoggedIn: false,
			UserID:   0,
			Username: "",
		}

		// Check for session cookie
		cookie, err := r.Cookie(SessionCookieName)
		if err == nil {
			prep, err := db.DB.Prepare(`
                SELECT s.user_id, s.expires_at, u.username 
                FROM sessions s
                JOIN users u ON s.user_id = u.id 
                WHERE s.id = ? AND s.expires_at > ?
            `)
			if err != nil {
				log.Println("DB prepare error:", err)
			}
			if err == nil {
				defer prep.Close()
				var session Sessions
				err = prep.QueryRow(cookie.Value, time.Now()).Scan(
					&session.UserID,
					&session.ExpiresAt,
					&session.Username,
				)
				if err != nil {
					if err == sql.ErrNoRows {
						log.Println("No session found for the provided cookie.")
					} else {
						log.Println("Error during session lookup:", err)
						db.HandleError(w, http.StatusInternalServerError, "Internal server error")
						return
					}
				}
				if err == nil {
					userData.LoggedIn = true
					userData.UserID = session.UserID
					userData.Username = session.Username
					log.Printf("Session set to user: %v", session.Username)
				} else {
					log.Println("Invalid or expired session - remove cookie")
					http.SetCookie(w, &http.Cookie{
						Name:     SessionCookieName,
						Value:    "",
						Path:     "/",
						Expires:  time.Now().Add(-1 * time.Hour),
						MaxAge:   -1,
						HttpOnly: true,
						Secure:   *secureCookie,
						SameSite: http.SameSiteLaxMode,
					})
				}
			}
		}

		// Add user data to context
		ctx = context.WithValue(ctx, UserKey, userData)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userData, ok := r.Context().Value(UserKey).(ContextUser)
		if !ok || !userData.LoggedIn {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}
