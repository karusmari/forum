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

//this handles the reactions for the posts (likes and dislikes)
func (h *Handler) PostReaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.ErrorHandler(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := h.GetSessionUser(r)
	if user == nil {
		h.ErrorHandler(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	//reading the request(body) and decoding into the ReactionRequest struct
	var req ReactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.ErrorHandler(w, "Invalid request", http.StatusBadRequest)
		return
	}

	//starting a transaction with the database
	tx, err := h.db.Begin()
	if err != nil {
		h.ErrorHandler(w, "Database error", http.StatusInternalServerError)
		return
	}
	//if something goes wrong, rollback the transaction
	defer tx.Rollback()

	//checking if the user has already reacted to the post
	var existingType string
	err = tx.QueryRow(`
		SELECT type FROM reactions 
		WHERE user_id = ? AND post_id = ?`,
		user.ID, req.PostID,
	).Scan(&existingType)

	//if there is no reaction, then we will add a new one
	if err == sql.ErrNoRows {
		//adding a new reaction
		_, err = tx.Exec(`
			INSERT INTO reactions (user_id, post_id, type)
			VALUES (?, ?, ?)`,
			user.ID, req.PostID, req.Type,
		)
	} else if err == nil {
		if existingType == req.Type {
			//if the user has already reacted with the same type, then we will delete the reaction
			_, err = tx.Exec(`
				DELETE FROM reactions 
				WHERE user_id = ? AND post_id = ?`,
				user.ID, req.PostID,
			)
		} else {
			//if the user has reacted with a different type, then we will update the reaction
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

	//getting the updated reaction counts
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
		h.ErrorHandler(w, "Database error", http.StatusInternalServerError)
		return
	}

	//if everything is successful, commit the transaction and send the response
	if err = tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		h.ErrorHandler(w, "Database error", http.StatusInternalServerError)
		return
	}

	//creating a json response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ReactionResponse{
		Success:  true,
		Likes:    likes,
		Dislikes: dislikes,
	})
}

//this checks if the user has already reacted to the post with the same type
func (h *Handler) hasUserReaction(userID int64, postID int64, reactionType string) bool {
	var exists bool
	//checking if the db has a reaction with the same user, post and type
	err := h.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM reactions 
			WHERE user_id = ? AND post_id = ? AND type = ?
		)
	`, userID, postID, reactionType).Scan(&exists)

	if err != nil {
		return false
	}
	return exists //returning the result: either a reaction exists or not
}

//this handles the reactions for the comments (likes and dislikes)
func (h *Handler) HandleCommentReaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.ErrorHandler(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//func to get the user from the session(are they authorized to react)
	user := h.GetSessionUser(r)
	if user == nil {
		h.ErrorHandler(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	//reading the request(body) and decoding into the ReactionRequest struct
	var req ReactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		h.ErrorHandler(w, "Invalid request", http.StatusBadRequest)
		return
	}

	//starting a transaction with the database
	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		h.ErrorHandler(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	//checking from the db if the user has already reacted to the comment
	var existingType string
	err = tx.QueryRow(`
		SELECT type FROM reactions 
		WHERE user_id = ? AND comment_id = ?`,
		user.ID, req.CommentID,
	).Scan(&existingType)

	if err == sql.ErrNoRows {
		//if there is no reaction, then we will add a new one
		_, err = tx.Exec(`
			INSERT INTO reactions (user_id, comment_id, type)
			VALUES (?, ?, ?)`,
			user.ID, req.CommentID, req.Type,
		)
	} else if err == nil {
		if existingType == req.Type {
			//if the user has already reacted with the same type, then we will delete the reaction
			_, err = tx.Exec(`
				DELETE FROM reactions 
				WHERE user_id = ? AND comment_id = ?`,
				user.ID, req.CommentID,
			)
		} else {
			//if the user has reacted with a different type, then we will update the reaction
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
		h.ErrorHandler(w, "Database error", http.StatusInternalServerError)
		return
	}

	//counting the updated reaction counts
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

	//if everything went fine, commit the transaction and send the response
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		h.ErrorHandler(w, "Database error", http.StatusInternalServerError)
		return
	}

	//creating a json response which includes the updated reaction counts
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ReactionResponse{
		Success:  true,
		Likes:    likes,
		Dislikes: dislikes,
	})
}
