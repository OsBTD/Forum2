package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forum/internal/auth"

	db "forum/internal/database"
)

type Category struct {
	ID          int
	Name        string
	Description string
}

type Comment struct {
	ID        int
	PostID    int
	UserID    int
	Username  string
	Content   string
	CreatedAt time.Time
	Likes     int
	Dislikes  int
}

type Post struct {
	ID         int
	UserID     int
	Username   string
	Title      string
	Content    string
	CreatedAt  time.Time
	Likes      int
	Dislikes   int
	Categories []string
	Comments   []Comment
}

func buildPostsQuery(r *http.Request, userData auth.ContextUser) (string, []interface{}) {
	baseQuery := `
	SELECT 
	    p.id, 
	    p.title, 
	    p.content, 
	    p.created_at, 
	    u.username,
	    (SELECT COUNT(*) FROM post_reactions pr WHERE pr.post_id = p.id AND pr.liked = 1) AS likes,
	    (SELECT COUNT(*) FROM post_reactions pr WHERE pr.post_id = p.id AND pr.liked = 0) AS dislikes,
	    COALESCE(GROUP_CONCAT(DISTINCT c.name), '') AS categories
	FROM posts p
	JOIN users u ON p.user_id = u.id
	LEFT JOIN post_categories pc ON p.id = pc.post_id
	LEFT JOIN categories c ON pc.category_id = c.id
	WHERE 1=1
	`
	var args []interface{}
	q := r.URL.Query()
	if categories, exists := q["category"]; exists && len(categories) > 0 {
		baseQuery += " AND c.id IN ("
		for i, category := range categories {
			if i > 0 {
				baseQuery += ", "
			}
			baseQuery += "?"
			args = append(args, category)
		}
		baseQuery += ")"
	}

	if q.Get("created") == "1" && userData.LoggedIn {
		baseQuery += " AND p.user_id = ?"
		args = append(args, userData.UserID)
	}

	if q.Get("liked") == "1" && userData.LoggedIn {
		baseQuery += " AND p.id IN (SELECT post_id FROM post_reactions WHERE user_id = ? AND liked = 1)"
		args = append(args, userData.UserID)
	}

	// complete the query with grouping and ordering.
	baseQuery += " GROUP BY p.id ORDER BY p.created_at DESC"

	return baseQuery, args
}

func fetchPosts(query string, args []interface{}) ([]Post, []int, error) {
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var posts []Post
	var postIDs []int
	for rows.Next() {
		var post Post
		var categoriesStr string

		if err := rows.Scan(
			&post.ID,
			&post.Title,
			&post.Content,
			&post.CreatedAt,
			&post.Username,
			&post.Likes,
			&post.Dislikes,
			&categoriesStr,
		); err != nil {
			return nil, nil, err
		}
		if categoriesStr != "" {
			post.Categories = strings.Split(categoriesStr, ",")
		} else {
			post.Categories = []string{}
		}

		posts = append(posts, post)
		postIDs = append(postIDs, post.ID)

		posts = append(posts, post)
		postIDs = append(postIDs, post.ID)
	}
	if err := rows.Err(); err != nil {
		log.Println("Error scanning post:", err)
		return nil, nil, err
	}
	return posts, postIDs, nil
}

func fetchComments(postIDs []int) (map[int][]Comment, error) {
	if len(postIDs) == 0 {
		return make(map[int][]Comment), nil
	}

	// build a placeholder string (e.g., "?, ?, ?") for the SQL IN clause.
	placeholders := make([]string, len(postIDs))
	args := make([]interface{}, len(postIDs))
	for i, id := range postIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := `
		SELECT cm.id, cm.post_id, cm.user_id, u.username, cm.content, cm.created_at, 
		(SELECT COUNT(*) FROM comment_reactions cr WHERE cr.comment_id = cm.id AND cr.liked = 1) AS likes,
		(SELECT COUNT(*) FROM comment_reactions cr WHERE cr.comment_id = cm.id AND cr.liked = 0) AS dislikes
		FROM comments cm
		JOIN users u ON cm.user_id = u.id
		WHERE cm.post_id IN (` + strings.Join(placeholders, ",") + `)
		ORDER BY cm.created_at DESC
		`
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	commentsMap := make(map[int][]Comment)
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Username, &c.Content, &c.CreatedAt, &c.Likes, &c.Dislikes); err != nil {
			// log error and continue with other comments.
			fmt.Println("Error at rows scan comment", err)
			continue
		}
		commentsMap[c.PostID] = append(commentsMap[c.PostID], c)
		// log.Printf("Comment ID %d: Likes = %d, Dislikes = %d", c.ID, c.Likes, c.Dislikes)

	}

	return commentsMap, nil
}

func getCategories() ([]Category, error) {
	rows, err := db.DB.Query("SELECT id, name, description FROM categories")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Description); err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}
	return categories, nil
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		db.HandleError(w, http.StatusNotFound, "Page not found")
		return
	}
	if r.Method != http.MethodGet {
		db.HandleError(w, http.StatusMethodNotAllowed, "Invalid method")
		return

	}

	userData, _ := r.Context().Value(auth.UserKey).(auth.ContextUser)

	query, args := buildPostsQuery(r, userData)

	posts, postIDs, err := fetchPosts(query, args)
	if err != nil {
		db.HandleError(w, http.StatusInternalServerError, "Error loading posts")
		return
	}
	categories, err := getCategories()
	if err != nil {
		db.HandleError(w, http.StatusInternalServerError, "Error loading categories")
		return
	}
	commentsMap, err := fetchComments(postIDs)
	if err != nil {
		log.Fatal("Error fetching comments ", err)
		db.HandleError(w, http.StatusInternalServerError, "Error loading comments")
		return
	}
	// attach loaded comments to the corresponding posts.
	for i, post := range posts {
		posts[i].Comments = commentsMap[post.ID]
	}

	selectedCategoryIDs := r.URL.Query()["category"] // Query returns a slice of values
	var selectedCategories []int
	for _, catID := range selectedCategoryIDs {
		if id, err := strconv.Atoi(catID); err == nil {
			selectedCategories = append(selectedCategories, id)
		}
	}

	// prepare data for the template.
	data := map[string]interface{}{
		"Title":              "Home Page",
		"LoggedIn":           userData.LoggedIn,
		"Username":           userData.Username,
		"Posts":              posts,
		"FilterCategories":   categories,
		"SelectedCategories": selectedCategories,
		"FilterCreated":      r.URL.Query().Get("created") == "1",
		"FilterLiked":        r.URL.Query().Get("liked") == "1",
	}

	// render the home page template with the collected data.
	db.RenderTemplate(w, "home", data)
}
