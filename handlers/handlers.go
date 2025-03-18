package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

type Handler struct {
	db        *sql.DB
	templates *template.Template
	location  *time.Location
}

// this will create a new handler which contains the database and the templates
func NewHandler(db *sql.DB, templates *template.Template) *Handler {
	location, err := time.LoadLocation("Europe/Helsinki") // UTC+2
	if err != nil {
		log.Printf("Error loading location: %v", err)
		location = time.UTC
	}

	return &Handler{
		db:        db,
		templates: templates,
		location:  location,
	}
}

// displaying the rules page
func (h *Handler) Rules(w http.ResponseWriter, r *http.Request) {
	data := &TemplateData{
		Title: "Forum Rules",
		User:  h.GetSessionUser(w, r),
	}
	h.templates.ExecuteTemplate(w, "rules.html", data)
}

// handling the error messages
func (h *Handler) ErrorHandler(w http.ResponseWriter, errorMessage string, statusCode int) {
    w.WriteHeader(statusCode)

    data := ErrorData{
        ErrorMessage: errorMessage,
        ErrorCode:    fmt.Sprintf("%d", statusCode),
    }

    err := h.templates.ExecuteTemplate(w, "error.html", data)
    if err != nil {
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
    }
}
