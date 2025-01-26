package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

type Handler struct {
	db        *sql.DB
	templates *template.Template // Шаблоны страниц
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{
		db:        db,
		templates: template.Must(template.ParseGlob("templates/*.html")),
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
			   u.username,
			   COUNT(DISTINCT CASE WHEN r.type = 'like' THEN r.id END) as likes,
			   COUNT(DISTINCT CASE WHEN r.type = 'dislike' THEN r.id END) as dislikes
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN post_categories pc ON p.id = pc.post_id
		LEFT JOIN reactions r ON p.id = r.post_id
	`

	var conditions []string
	var args []interface{}

	// Фильтр по категории
	categoryID := r.URL.Query().Get("category")
	if categoryID != "" {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM post_categories pc2 WHERE pc2.post_id = p.id AND pc2.category_id = ?)")
		args = append(args, categoryID)
	}

	// Фильтры для авторизованного пользователя
	if user := h.GetSessionUser(r); user != nil {
		if r.URL.Query().Get("my_posts") == "true" {
			conditions = append(conditions, "p.user_id = ?")
			args = append(args, user.ID)
		}
		if r.URL.Query().Get("liked_posts") == "true" {
			conditions = append(conditions, "EXISTS (SELECT 1 FROM reactions r2 WHERE r2.post_id = p.id AND r2.user_id = ? AND r2.type = 'like')")
			args = append(args, user.ID)
		}
	}

	// Добавляем условия к запросу
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Группировка и сортировка
	query += " GROUP BY p.id ORDER BY p.created_at DESC"

	// Выполняем запрос
	rows, err := h.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []Post
	currentUser := h.GetSessionUser(r)

	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.CreatedAt,
			&p.Username, &p.Likes, &p.Dislikes); err != nil {
			return nil, err
		}

		// Проверяем реакции текущего пользователя
		if currentUser != nil {
			p.UserLiked = h.hasUserReaction(currentUser.ID, p.ID, "like")
			p.UserDisliked = h.hasUserReaction(currentUser.ID, p.ID, "dislike")
		}

		// Получаем категории поста
		p.Categories, err = h.getPostCategories(p.ID)
		if err != nil {
			return nil, err
		}

		posts = append(posts, p)
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

func (h *Handler) GetPostsByCategory(w http.ResponseWriter, r *http.Request) {
	categoryID := strings.TrimPrefix(r.URL.Path, "/category/")

	// Проверяем, что ID является числом
	if categoryID == "" {
		http.Error(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	// Получаем информацию о категории
	var category Category
	err := h.db.QueryRow(`
		SELECT id, name, description 
		FROM categories 
		WHERE id = ?
	`, categoryID).Scan(&category.ID, &category.Name, &category.Description)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Category not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Получаем посты для этой категории
	query := `
		SELECT DISTINCT p.id, p.user_id, p.title, p.content, p.created_at,
			   u.username,
			   COUNT(DISTINCT CASE WHEN r.type = 'like' THEN r.id END) as likes,
			   COUNT(DISTINCT CASE WHEN r.type = 'dislike' THEN r.id END) as dislikes
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN post_categories pc ON p.id = pc.post_id
		LEFT JOIN reactions r ON p.id = r.post_id
		WHERE pc.category_id = ?
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`

	rows, err := h.db.Query(query, categoryID)
	if err != nil {
		http.Error(w, "Error loading posts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post
	currentUser := h.GetSessionUser(r)

	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.CreatedAt,
			&p.Username, &p.Likes, &p.Dislikes); err != nil {
			http.Error(w, "Error scanning posts", http.StatusInternalServerError)
			return
		}

		// Проверяем реакции текущего пользователя
		if currentUser != nil {
			p.UserLiked = h.hasUserReaction(currentUser.ID, p.ID, "like")
			p.UserDisliked = h.hasUserReaction(currentUser.ID, p.ID, "dislike")
		}

		// Получаем категории поста
		p.Categories, err = h.getPostCategories(p.ID)
		if err != nil {
			http.Error(w, "Error loading categories", http.StatusInternalServerError)
			return
		}

		posts = append(posts, p)
	}

	data := &TemplateData{
		Title:      category.Name,
		User:       currentUser,
		Posts:      posts,
		Categories: []Category{category},
	}

	if err := h.templates.ExecuteTemplate(w, "category.html", data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}

// Вспомогательная функция для проверки реакции пользователя
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
		INSERT INTO posts (user_id, title, content)
		VALUES (?, ?, ?)
	`, user.ID, title, content)
	if err != nil {
		http.Error(w, "Error creating post", http.StatusInternalServerError)
		return
	}

	postID, err := result.LastInsertId()
	if err != nil {
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
			http.Error(w, "Error adding categories", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/post/"+strconv.FormatInt(postID, 10), http.StatusSeeOther)
}

func (h *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	postID := strings.TrimPrefix(r.URL.Path, "/post/")
	if postID == "" {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	var post Post
	err := h.db.QueryRow(`
		SELECT p.id, p.user_id, p.title, p.content, p.created_at,
			   u.username,
			   COUNT(DISTINCT CASE WHEN r.type = 'like' THEN r.id END) as likes,
			   COUNT(DISTINCT CASE WHEN r.type = 'dislike' THEN r.id END) as dislikes
		FROM posts p
		JOIN users u ON p.user_id = u.id
		LEFT JOIN reactions r ON p.id = r.post_id
		WHERE p.id = ?
		GROUP BY p.id
	`, postID).Scan(&post.ID, &post.UserID, &post.Title, &post.Content, &post.CreatedAt,
		&post.Username, &post.Likes, &post.Dislikes)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Post not found", http.StatusNotFound)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Получаем категории поста
	post.Categories, err = h.getPostCategories(post.ID)
	if err != nil {
		http.Error(w, "Error loading categories", http.StatusInternalServerError)
		return
	}

	data := &TemplateData{
		Title: post.Title,
		User:  h.GetSessionUser(r),
		Post:  &post,
	}

	h.templates.ExecuteTemplate(w, "post.html", data)
}

func (h *Handler) ReactToPost(w http.ResponseWriter, r *http.Request) {
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

	postID := r.FormValue("post_id")
	reactionType := r.FormValue("type")

	if reactionType != "like" && reactionType != "dislike" {
		http.Error(w, "Invalid reaction type", http.StatusBadRequest)
		return
	}

	_, err := h.db.Exec(`
		INSERT INTO reactions (user_id, post_id, type)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id, post_id) DO UPDATE SET type = ?
	`, user.ID, postID, reactionType, reactionType)

	if err != nil {
		http.Error(w, "Error saving reaction", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
