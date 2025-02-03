package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

func (h *Handler) DeletePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := h.GetSessionUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	postID := r.FormValue("post_id")

	// Проверяем, что пользователь имеет право удалить пост
	var userID int64
	err := h.db.QueryRow("SELECT user_id FROM posts WHERE id = ?", postID).Scan(&userID)
	if err != nil {
		log.Printf("Error checking post ownership: %v", err)
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	// Разрешаем удаление только владельцу поста или админу
	if userID != user.ID && !user.IsAdmin {
		http.Error(w, "Not authorized to delete this post", http.StatusForbidden)
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

	// Удаляем связанные комментарии
	_, err = tx.Exec("DELETE FROM comments WHERE post_id = ?", postID)
	if err != nil {
		log.Printf("Error deleting comments: %v", err)
		http.Error(w, "Error deleting post", http.StatusInternalServerError)
		return
	}

	// Удаляем связи с категориями
	_, err = tx.Exec("DELETE FROM post_categories WHERE post_id = ?", postID)
	if err != nil {
		log.Printf("Error deleting category links: %v", err)
		http.Error(w, "Error deleting post", http.StatusInternalServerError)
		return
	}

	// Удаляем реакции
	_, err = tx.Exec("DELETE FROM reactions WHERE post_id = ?", postID)
	if err != nil {
		log.Printf("Error deleting reactions: %v", err)
		http.Error(w, "Error deleting post", http.StatusInternalServerError)
		return
	}

	// Удаляем сам пост
	_, err = tx.Exec("DELETE FROM posts WHERE id = ?", postID)
	if err != nil {
		log.Printf("Error deleting post: %v", err)
		http.Error(w, "Error deleting post", http.StatusInternalServerError)
		return
	}

	// Завершаем транзакцию
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Error deleting post", http.StatusInternalServerError)
		return
	}

	// Перенаправляем на главную страницу
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) ReactToPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := h.GetSessionUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	postID := r.FormValue("post_id")
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
		WHERE user_id = ? AND post_id = ?
	`, user.ID, postID)
	if err != nil {
		log.Printf("Error removing old reaction: %v", err)
		http.Error(w, "Error saving reaction", http.StatusInternalServerError)
		return
	}

	// Добавляем новую реакцию
	_, err = tx.Exec(`
		INSERT INTO reactions (user_id, post_id, type)
		VALUES (?, ?, ?)
	`, user.ID, postID, reactionType)
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
		WHERE post_id = ?
	`, postID).Scan(&likes, &dislikes)
	if err != nil {
		log.Printf("Error getting reaction counts: %v", err)
		http.Error(w, "Error getting reaction counts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"likes":    likes,
		"dislikes": dislikes,
	})
} 