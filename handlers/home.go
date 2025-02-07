package handlers

import (
	"database/sql"
	"fmt"
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
		h.ErrorHandler(w, "Category not found", http.StatusNotFound)
		return
	}

	// Конвертируем строку в int64
	categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
	if err != nil {
		log.Printf("Invalid category ID: %v", err)
		h.ErrorHandler(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	// Получаем информацию о категории
	var category Category
	err = h.db.QueryRow("SELECT id, name, description FROM categories WHERE id = ?", categoryID).
		Scan(&category.ID, &category.Name, &category.Description)
	if err != nil {
		log.Printf("Error getting category: %v", err)
		h.ErrorHandler(w, "Category not found", http.StatusNotFound)
		return
	}
	log.Printf("Found category: %s", category.Name)

	// Получаем текущего пользователя
	user := h.GetSessionUser(r)

	// Обновляем запрос для получения постов с информацией о лайках
	query := `
		SELECT p.id, p.title, p.content, p.username, p.created_at, p.user_id,
			   COUNT(DISTINCT cm.id) as comment_count,
			   EXISTS(
				   SELECT 1 FROM reactions r 
				   WHERE r.post_id = p.id 
				   AND r.user_id = ? 
				   AND r.type = 'like'
			   ) as user_liked
		FROM posts p
		INNER JOIN post_categories pc ON p.id = pc.post_id
		LEFT JOIN comments cm ON p.id = cm.post_id
		WHERE pc.category_id = ?
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`

	// Выполняем запрос с ID пользователя
	var userID int64
	if user != nil {
		userID = user.ID
	}

	rows, err := h.db.Query(query, userID, categoryID)
	if err != nil {
		log.Printf("Error getting posts: %v", err)
		h.ErrorHandler(w, "Error getting posts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		var p Post
		err := rows.Scan(
			&p.ID, &p.Title, &p.Content, &p.Username, &p.CreatedAt, &p.UserID,
			&p.CommentCount, &p.UserLiked,
		)
		if err != nil {
			log.Printf("Error scanning post: %v", err)
			continue
		}
		posts = append(posts, p)
	}

	data := TemplateData{
		Title:    category.Name,
		User:     user,
		Category: &category,
		Posts:    posts,
	}

	h.templates.ExecuteTemplate(w, "category.html", data)
}

func (h *Handler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		h.ErrorHandler(w, "Page not found", http.StatusNotFound)
		return
	}

	// Получаем все категории
	categories, err := h.getCategories()
	if err != nil {
		log.Printf("Error getting categories: %v", err)
		h.ErrorHandler(w, "Server error", http.StatusInternalServerError)
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
		h.ErrorHandler(w, "Error rendering page", http.StatusInternalServerError)
	}
}

// Обновляем существующий метод GetPost в handlers.go:
func (h *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	postID := r.URL.Path[len("/post/"):]
	log.Printf("Getting post with ID: %s", postID)
	catId := r.URL.Query().Get("cat")
	Category := Category{}
	Category.ID, _ = strconv.ParseInt(catId, 10, 64)

	// Получаем пользователя из сессии
	user := h.GetSessionUser(r)

	// Получаем пост
	post, err := h.getPostByID(postID)
	if err != nil {
		log.Printf("Error getting post: %v", err)
		h.ErrorHandler(w, "Post not found", http.StatusNotFound)
		return
	}

	// Проверяем реакции пользователя на пост
	if user != nil {
		post.UserLiked = h.hasUserReaction(user.ID, post.ID, "like")
		post.UserDisliked = h.hasUserReaction(user.ID, post.ID, "dislike")
	}

	// Получаем комментарии
	comments, err := h.getComments(post.ID)
	if err != nil {
		log.Printf("Error getting comments: %v", err)
		h.ErrorHandler(w, "Error loading comments", http.StatusInternalServerError)
		return
	}
	log.Printf("Found %d comments for post %d", len(comments), post.ID)

	// Проверяем реакции пользователя на комментарии
	if user != nil {
		for _, comment := range comments {
			comment.UserLiked = h.hasCommentReaction(user.ID, comment.ID, "like")
			comment.UserDisliked = h.hasCommentReaction(user.ID, comment.ID, "dislike")
		}
	}

	// Подготавливаем данные для комментариев
	var commentDataList []CommentData
	for _, comment := range comments {
		commentData := CommentData{
			Comment: comment,
			User:    user,
			Post:    post,
		}
		commentDataList = append(commentDataList, commentData)
	}

	data := TemplateData{
		Title:           post.Title,
		Post:            post,
		User:            user,
		Comments:        comments,
		CommentDataList: commentDataList,
		Category:        &Category,
	}

	if err := h.templates.ExecuteTemplate(w, "post.html", data); err != nil {
		log.Printf("Template error: %v", err)
		h.ErrorHandler(w, "Error rendering page", http.StatusInternalServerError)
	}
}

// Добавляем метод getPostByID
func (h *Handler) getPostByID(postID string) (*Post, error) {
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
		GROUP BY p.id, p.user_id, p.title, p.content, p.created_at, u.username
	`, postID).Scan(
		&post.ID, &post.UserID, &post.Title, &post.Content, &post.CreatedAt,
		&post.Username, &post.Likes, &post.Dislikes,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("post not found: %s", postID)
		}
		return nil, err
	}

	// Получаем категории поста
	post.Categories, err = h.getPostCategories(post.ID)
	if err != nil {
		return nil, err
	}

	// Получаем количество комментариев
	var commentCount int
	err = h.db.QueryRow(`
		SELECT COUNT(*) 
		FROM comments 
		WHERE post_id = ?
	`, post.ID).Scan(&commentCount)
	if err != nil {
		return nil, err
	}
	post.CommentCount = commentCount

	return &post, nil
}

// Add a new method to get comments
func (h *Handler) getComments(postID int64) ([]*Comment, error) {
	rows, err := h.db.Query(`
		SELECT c.id, c.user_id, c.content, c.created_at, c.username,
			   COUNT(CASE WHEN r.type = 'like' THEN 1 END) as likes,
			   COUNT(CASE WHEN r.type = 'dislike' THEN 1 END) as dislikes
		FROM comments c
		LEFT JOIN reactions r ON c.id = r.comment_id
		WHERE c.post_id = ?
		GROUP BY c.id, c.user_id, c.content, c.created_at, c.username
		ORDER BY c.created_at DESC
	`, postID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*Comment
	for rows.Next() {
		var c Comment
		err := rows.Scan(
			&c.ID, &c.UserID, &c.Content, &c.CreatedAt, &c.Username,
			&c.Likes, &c.Dislikes,
		)
		if err != nil {
			return nil, err
		}
		comments = append(comments, &c)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}

// Add a new method to check reactions on comments
func (h *Handler) hasCommentReaction(userID int64, commentID int64, reactionType string) bool {
	var exists bool
	err := h.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM reactions 
			WHERE user_id = ? AND comment_id = ? AND type = ?
		)
	`, userID, commentID, reactionType).Scan(&exists)

	if err != nil {
		return false
	}
	return exists
}
