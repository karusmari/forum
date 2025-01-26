package handlers

import (
	"net/http"
)

func (h *Handler) renderTemplate(w http.ResponseWriter, name string, data *TemplateData) {
	err := h.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		http.Error(w, "Template rendering error", http.StatusInternalServerError)
	}
}
