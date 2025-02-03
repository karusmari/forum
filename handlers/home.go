package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
)

// При получении постов для главной страницы
func getPosts(db *sql.DB) []Post {
	query := `
		SELECT p.id, p.title, p.content, p.username, p.created_at, p.user_id,
			   p.likes, p.dislikes, GROUP_CONCAT(c.name) as categories
		FROM posts p
		LEFT JOIN post_categories pc ON p.id = pc.post_id
		LEFT JOIN categories c ON pc.category_id = c.id
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Query error: %v", err)
		return nil
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		var categories sql.NullString
		err := rows.Scan(
			&p.ID, &p.Title, &p.Content, &p.Username, &p.CreatedAt, &p.UserID,
			&p.Likes, &p.Dislikes, &categories,
		)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		if categories.Valid {
			p.Categories = strings.Split(categories.String, ",")
		}
		posts = append(posts, p)
	}
	return posts
}

func (h *Handler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	posts := getPosts(h.db)
	data := TemplateData{
		Posts: posts,
	}
	h.templates.ExecuteTemplate(w, "index.html", data)
}

func (h *Handler) CategoryHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем ID категории из URL и добавляем логирование
	categoryID := r.URL.Path[len("/category/"):]
	log.Printf("Accessing category with ID: %s", categoryID)

	// Проверяем, что ID не пустой
	if categoryID == "" {
		log.Printf("Empty category ID")
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}

	// Получаем информацию о категории
	var category Category
	err := h.db.QueryRow("SELECT id, name, description FROM categories WHERE id = ?", categoryID).
		Scan(&category.ID, &category.Name, &category.Description)
	if err != nil {
		log.Printf("Error getting category: %v", err)
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}
	log.Printf("Found category: %s", category.Name)

	// Получаем посты категории
	query := `
		SELECT p.id, p.title, p.content, p.username, p.created_at, p.user_id,
			   p.likes, p.dislikes, c.name as category_name
		FROM posts p
		INNER JOIN post_categories pc ON p.id = pc.post_id
		INNER JOIN categories c ON pc.category_id = c.id
		WHERE c.id = ?
		ORDER BY p.created_at DESC
	`

	rows, err := h.db.Query(query, categoryID)
	if err != nil {
		log.Printf("Error getting posts: %v", err)
		http.Error(w, "Error loading posts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		var categoryName string
		err := rows.Scan(
			&p.ID, &p.Title, &p.Content, &p.Username, &p.CreatedAt, &p.UserID,
			&p.Likes, &p.Dislikes, &categoryName,
		)
		if err != nil {
			log.Printf("Error scanning post: %v", err)
			continue
		}
		p.Categories = []string{categoryName}
		posts = append(posts, p)
	}
	log.Printf("Found %d posts in category", len(posts))

	data := TemplateData{
		Posts:    posts,
		Category: &category,
	}

	if err := h.templates.ExecuteTemplate(w, "category.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}


