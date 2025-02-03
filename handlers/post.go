package handlers

import (
	"net/http"
)

// Удаляем дублирующиеся функции, так как они уже определены в handlers.go
// Удаляем CreatePost и getCategories

func (h *Handler) CreatePostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		categories, err := h.getCategories()
		if err != nil {
			http.Error(w, "Error loading categories", http.StatusInternalServerError)
			return
		}
		data := TemplateData{
			Categories: categories,
		}
		h.templates.ExecuteTemplate(w, "new_post.html", data)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Перенаправляем на существующий обработчик CreatePost
	h.CreatePost(w, r)
}
