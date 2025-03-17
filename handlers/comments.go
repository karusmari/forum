package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (h *Handler) AddComment(w http.ResponseWriter, r *http.Request) {
	// Check method
	if r.Method != http.MethodPost {
		h.ErrorHandler(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication
	user := h.GetSessionUser(w, r)
	if user == nil {
		h.ErrorHandler(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		h.ErrorHandler(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get and validate data
	postID := r.FormValue("post_id")
	content := strings.TrimSpace(r.FormValue("content"))

	// Check if comment is not empty
	if content == "" {
		h.ErrorHandler(w, "Comment cannot be empty", http.StatusBadRequest)
		return
	}

	// Check if postID is a valid number
	pid, err := strconv.ParseInt(postID, 10, 64)
	if err != nil {
		h.ErrorHandler(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	// Check if post exists in the db
	var exists bool
	err = h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM posts WHERE id = ?)", pid).Scan(&exists)
	if err != nil {
		log.Printf("Database error: %v", err)
		h.ErrorHandler(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}
	if !exists {
		h.ErrorHandler(w, "Post not found", http.StatusNotFound)
		return
	}

	// Create comment with correct timestamp
	now := time.Now().In(h.location)

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("Database error: %v", err)
		h.ErrorHandler(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	//inserting the comment into the database
	result, err := tx.Exec(`
		INSERT INTO comments (post_id, user_id, content, username, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, pid, user.ID, content, user.Username, now)

	if err != nil {
		log.Printf("Error creating comment: %v", err)
		h.ErrorHandler(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}

	//committing the transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Database error: %v", err)
		h.ErrorHandler(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}

	//getting the ID of the comment
	commentID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error creating comment: %v", err)
		h.ErrorHandler(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}

	//checking if the comment was successfully added into the database
	var count int
	err = h.db.QueryRow("SELECT COUNT(*) FROM comments WHERE id = ?", commentID).Scan(&count)
	if err != nil {
		log.Printf("Database error: %v", err)
		h.ErrorHandler(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}

	if count == 0 {
		log.Printf("Failed to create comment: %v", err)
		h.ErrorHandler(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}

	//redirecting the user back to the post page
	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
}

// Add a new method to get comments
func (h *Handler) getComments(postID int64) ([]*Comment, error) {
	rows, err := h.db.Query(`
		SELECT c.id, c.user_id, c.content, c.created_at, c.username,
		COUNT(CASE WHEN r.type = 'like' THEN 1 END) as likes,
		COUNT(CASE WHEN r.type = 'dislike' THEN 1 END) as dislikes
		FROM comments c
		LEFT JOIN reactions r ON c.id = r.comment_id
		WHERE c.post_id = ?
		GROUP BY c.id, c.user_id, c.content, c.created_at, c.username
		ORDER BY c.created_at DESC
	`, postID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	//collecting all the comments into a slice
	var comments []*Comment
	for rows.Next() {
		var c Comment
		err := rows.Scan(
			&c.ID, &c.UserID, &c.Content, &c.CreatedAt, &c.Username,
			&c.Likes, &c.Dislikes,
		)
		if err != nil {
			return nil, err
		}
		comments = append(comments, &c)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}
