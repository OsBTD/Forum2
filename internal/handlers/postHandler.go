package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"forum/internal/auth"
	db "forum/internal/database"
)

func AddPostHandler(w http.ResponseWriter, r *http.Request) {
	userData := r.Context().Value(auth.UserKey).(auth.ContextUser)
	categories, err := getCategories()
	if err != nil {
		db.HandleError(w, http.StatusInternalServerError, "Error loading categories")
		return
	}

	if r.Method == http.MethodGet {
		db.RenderTemplate(w, "add_post", map[string]interface{}{
			"Title":              "Add Post",
			"Categories":         categories,
			"LoggedIn":           userData.LoggedIn,
			"Username":           userData.Username,
			"SelectedCategories": []int{},
		})
		return
	}

	if r.Method == http.MethodPost {
		//starting the transaction
		tx, err := db.DB.Begin()
		if err != nil {
			db.HandleError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
		defer tx.Rollback()

		//insert post
		Title := r.FormValue("title")
		Content := r.FormValue("content")
		categoryStrs := r.Form["categories"]

		var selectedCategories []int
		for _, catID := range categoryStrs {
			if id, err := strconv.Atoi(catID); err == nil {
				selectedCategories = append(selectedCategories, id)
			}
		}

		if strings.TrimSpace(Title) == "" || strings.TrimSpace(Content) == "" || len(selectedCategories) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			db.RenderTemplate(w, "add_post", map[string]interface{}{
				"Title":              "Add Post",
				"Categories":         categories,
				"LoggedIn":           userData.LoggedIn,
				"Username":           userData.Username,
				"SelectedCategories": selectedCategories,
				"Error":              "Please fill in all fields and select at least one category.",
			})
			return
		}

		if len(Title) > 50 || len(Content) > 1000 {
			w.WriteHeader(http.StatusBadRequest)
			db.RenderTemplate(w, "add_post", map[string]interface{}{
				"Title":              "Add Post",
				"Categories":         categories,
				"LoggedIn":           userData.LoggedIn,
				"Username":           userData.Username,
				"SelectedCategories": selectedCategories,
				"Error":              "Title or content length exceeded. Title must be <= 50 characters and content <= 1000 characters.",
			})
			return
		}

		result, err := tx.Exec("INSERT INTO posts (user_id, title, content) VALUES (?, ?, ?)",
			userData.UserID, Title, Content)
		if err != nil {
			db.HandleError(w, http.StatusInternalServerError, "Failed to create post")
			return
		}

		postID, err := result.LastInsertId()
		if err != nil {
			db.HandleError(w, http.StatusInternalServerError, "Failed to get post id")
			return
		}

		//handling categories
		for _, catID := range selectedCategories {
			_, err = tx.Exec("INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)",
				postID, catID)
			if err != nil {
				db.HandleError(w, http.StatusInternalServerError, "Failed to associate categories")
				return
			}
		}

		if err = tx.Commit(); err != nil {
			db.HandleError(w, http.StatusInternalServerError, "Failed to complete post creation")
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		db.HandleError(w, http.StatusMethodNotAllowed, "Invalid method")
		return
	}
}

//like and dislike handlers
func LikePostHandler(w http.ResponseWriter, r *http.Request) {
	userData := r.Context().Value(auth.UserKey).(auth.ContextUser)

	postID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	//start transaction
	tx, err := db.DB.Begin()
	if err != nil {
		db.HandleError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer tx.Rollback()

	//checking if there's already a reaction from user on post.
	var currentReaction bool
	err = tx.QueryRow(
		"SELECT liked FROM post_reactions WHERE post_id = ? AND user_id = ?",
		postID, userData.UserID,
	).Scan(&currentReaction)

	if err != nil && err != sql.ErrNoRows {
		db.HandleError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if err == sql.ErrNoRows {
		//if no existing reaction: insert a new like.
		_, err = tx.Exec(
			"INSERT INTO post_reactions (post_id, user_id, liked) VALUES (?, ?, ?)",
			postID, userData.UserID, true)
		if err != nil {
			db.HandleError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	} else {
		if currentReaction {
			//if there's already a like, delete it.
			_, err = tx.Exec("DELETE FROM post_reactions WHERE post_id = ? AND user_id = ?", postID, userData.UserID)
			if err != nil {
				db.HandleError(w, http.StatusInternalServerError, "Internal server error")
				return
			}
		} else {
			//if current reaction is dislike, change to like.
			_, err = tx.Exec("UPDATE post_reactions SET liked = ? WHERE post_id = ? AND user_id = ?", true, postID, userData.UserID)
			if err != nil {
				db.HandleError(w, http.StatusInternalServerError, "Internal server error")
				return
			}
		}
	}

	if err = tx.Commit(); err != nil {
		db.HandleError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func DislikePostHandler(w http.ResponseWriter, r *http.Request) {
	userData := r.Context().Value(auth.UserKey).(auth.ContextUser)

	postID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	//begin transaction
	tx, err := db.DB.Begin()
	if err != nil {
		db.HandleError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer tx.Rollback()

	//check if there's already a reaction from user on post.
	var currentReaction bool
	err = tx.QueryRow("SELECT liked FROM post_reactions WHERE post_id = ? AND user_id = ?", postID, userData.UserID).Scan(&currentReaction)
	if err != nil && err != sql.ErrNoRows {
		db.HandleError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if err == sql.ErrNoRows {
		//if no existing reaction add a new dislike.
		_, err = tx.Exec("INSERT INTO post_reactions (post_id, user_id, liked) VALUES (?, ?, ?)", postID, userData.UserID, false)
		if err != nil {
			db.HandleError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	} else {
		if !currentReaction {
			//if current reaction is dislike delete it.
			_, err = tx.Exec("DELETE FROM post_reactions WHERE post_id = ? AND user_id = ?", postID, userData.UserID)
			if err != nil {
				db.HandleError(w, http.StatusInternalServerError, "Internal server error")
				return
			}
		} else {
			//if current reaction is like, change it to a dislike.
			_, err = tx.Exec("UPDATE post_reactions SET liked = ? WHERE post_id = ? AND user_id = ?", false, postID, userData.UserID)
			if err != nil {
				db.HandleError(w, http.StatusInternalServerError, "Internal server error")
				return
			}
		}
	}

	if err = tx.Commit(); err != nil {
		db.HandleError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
