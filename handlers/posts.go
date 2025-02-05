package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

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

func (h *Handler) DeletePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := h.GetSessionUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	postID := r.FormValue("post_id")

	// Проверяем, что пользователь имеет право удалить пост
	var userID int64
	err := h.db.QueryRow("SELECT user_id FROM posts WHERE id = ?", postID).Scan(&userID)
	if err != nil {
		log.Printf("Error checking post ownership: %v", err)
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	// Разрешаем удаление только владельцу поста или админу
	if userID != user.ID && !user.IsAdmin {
		http.Error(w, "Not authorized to delete this post", http.StatusForbidden)
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

	// Удаляем связанные комментарии
	_, err = tx.Exec("DELETE FROM comments WHERE post_id = ?", postID)
	if err != nil {
		log.Printf("Error deleting comments: %v", err)
		http.Error(w, "Error deleting post", http.StatusInternalServerError)
		return
	}

	// Удаляем связи с категориями
	_, err = tx.Exec("DELETE FROM post_categories WHERE post_id = ?", postID)
	if err != nil {
		log.Printf("Error deleting category links: %v", err)
		http.Error(w, "Error deleting post", http.StatusInternalServerError)
		return
	}

	// Удаляем реакции
	_, err = tx.Exec("DELETE FROM reactions WHERE post_id = ?", postID)
	if err != nil {
		log.Printf("Error deleting reactions: %v", err)
		http.Error(w, "Error deleting post", http.StatusInternalServerError)
		return
	}

	// Удаляем сам пост
	_, err = tx.Exec("DELETE FROM posts WHERE id = ?", postID)
	if err != nil {
		log.Printf("Error deleting post: %v", err)
		http.Error(w, "Error deleting post", http.StatusInternalServerError)
		return
	}

	// Завершаем транзакцию
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Error deleting post", http.StatusInternalServerError)
		return
	}

	// Перенаправляем на главную страницу
	http.Redirect(w, r, "/", http.StatusSeeOther)
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

	postID := r.FormValue("post_id")
	reactionType := r.FormValue("type")

	if reactionType != "like" && reactionType != "dislike" {
		http.Error(w, "Invalid reaction type", http.StatusBadRequest)
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

	// Удаляем существующую реакцию, если есть
	_, err = tx.Exec(`
		DELETE FROM reactions 
		WHERE user_id = ? AND post_id = ?
	`, user.ID, postID)
	if err != nil {
		log.Printf("Error removing old reaction: %v", err)
		http.Error(w, "Error saving reaction", http.StatusInternalServerError)
		return
	}

	// Добавляем новую реакцию
	_, err = tx.Exec(`
		INSERT INTO reactions (user_id, post_id, type)
		VALUES (?, ?, ?)
	`, user.ID, postID, reactionType)
	if err != nil {
		log.Printf("Error adding new reaction: %v", err)
		http.Error(w, "Error saving reaction", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Возвращаем обновленное количество реакций
	var likes, dislikes int
	err = h.db.QueryRow(`
		SELECT 
			COUNT(CASE WHEN type = 'like' THEN 1 END) as likes,
			COUNT(CASE WHEN type = 'dislike' THEN 1 END) as dislikes
		FROM reactions
		WHERE post_id = ?
	`, postID).Scan(&likes, &dislikes)
	if err != nil {
		log.Printf("Error getting reaction counts: %v", err)
		http.Error(w, "Error getting reaction counts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"likes":    likes,
		"dislikes": dislikes,
	})
}
