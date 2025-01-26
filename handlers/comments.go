package handlers

import (
	"net/http"
	"strconv"
)

func (h *Handler) AddComment(w http.ResponseWriter, r *http.Request) {
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

	postID, err := strconv.ParseInt(r.FormValue("post_id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	if content == "" {
		http.Error(w, "Comment content is required", http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec(`
        INSERT INTO comments (post_id, user_id, content)
        VALUES (?, ?, ?)
    `, postID, user.ID, content)

	if err != nil {
		http.Error(w, "Error saving comment", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/post/"+strconv.FormatInt(postID, 10), http.StatusSeeOther)
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

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	commentID, err := strconv.ParseInt(r.FormValue("comment_id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid comment ID", http.StatusBadRequest)
		return
	}

	reactionType := r.FormValue("type")
	if reactionType != "like" && reactionType != "dislike" {
		http.Error(w, "Invalid reaction type", http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec(`
        INSERT INTO reactions (user_id, comment_id, type)
        VALUES (?, ?, ?)
        ON CONFLICT(user_id, comment_id) DO UPDATE SET type = ?
    `, user.ID, commentID, reactionType, reactionType)

	if err != nil {
		http.Error(w, "Error saving reaction", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
