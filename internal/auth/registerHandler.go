package auth

import (
	"log"
	"net/http"
	"strings"

	db "forum/internal/database"

	"golang.org/x/crypto/bcrypt"
)

// Register a new user
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var cred Credentials

	if r.Method == http.MethodGet {
		db.RenderTemplate(w, "register", map[string]interface{}{
			"Title":       "Registration",
			"Credentials": cred,
		})
		return
	}
	if r.Method == http.MethodPost {
		r.ParseForm()

		cred.Username = strings.TrimSpace(r.FormValue("username"))
		cred.Email = strings.ToLower(strings.TrimSpace(r.FormValue("email")))
		cred.Password = strings.TrimSpace(r.FormValue("password"))

		valid := true
		if len(cred.Username) < 5 || len(cred.Username) > 25 {
			cred.Error.Username = "Username must be between 5 and 25 characters"
			valid = false
		}
		if len(cred.Email) < 5 || len(cred.Email) > 25 {
			cred.Error.Email = "Email must be between 5 and 25 characters"
			valid = false
		}
		if len(cred.Password) < 5 || len(cred.Password) > 25 {
			cred.Error.Password = "Password must be between 5 and 25 characters"
			valid = false
		}

		if !valid {
			w.WriteHeader(http.StatusBadRequest)
			db.RenderTemplate(w, "register", map[string]interface{}{
				"Title":       "Registration",
				"Credentials": cred,
			})
			return
		}

		var count int
		err := db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", cred.Email).Scan(&count)
		if err != nil {
			log.Printf("Error checking email uniqueness: %v", err)
			db.HandleError(w, http.StatusInternalServerError, "Database error")

			return
		}
		if count > 0 {
			w.WriteHeader(http.StatusBadRequest)
			cred.Error.Email = "Email already in use"
			db.RenderTemplate(w, "register", map[string]interface{}{
				"Title":       "Registration",
				"Credentials": cred,
			})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(cred.Password), bcrypt.DefaultCost)
		if err != nil {
			db.HandleError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}

		// Insert the new user into the database
		prep, err := db.DB.Prepare("INSERT INTO users (username, email, password) VALUES (?, ?, ?)")
		if err != nil {
			db.HandleError(w, http.StatusInternalServerError, "Prep error")

			return
		}
		defer prep.Close()

		if _, err = prep.Exec(cred.Username, cred.Email, hashedPassword); err != nil {
			db.HandleError(w, http.StatusInternalServerError, "registration failed")

			return
		}

		// Redirect to login page or success page
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	} else {
		db.HandleError(w, http.StatusMethodNotAllowed, "Invalid method")
		return
	}
}
