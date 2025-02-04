package handlers

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Handler struct {
	db        *sql.DB
	templates *template.Template
}

// Добавляем структуру для ответа API
type CommentReactionResponse struct {
	Success bool `json:"success"`
	Likes    int  `json:"likes"`
	Dislikes int  `json:"dislikes"`
}

func NewHandler(db *sql.DB) *Handler {
	funcMap := template.FuncMap{
		"timezone": func(name string) *time.Location {
			loc, err := time.LoadLocation(name)
			if err != nil {
				return time.UTC
			}
			return loc
		},
	}

	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))
	return &Handler{
		db:        db,
		templates: tmpl,
	}
}

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	user := h.GetSessionUser(r)

	// Получаем значения фильтров из URL
	selectedCategory, _ := strconv.ParseInt(r.URL.Query().Get("category"), 10, 64)
	showMyPosts := r.URL.Query().Get("my_posts") == "true"
	showLikedPosts := r.URL.Query().Get("liked_posts") == "true"

	// Получаем посты с учетом фильтров
	posts, err := h.getPosts(r)
	if err != nil {
		http.Error(w, "Error loading posts", http.StatusInternalServerError)
		return
	}

	categories, err := h.getCategories()
	if err != nil {
		http.Error(w, "Error loading categories", http.StatusInternalServerError)
		return
	}

	data := &TemplateData{
		Title:            "Home",
		User:             user,
		Posts:            posts,
		Categories:       categories,
		SelectedCategory: selectedCategory,
		ShowMyPosts:      showMyPosts,
		ShowLikedPosts:   showLikedPosts,
	}

	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}

func (h *Handler) getCategories() ([]Category, error) {
	rows, err := h.db.Query(`
		SELECT id, name, description 
		FROM categories
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var cat Category
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Description); err != nil {
			return nil, err
		}
		categories = append(categories, cat)
	}
	return categories, nil
}

func (h *Handler) getPosts(r *http.Request) ([]Post, error) {
	query := `
		SELECT DISTINCT p.id, p.user_id, p.title, p.content, p.created_at,
			   p.username,
			   COUNT(DISTINCT CASE WHEN r.type = 'like' THEN r.id END) as likes,
			   COUNT(DISTINCT CASE WHEN r.type = 'dislike' THEN r.id END) as dislikes
		FROM posts p
		LEFT JOIN reactions r ON p.id = r.post_id
		LEFT JOIN post_categories pc ON p.id = pc.post_id
	`

	var conditions []string
	var args []interface{}

	// Фильтр по категории
	categoryID := r.URL.Query().Get("category")
	if categoryID != "" {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM post_categories pc2 WHERE pc2.post_id = p.id AND pc2.category_id = ?)")
		args = append(args, categoryID)
	}

	// Фильтр по моим постам
	if r.URL.Query().Get("my_posts") == "true" {
		user := h.GetSessionUser(r)
		if user != nil {
			conditions = append(conditions, "p.user_id = ?")
			args = append(args, user.ID)
		}
	}

	// Фильтр по лайкнутым постам
	if r.URL.Query().Get("liked_posts") == "true" {
		user := h.GetSessionUser(r)
		if user != nil {
			conditions = append(conditions, "EXISTS (SELECT 1 FROM reactions r2 WHERE r2.post_id = p.id AND r2.user_id = ? AND r2.type = 'like')")
			args = append(args, user.ID)
		}
	}

	// Добавляем условия в запрос
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Группировка и сортировка
	query += `
		GROUP BY p.id, p.user_id, p.title, p.content, p.created_at, p.username
		ORDER BY p.created_at DESC
	`

	// Выполняем запрос
	rows, err := h.db.Query(query, args...)
	if err != nil {
		log.Printf("Error querying posts: %v", err)
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var post Post
		err := rows.Scan(
			&post.ID,
			&post.UserID,
			&post.Title,
			&post.Content,
			&post.CreatedAt,
			&post.Username,
			&post.Likes,
			&post.Dislikes,
		)
		if err != nil {
			log.Printf("Error scanning post: %v", err)
			continue
		}

		// Получаем категории для поста
		post.Categories, err = h.getPostCategories(post.ID)
		if err != nil {
			log.Printf("Error getting categories for post %d: %v", post.ID, err)
			continue
		}

		// Проверяем реакции пользователя
		user := h.GetSessionUser(r)
		if user != nil {
			post.UserLiked = h.hasUserReaction(user.ID, post.ID, "like")
			post.UserDisliked = h.hasUserReaction(user.ID, post.ID, "dislike")
		}

		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error iterating posts: %v", err)
		return nil, err
	}

	return posts, nil
}

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

func (h *Handler) Rules(w http.ResponseWriter, r *http.Request) {
	data := &TemplateData{
		Title: "Forum Rules",
		User:  h.GetSessionUser(r),
	}
	h.templates.ExecuteTemplate(w, "rules.html", data)
}

func (h *Handler) Categories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.getCategories()
	if err != nil {
		http.Error(w, "Error loading categories", http.StatusInternalServerError)
		return
	}

	data := &TemplateData{
		Title:      "Categories",
		User:       h.GetSessionUser(r),
		Categories: categories,
	}
	h.templates.ExecuteTemplate(w, "categories.html", data)
}

func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	user := h.GetSessionUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		categories, err := h.getCategories()
		if err != nil {
			http.Error(w, "Error loading categories", http.StatusInternalServerError)
			return
		}

		data := &TemplateData{
			Title:      "Create Post",
			User:       user,
			Categories: categories,
		}
		h.templates.ExecuteTemplate(w, "new_post.html", data)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	content := r.FormValue("content")
	categories := r.Form["categories"] // Получаем массив выбранных категорий

	// Создаем пост
	result, err := h.db.Exec(`
		INSERT INTO posts (user_id, title, content, username)
		SELECT ?, ?, ?, username FROM users WHERE id = ?
	`, user.ID, title, content, user.ID)
	if err != nil {
		log.Printf("Error creating post: %v", err)
		http.Error(w, "Error creating post", http.StatusInternalServerError)
		return
	}

	postID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error getting post ID: %v", err)
		http.Error(w, "Error getting post ID", http.StatusInternalServerError)
		return
	}

	// Добавляем категории к посту
	for _, categoryID := range categories {
		_, err = h.db.Exec(`
			INSERT INTO post_categories (post_id, category_id)
			VALUES (?, ?)
		`, postID, categoryID)
		if err != nil {
			log.Printf("Error adding category %s to post %d: %v", categoryID, postID, err)
			continue
		}
	}

	http.Redirect(w, r, "/post/"+strconv.FormatInt(postID, 10), http.StatusSeeOther)
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

func (h *Handler) EditPost(w http.ResponseWriter, r *http.Request) {
	// Получаем ID поста из URL
	postID := strings.TrimPrefix(r.URL.Path, "/post/edit/")
	if postID == "" {
		log.Printf("Empty post ID")
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	user := h.GetSessionUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Получаем информацию о посте
	var post Post
	err := h.db.QueryRow(`
		SELECT p.id, p.user_id, p.title, p.content, p.created_at,
			   p.username
		FROM posts p
		WHERE p.id = ?
	`, postID).Scan(
		&post.ID, &post.UserID, &post.Title, &post.Content, &post.CreatedAt,
		&post.Username,
	)

	if err != nil {
		log.Printf("Error getting post: %v", err)
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	// Проверяем права на редактирование
	if post.UserID != user.ID && !user.IsAdmin {
		http.Error(w, "Not authorized to edit this post", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodGet {
		// Получаем все категории
		categories, err := h.getCategories()
		if err != nil {
			http.Error(w, "Error loading categories", http.StatusInternalServerError)
			return
		}

		// Получаем выбранные категории поста
		post.Categories, err = h.getPostCategories(post.ID)
		if err != nil {
			http.Error(w, "Error loading post categories", http.StatusInternalServerError)
			return
		}

		data := &TemplateData{
			Title:      "Edit Post",
			User:       user,
			Post:       &post,
			Categories: categories,
		}
		h.templates.ExecuteTemplate(w, "edit_post.html", data)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Обработка POST запроса
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Начинаем транзакцию
	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Обновляем пост
	_, err = tx.Exec(`
		UPDATE posts 
		SET title = ?, content = ?
		WHERE id = ?
	`, r.FormValue("title"), r.FormValue("content"), postID)

	if err != nil {
		log.Printf("Error updating post: %v", err)
		http.Error(w, "Error updating post", http.StatusInternalServerError)
		return
	}

	// Удаляем старые категории
	_, err = tx.Exec("DELETE FROM post_categories WHERE post_id = ?", postID)
	if err != nil {
		log.Printf("Error deleting old categories: %v", err)
		http.Error(w, "Error updating categories", http.StatusInternalServerError)
		return
	}

	// Добавляем новые категории
	for _, categoryID := range r.Form["categories"] {
		_, err = tx.Exec(`
			INSERT INTO post_categories (post_id, category_id)
			VALUES (?, ?)
		`, postID, categoryID)
		if err != nil {
			log.Printf("Error adding category %s to post %s: %v", categoryID, postID, err)
			continue
		}
	}

	// Завершаем транзакцию
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
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

	var data struct {
		CommentID int64  `json:"comment_id"`
		Type      string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
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

	// Удаляем существующую реакцию пользователя
	_, err = tx.Exec(`
		DELETE FROM reactions 
		WHERE user_id = ? AND comment_id = ?`,
		user.ID, data.CommentID,
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Добавляем новую реакцию
	_, err = tx.Exec(`
		INSERT INTO reactions (user_id, comment_id, type)
		VALUES (?, ?, ?)`,
		user.ID, data.CommentID, data.Type,
	)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Обновляем счетчики в комментарии
	var likes, dislikes int
	err = tx.QueryRow(`
		SELECT 
			COUNT(CASE WHEN type = 'like' THEN 1 END) as likes,
			COUNT(CASE WHEN type = 'dislike' THEN 1 END) as dislikes
		FROM reactions
		WHERE comment_id = ?`,
		data.CommentID,
	).Scan(&likes, &dislikes)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	response := CommentReactionResponse{
		Success:  true,
		Likes:    likes,
		Dislikes: dislikes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
