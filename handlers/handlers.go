package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

type Handler struct {
	db        *sql.DB
	templates *template.Template
}

// this will create a new handler which contains the database and the templates
func NewHandler(db *sql.DB) *Handler {
	//creating a new template with the timezone function
	funcMap := template.FuncMap{
		"timezone": func(name string) *time.Location {
			//loads the timezone by name
			loc, err := time.LoadLocation(name)
			if err != nil {
				//if doesn't exist, then we will use UTC
				return time.UTC
			}
			return loc //returns the timezone
		},
	}

	//parsing the templates from the templates folder and adding the function map
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))
	//returning the handler with the database and the templates
	return &Handler{
		db:        db,
		templates: tmpl,
	}
}

// getting the categories from the database
func (h *Handler) getCategories() ([]Category, error) {
	//this will query the database to get the categories
	rows, err := h.db.Query(`
		SELECT c.id, c.name, c.description, 
		COUNT(pc.post_id) as post_count 
		FROM categories c
		LEFT JOIN post_categories pc ON c.id = pc.category_id
		GROUP BY c.id, c.name, c.description
		ORDER BY c.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	//iterating the results and adding them to the categories slice
	var categories []Category
	for rows.Next() {
		var cat Category
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Description, &cat.PostCount); err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}
	return categories, nil
}

//getting the posts from the database with the post ID
func (h *Handler) getPostCategories(postID int64) ([]string, error) {
	rows, err := h.db.Query(`
		SELECT c.name
		FROM categories c
		JOIN post_categories pc ON c.id = pc.category_id
		WHERE pc.post_id = ?
	`, postID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		categories = append(categories, name)
	}
	return categories, nil
}

//displaying the rules page
func (h *Handler) Rules(w http.ResponseWriter, r *http.Request) {
	data := &TemplateData{
		Title: "Forum Rules",
		User:  h.GetSessionUser(w, r),
	}
	h.templates.ExecuteTemplate(w, "rules.html", data)
}

//handling the error messages
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
