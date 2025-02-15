package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"forum/internal/auth"
	db "forum/internal/database"
)

//commentHandler handles displaying the comment form and processing new comments.
func CommentHandler(w http.ResponseWriter, r *http.Request) {
	userData := r.Context().Value(auth.UserKey).(auth.ContextUser)

	if r.Method == http.MethodGet {
		postID, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			db.HandleError(w, http.StatusBadRequest, "Invalid post id")
			return
		}

		db.RenderTemplate(w, "add_comment", map[string]interface{}{
			"Title":    "Add Comment",
			"PostID":   postID,
			"LoggedIn": userData.LoggedIn,
			"Username": userData.Username,
		})
		return
	}
	if r.Method == http.MethodPost {
		postIDStr := r.FormValue("post_id")
		postID, err := strconv.Atoi(postIDStr)
		if err != nil {
			db.HandleError(w, http.StatusBadRequest, "Invalid post id")
			return
		}

		content := r.FormValue("content")
		if strings.TrimSpace(content) == "" {
			w.WriteHeader(http.StatusBadRequest)
			db.RenderTemplate(w, "add_comment", map[string]interface{}{
				"Title":    "Add Comment",
				"PostID":   postID,
				"LoggedIn": userData.LoggedIn,
				"Username": userData.Username,
				"Error":    "Content cannot be empty",
			})
			return
		}

		_, err = db.DB.Exec("INSERT INTO comments (post_id, user_id, content) VALUES (?, ?, ?)", postID, userData.UserID, content)
		if err != nil {
			db.HandleError(w, http.StatusInternalServerError, "Failed to add comment")
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		db.HandleError(w, http.StatusMethodNotAllowed, "Invalid method")
		return
	}
}

func LikeCommentHandler(w http.ResponseWriter, r *http.Request) {
	userData := r.Context().Value(auth.UserKey).(auth.ContextUser)

	//extract and validate the comment id from the query parameters
	commentID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		db.HandleError(w, http.StatusBadRequest, "invalid comment id")
		return
	}

	//begin a transaction
	tx, err := db.DB.Begin()
	if err != nil {
		db.HandleError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer tx.Rollback()

	//check if a reaction already exists for this comment and user
	var currentReaction bool
	err = tx.QueryRow(
		"SELECT liked FROM comment_reactions WHERE comment_id = ? AND user_id = ?",
		commentID, userData.UserID,
	).Scan(&currentReaction)

	if err != nil && err != sql.ErrNoRows {
		db.HandleError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if err == sql.ErrNoRows {
		//if no reaction add a new like
		_, err = tx.Exec(
			"INSERT INTO comment_reactions (comment_id, user_id, liked) VALUES (?, ?, ?)",
			commentID, userData.UserID, true,
		)
		if err != nil {
			db.HandleError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	} else {
		if currentReaction {
			//if reaction already exists delete it
			_, err = tx.Exec(
				"DELETE FROM comment_reactions WHERE comment_id = ? AND user_id = ?",
				commentID, userData.UserID,
			)
			if err != nil {
				db.HandleError(w, http.StatusInternalServerError, "Internal server error")
				return
			}
		} else {
			//if dislike exists change it to like
			_, err = tx.Exec(
				"UPDATE comment_reactions SET liked = ? WHERE comment_id = ? AND user_id = ?",
				true, commentID, userData.UserID,
			)
			if err != nil {
				db.HandleError(w, http.StatusInternalServerError, "Internal server error")
				return
			}
		}
	}

	//commit the transaction
	if err = tx.Commit(); err != nil {
		db.HandleError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func DislikeCommentHandler(w http.ResponseWriter, r *http.Request) {
	//rtrieve user data from context
	userData, ok := r.Context().Value(auth.UserKey).(auth.ContextUser)
	if !ok {
		db.HandleError(w, http.StatusUnauthorized, "Unauthorized action")
		return
	}

	//extract and validate the comment id
	commentID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		db.HandleError(w, http.StatusBadRequest, "Invalid comment id")
		return
	}

	//begin transaction
	tx, err := db.DB.Begin()
	if err != nil {
		db.HandleError(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	defer tx.Rollback()

	//check if a reaction exists
	var currentReaction bool
	err = tx.QueryRow(
		"SELECT liked FROM comment_reactions WHERE comment_id = ? AND user_id = ?",
		commentID, userData.UserID,
	).Scan(&currentReaction)

	if err != nil && err != sql.ErrNoRows {
		db.HandleError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if err == sql.ErrNoRows {
		//no reaction insert a new dislike
		_, err = tx.Exec(
			"INSERT INTO comment_reactions (comment_id, user_id, liked) VALUES (?, ?, ?)",
			commentID, userData.UserID, false,
		)
		if err != nil {
			db.HandleError(w, http.StatusInternalServerError, "Internal server error")
			return
		}
	} else {
		if !currentReaction {
			//already disliked delete
			_, err = tx.Exec(
				"DELETE FROM comment_reactions WHERE comment_id = ? AND user_id = ?",
				commentID, userData.UserID,
			)
			if err != nil {
				db.HandleError(w, http.StatusInternalServerError, "Internal server error")
				return
			}
		} else {
			//like exists change to dislike
			_, err = tx.Exec(
				"UPDATE comment_reactions SET liked = ? WHERE comment_id = ? AND user_id = ?",
				false, commentID, userData.UserID,
			)
			if err != nil {
				db.HandleError(w, http.StatusInternalServerError, "Internal server error")
				return
			}
		}
	}

	//commit
	if err = tx.Commit(); err != nil {
		db.HandleError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	//redirect after processing
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
