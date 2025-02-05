package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

type ReactionRequest struct {
	PostID    int64  `json:"post_id"`
	CommentID int64  `json:"comment_id"`
	Type      string `json:"type"`
}

type ReactionResponse struct {
	Success  bool `json:"success"`
	Likes    int  `json:"likes"`
	Dislikes int  `json:"dislikes"`
}

func (h *Handler) HandleReaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := h.GetSessionUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req ReactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Начинаем транзакцию
	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Проверяем существующую реакцию
	var existingType string
	err = tx.QueryRow(`
		SELECT type FROM reactions 
		WHERE user_id = ? AND post_id = ?`,
		user.ID, req.PostID,
	).Scan(&existingType)

	if err == sql.ErrNoRows {
		// Если реакции нет, добавляем новую
		_, err = tx.Exec(`
			INSERT INTO reactions (user_id, post_id, type)
			VALUES (?, ?, ?)`,
			user.ID, req.PostID, req.Type,
		)
	} else if err == nil {
		if existingType == req.Type {
			// Если такая же реакция уже есть, удаляем её
			_, err = tx.Exec(`
				DELETE FROM reactions 
				WHERE user_id = ? AND post_id = ?`,
				user.ID, req.PostID,
			)
		} else {
			// Если есть другая реакция, обновляем тип
			_, err = tx.Exec(`
				UPDATE reactions 
				SET type = ? 
				WHERE user_id = ? AND post_id = ?`,
				req.Type, user.ID, req.PostID,
			)
		}
	}

	if err != nil {
		log.Printf("Error handling reaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Получаем обновленные счетчики
	var likes, dislikes int
	err = tx.QueryRow(`
		SELECT 
			COUNT(CASE WHEN type = 'like' THEN 1 END) as likes,
			COUNT(CASE WHEN type = 'dislike' THEN 1 END) as dislikes
		FROM reactions 
		WHERE post_id = ?`,
		req.PostID,
	).Scan(&likes, &dislikes)

	if err != nil {
		log.Printf("Error getting reaction counts: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if err = tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ReactionResponse{
		Success:  true,
		Likes:    likes,
		Dislikes: dislikes,
	})
}

func (h *Handler) hasUserReaction(userID int64, postID int64, reactionType string) bool {
	var exists bool
	err := h.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM reactions 
			WHERE user_id = ? AND post_id = ? AND type = ?
		)
	`, userID, postID, reactionType).Scan(&exists)

	if err != nil {
		return false
	}
	return exists
}

// Добавляем обработчик реакций на комментарии
func (h *Handler) HandleCommentReaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := h.GetSessionUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req ReactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Проверяем существующую реакцию
	var existingType string
	err = tx.QueryRow(`
		SELECT type FROM reactions 
		WHERE user_id = ? AND comment_id = ?`,
		user.ID, req.CommentID,
	).Scan(&existingType)

	if err == sql.ErrNoRows {
		// Если реакции нет, добавляем новую
		_, err = tx.Exec(`
			INSERT INTO reactions (user_id, comment_id, type)
			VALUES (?, ?, ?)`,
			user.ID, req.CommentID, req.Type,
		)
	} else if err == nil {
		if existingType == req.Type {
			// Если такая же реакция уже есть, удаляем её
			_, err = tx.Exec(`
				DELETE FROM reactions 
				WHERE user_id = ? AND comment_id = ?`,
				user.ID, req.CommentID,
			)
		} else {
			// Если есть другая реакция, обновляем тип
			_, err = tx.Exec(`
				UPDATE reactions 
				SET type = ? 
				WHERE user_id = ? AND comment_id = ?`,
				req.Type, user.ID, req.CommentID,
			)
		}
	}

	if err != nil {
		log.Printf("Error handling reaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Получаем обновленные счетчики
	var likes, dislikes int
	err = tx.QueryRow(`
		SELECT 
			COUNT(CASE WHEN type = 'like' THEN 1 END) as likes,
			COUNT(CASE WHEN type = 'dislike' THEN 1 END) as dislikes
		FROM reactions 
		WHERE comment_id = ?`,
		req.CommentID,
	).Scan(&likes, &dislikes)

	if err != nil {
		log.Printf("Error getting reaction counts: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ReactionResponse{
		Success:  true,
		Likes:    likes,
		Dislikes: dislikes,
	})
}
