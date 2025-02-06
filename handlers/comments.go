package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var location *time.Location

// this will initialize the location for a timezone as soon as the package is loaded
func init() {
	var err error
	location, err = time.LoadLocation("Europe/Helsinki") // UTC+2
	if err != nil {
		log.Printf("Error loading timezone: %v", err)
		location = time.UTC
	}
}

func (h *Handler) AddComment(w http.ResponseWriter, r *http.Request) {
	// Check method
	if r.Method != http.MethodPost {
		log.Printf("Invalid method: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication
	user := h.GetSessionUser(r)
	if user == nil {
		log.Printf("User not authenticated")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		log.Printf("Form parse error: %v", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get and validate data
	postID := r.FormValue("post_id")
	content := strings.TrimSpace(r.FormValue("content"))

	// Check if comment is not empty
	if content == "" {
		log.Printf("Empty comment content")
		http.Error(w, "Comment cannot be empty", http.StatusBadRequest)
		return
	}

	// Check if postID is a valid number
	pid, err := strconv.ParseInt(postID, 10, 64)
	if err != nil {
		log.Printf("Invalid post ID: %v", err)
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	// Check if post exists
	var exists bool
	err = h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM posts WHERE id = ?)", pid).Scan(&exists)
	if err != nil {
		log.Printf("Error checking post existence: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if !exists {
		log.Printf("Post %d does not exist", pid)
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	// Create comment with correct timestamp
	now := time.Now().In(location)
	log.Printf("Adding comment: postID=%s, userID=%d, content=%s, username=%s, time=%v",
		postID, user.ID, content, user.Username, now)

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		INSERT INTO comments (post_id, user_id, content, username, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, pid, user.ID, content, user.Username, now)

	if err != nil {
		log.Printf("Error creating comment: %v", err)
		http.Error(w, "Error creating comment", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	commentID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error getting comment ID: %v", err)
	} else {
		log.Printf("Created comment %d for post %d", commentID, pid)
	}

	// Проверяем, что комментарий действительно сохранился
	var count int
	err = h.db.QueryRow("SELECT COUNT(*) FROM comments WHERE id = ?", commentID).Scan(&count)
	if err != nil {
		log.Printf("Error verifying comment: %v", err)
	} else {
		log.Printf("Comment verification: found %d comments with ID %d", count, commentID)
	}

	// Перенаправляем обратно на страницу поста
	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
}

func (h *Handler) ReactToComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := h.GetSessionUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	commentID := r.FormValue("comment_id")
	reactionType := r.FormValue("type")

	if reactionType != "like" && reactionType != "dislike" {
		http.Error(w, "Invalid reaction type", http.StatusBadRequest)
		return
	}

	// Начинаем транзакцию
	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Удаляем существующую реакцию, если есть
	_, err = tx.Exec(`
		DELETE FROM reactions 
		WHERE user_id = ? AND comment_id = ?
	`, user.ID, commentID)
	if err != nil {
		log.Printf("Error removing old reaction: %v", err)
		http.Error(w, "Error saving reaction", http.StatusInternalServerError)
		return
	}

	// Добавляем новую реакцию
	_, err = tx.Exec(`
		INSERT INTO reactions (user_id, comment_id, type)
		VALUES (?, ?, ?)
	`, user.ID, commentID, reactionType)
	if err != nil {
		log.Printf("Error adding new reaction: %v", err)
		http.Error(w, "Error saving reaction", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Возвращаем обновленное количество реакций
	var likes, dislikes int
	err = h.db.QueryRow(`
		SELECT 
			COUNT(CASE WHEN type = 'like' THEN 1 END) as likes,
			COUNT(CASE WHEN type = 'dislike' THEN 1 END) as dislikes
		FROM reactions
		WHERE comment_id = ?
	`, commentID).Scan(&likes, &dislikes)
	if err != nil {
		log.Printf("Error getting reaction counts: %v", err)
		http.Error(w, "Error getting reaction counts", http.StatusInternalServerError)
		return
	}

	// w.Header().Set("Content-Type", "application/json")
	// json.NewEncoder(w).Encode(map[string]interface{}{
	// 	"likes":    likes,
	// 	"dislikes": dislikes,
	// })
}

func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := h.GetSessionUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	commentID := r.FormValue("comment_id")
	postID := r.FormValue("post_id")

	// Проверяем, что пользователь имеет право удалить комментарий
	var userID int64
	err := h.db.QueryRow("SELECT user_id FROM comments WHERE id = ?", commentID).Scan(&userID)
	if err != nil {
		log.Printf("Error checking comment ownership: %v", err)
		http.Error(w, "Comment not found", http.StatusNotFound)
		return
	}

	// Разрешаем удаление только владельцу комментария или админу
	if userID != user.ID && !user.IsAdmin {
		http.Error(w, "Not authorized to delete this comment", http.StatusForbidden)
		return
	}

	// Удаляем комментарий
	_, err = h.db.Exec("DELETE FROM comments WHERE id = ?", commentID)
	if err != nil {
		log.Printf("Error deleting comment: %v", err)
		http.Error(w, "Error deleting comment", http.StatusInternalServerError)
		return
	}

	// Перенаправляем обратно на страницу поста
	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
}

// Добавляем новый обработчик для редактирования комментариев
func (h *Handler) EditComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := h.GetSessionUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	commentID := r.FormValue("comment_id")
	postID := r.FormValue("post_id")
	newContent := r.FormValue("content")

	// Проверяем, что пользователь имеет право редактировать комментарий
	var userID int64
	err := h.db.QueryRow("SELECT user_id FROM comments WHERE id = ?", commentID).Scan(&userID)
	if err != nil {
		log.Printf("Error checking comment ownership: %v", err)
		http.Error(w, "Comment not found", http.StatusNotFound)
		return
	}

	// Разрешаем редактирование только владельцу комментария или админу
	if userID != user.ID && !user.IsAdmin {
		http.Error(w, "Not authorized to edit this comment", http.StatusForbidden)
		return
	}

	// Обновляем комментарий
	_, err = h.db.Exec("UPDATE comments SET content = ? WHERE id = ?", newContent, commentID)
	if err != nil {
		log.Printf("Error updating comment: %v", err)
		http.Error(w, "Error updating comment", http.StatusInternalServerError)
		return
	}

	// Перенаправляем обратно на страницу поста
	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
}
