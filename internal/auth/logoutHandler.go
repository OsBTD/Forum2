package auth

import (
	"net/http"
	"time"

	db "forum/internal/database"
)

// Logout Handler
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		db.HandleError(w, http.StatusBadRequest, "Not logged in")
		return
	}
	prep, err := db.DB.Prepare("DELETE FROM sessions WHERE id = ?")
	if err != nil {
		db.HandleError(w, http.StatusInternalServerError, "Prep error")
		return
	}
	defer prep.Close()

	if _, err = prep.Exec(cookie.Value); err != nil {
		db.HandleError(w, http.StatusInternalServerError, "Logout failed")
		return
	}

	// Expire cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-1 * time.Hour),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   *secureCookie,
		SameSite: http.SameSiteLaxMode, // Or SameSiteStrictMode
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
