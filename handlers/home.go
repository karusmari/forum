package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// При получении постов для главной страницы
func getPosts(db *sql.DB) []Post {
	query := `
		SELECT p.id, p.title, p.content, p.username, p.created_at, p.user_id,
			   p.likes, p.dislikes, GROUP_CONCAT(DISTINCT c.name) as categories,
			   COUNT(DISTINCT cm.id) as comment_count
		FROM posts p
		LEFT JOIN post_categories pc ON p.id = pc.post_id
		LEFT JOIN categories c ON pc.category_id = c.id
		LEFT JOIN comments cm ON p.id = cm.post_id
		GROUP BY p.id, p.title, p.content, p.username, p.created_at, p.user_id,
				 p.likes, p.dislikes
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
			&p.Likes, &p.Dislikes, &categories, &p.CommentCount,
		)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		if categories.Valid {
			p.Categories = strings.Split(categories.String, ",")
		}
		log.Printf("Post: ID=%d, Title=%s, Comments=%d, Categories=%v",
			p.ID, p.Title, p.CommentCount, p.Categories)
		posts = append(posts, p)
	}
	return posts
}

func (h *Handler) CategoryHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем ID категории из URL и добавляем логирование
	categoryIDStr := r.URL.Path[len("/category/"):]
	log.Printf("Accessing category with ID: %s", categoryIDStr)

	// Проверяем, что ID не пустой
	if categoryIDStr == "" {
		log.Printf("Empty category ID")
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}

	// Конвертируем строку в int64
	categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
	if err != nil {
		log.Printf("Invalid category ID: %v", err)
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	// Получаем информацию о категории
	var category Category
	err = h.db.QueryRow("SELECT id, name, description FROM categories WHERE id = ?", categoryID).
		Scan(&category.ID, &category.Name, &category.Description)
	if err != nil {
		log.Printf("Error getting category: %v", err)
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}
	log.Printf("Found category: %s", category.Name)

	// Обновляем запрос для категорий
	query := `
		SELECT p.id, p.title, p.content, p.username, p.created_at, p.user_id,
			   p.likes, p.dislikes, c.name as category_name,
			   COUNT(DISTINCT cm.id) as comment_count
		FROM posts p
		INNER JOIN post_categories pc ON p.id = pc.post_id
		INNER JOIN categories c ON pc.category_id = c.id
		LEFT JOIN comments cm ON p.id = cm.post_id
		WHERE c.id = ?
		GROUP BY p.id, p.title, p.content, p.username, p.created_at, p.user_id,
				 p.likes, p.dislikes, c.name
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
			&p.Likes, &p.Dislikes, &categoryName, &p.CommentCount,
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

func (h *Handler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Получаем все категории
	categories, err := h.getCategories()
	if err != nil {
		log.Printf("Error getting categories: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Получаем пользователя из сессии
	user := h.GetSessionUser(r)

	// Получаем параметры фильтрации и преобразуем categoryID в int64
	categoryIDStr := r.URL.Query().Get("category")
	var selectedCategoryID int64
	if categoryIDStr != "" {
		var err error
		selectedCategoryID, err = strconv.ParseInt(categoryIDStr, 10, 64)
		if err != nil {
			log.Printf("Invalid category ID: %v", err)
			selectedCategoryID = 0
		}
	}

	showMyPosts := r.URL.Query().Get("my_posts") == "true"
	showLikedPosts := r.URL.Query().Get("liked_posts") == "true"

	// Получаем посты с учетом фильтров
	posts := getPosts(h.db)

	// Если пользователь авторизован, получаем информацию о его реакциях
	if user != nil {
		for i := range posts {
			var reactionType string
			err := h.db.QueryRow(`
				SELECT type FROM reactions 
				WHERE user_id = ? AND post_id = ?`,
				user.ID, posts[i].ID,
			).Scan(&reactionType)

			if err == nil {
				posts[i].UserLiked = reactionType == "like"
				posts[i].UserDisliked = reactionType == "dislike"
			}
		}
	}

	data := TemplateData{
		User:             user,
		Posts:            posts,
		Categories:       categories,
		SelectedCategory: selectedCategoryID,
		ShowMyPosts:      showMyPosts,
		ShowLikedPosts:   showLikedPosts,
	}

	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Error rendering page", http.StatusInternalServerError)
	}
}
