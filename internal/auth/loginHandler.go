package auth

import (
	"net/http"
	"strings"
	"time"

	db "forum/internal/database"

	"golang.org/x/crypto/bcrypt"
)

// User login
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var cred Credentials
	if r.Method == http.MethodGet {
		db.RenderTemplate(w, "login", map[string]interface{}{
			"Title":       "Login Page",
			"Credentials": cred,
		})
		return
	}
	if r.Method == http.MethodPost {
		r.ParseForm()

		cred.Email = strings.ToLower(strings.TrimSpace(r.FormValue("email")))
		cred.Password = strings.TrimSpace(r.FormValue("password"))

		valid := true
		if len(cred.Email) < 5 || len(cred.Email) > 30 {
			cred.Error.Email = "Email must be between 5 and 100 characters"
			valid = false
		}
		if len(cred.Password) < 5 || len(cred.Password) > 30 {
			cred.Error.Password = "Password must be between 5 and 100 characters"
			valid = false
		}

		// If validation fails, show errors
		if !valid {
			w.WriteHeader(http.StatusBadRequest)
			db.RenderTemplate(w, "login", map[string]interface{}{
				"Title":       "Login",
				"Credentials": cred,
			})
			return
		}

		prep, err := db.DB.Prepare("SELECT id, email, username, password FROM users WHERE email = ?")
		if err != nil {
			db.HandleError(w, http.StatusInternalServerError, "Prep error")
			return
		}
		defer prep.Close()

		user := users{}
		if err = prep.QueryRow(cred.Email).Scan(&user.userID, &user.dbEmail, &user.username, &user.storedHash); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			// timing attack prevention (always returning an error on failed login)
			_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$dummy"), []byte(cred.Password))
			cred.Error.Password = "Invalid Password"
			cred.Error.Email = "Invalid email"
			db.RenderTemplate(w, "login", map[string]interface{}{
				"Title":       "Login",
				"Credentials": cred,
			})
			return
		}

		if err = bcrypt.CompareHashAndPassword([]byte(user.storedHash), []byte(cred.Password)); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			cred.Error.Password = "Invalid Password"
			db.RenderTemplate(w, "login", map[string]interface{}{
				"Title":       "Login",
				"Credentials": cred,
			})
			return
		}

		// avoid multiple active sessions for the same user
		_, err = db.DB.Exec("DELETE FROM sessions WHERE user_id = ?", user.userID)
		if err != nil {
			db.HandleError(w, http.StatusInternalServerError, "Failed to delete existing session")
			return
		}

		genSessionID := GenerateSessionID()
		expiresAT := time.Now().Add(24 * time.Hour)
		if genSessionID == "" {
			db.HandleError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		prep, err = db.DB.Prepare("INSERT INTO sessions (id , user_id, expires_at) VALUES (?,?,?)")
		if err != nil {
			db.HandleError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		defer prep.Close()

		if _, err = prep.Exec(genSessionID, user.userID, expiresAT); err != nil {
			db.HandleError(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		setSessionCookie(w, genSessionID, expiresAT)

		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		db.HandleError(w, http.StatusMethodNotAllowed, "Invalid method")
		return
	}
}
