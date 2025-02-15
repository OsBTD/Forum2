package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"forum/internal/auth"

	db "forum/internal/database"
	H "forum/internal/handlers"
)

func NewRouter() *http.ServeMux {
	//initialize router
	router := http.NewServeMux()

	//public routes
	router.HandleFunc("/", H.HomeHandler)
	router.HandleFunc("/login", auth.LoginHandler)
	router.HandleFunc("/register", auth.RegisterHandler)
	router.HandleFunc("/logout", auth.LogoutHandler)

	//routes + middleware
	router.Handle("/add-post", auth.RequireAuth(http.HandlerFunc(H.AddPostHandler)))
	router.Handle("/add-comment", auth.RequireAuth(http.HandlerFunc(H.CommentHandler)))
	router.Handle("/like-post", auth.RequireAuth(http.HandlerFunc(H.LikePostHandler)))
	router.Handle("/dislike-post", auth.RequireAuth(http.HandlerFunc(H.DislikePostHandler)))
	router.Handle("/like-comment", auth.RequireAuth(http.HandlerFunc(H.LikeCommentHandler)))
	router.Handle("/dislike-comment", auth.RequireAuth(http.HandlerFunc(H.DislikeCommentHandler)))

	//static files handler plus checks for directories and ".." and forbids users from accessing 
	http.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		subPath := strings.TrimPrefix(r.URL.Path, "/static/")

		if strings.Contains(subPath, "..") || strings.Contains(subPath, "//") {
			db.HandleError(w, http.StatusForbidden, "Invalid path pattern")
			return
		}

		filePath := filepath.Join("static", subPath)

		if info, err := os.Stat(filePath); err != nil || info.IsDir() {
			db.HandleError(w, http.StatusNotFound, "File not found")
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=86400")
		http.ServeFile(w, r, filePath)
	})

	return router
}
