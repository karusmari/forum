package handlers

import (
	"encoding/json"
	"net/http"
)

type ReactionRequest struct {
	PostID int64  `json:"post_id"`
	Type   string `json:"type"`
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

	// Validate reaction type
	if req.Type != "like" && req.Type != "dislike" {
		http.Error(w, "Invalid reaction type", http.StatusBadRequest)
		return
	}

	_, err := h.db.Exec(`
		INSERT INTO reactions (user_id, post_id, type)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id, post_id) DO UPDATE SET type = ?
	`, user.ID, req.PostID, req.Type, req.Type)

	if err != nil {
		http.Error(w, "Error saving reaction", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
