package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"time"
)

type Handler struct {
	db        *sql.DB
	templates *template.Template
}

// Добавляем структуру для ответа API
type CommentReactionResponse struct {
	Success  bool `json:"success"`
	Likes    int  `json:"likes"`
	Dislikes int  `json:"dislikes"`
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

// func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
// 	if r.URL.Path != "/" {
// 		http.NotFound(w, r)
// 		return
// 	}

// 	user := h.GetSessionUser(r)

// 	// Получаем значения фильтров из URL
// 	selectedCategory, _ := strconv.ParseInt(r.URL.Query().Get("category"), 10, 64)
// 	showMyPosts := r.URL.Query().Get("my_posts") == "true"
// 	showLikedPosts := r.URL.Query().Get("liked_posts") == "true"

// 	// Получаем посты с учетом фильтров
// 	posts, err := h.getPosts(r)
// 	if err != nil {
// 		http.Error(w, "Error loading posts", http.StatusInternalServerError)
// 		return
// 	}

// 	categories, err := h.getCategories()
// 	if err != nil {
// 		http.Error(w, "Error loading categories", http.StatusInternalServerError)
// 		return
// 	}

// 	data := &TemplateData{
// 		Title:            "Home",
// 		User:             user,
// 		Posts:            posts,
// 		Categories:       categories,
// 		SelectedCategory: selectedCategory,
// 		ShowMyPosts:      showMyPosts,
// 		ShowLikedPosts:   showLikedPosts,
// 	}

// 	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
// 		http.Error(w, "Error rendering template", http.StatusInternalServerError)
// 	}
// }

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

// func (h *Handler) getPosts(r *http.Request) ([]Post, error) {
// 	query := `
// 		SELECT DISTINCT p.id, p.user_id, p.title, p.content, p.created_at,
// 			   p.username,
// 			   COUNT(DISTINCT CASE WHEN r.type = 'like' THEN r.id END) as likes,
// 			   COUNT(DISTINCT CASE WHEN r.type = 'dislike' THEN r.id END) as dislikes
// 		FROM posts p
// 		LEFT JOIN reactions r ON p.id = r.post_id
// 		LEFT JOIN post_categories pc ON p.id = pc.post_id
// 	`

// 	var conditions []string
// 	var args []interface{}

// 	// Фильтр по категории
// 	categoryID := r.URL.Query().Get("category")
// 	if categoryID != "" {
// 		conditions = append(conditions, "EXISTS (SELECT 1 FROM post_categories pc2 WHERE pc2.post_id = p.id AND pc2.category_id = ?)")
// 		args = append(args, categoryID)
// 	}

// 	// Фильтр по моим постам
// 	if r.URL.Query().Get("my_posts") == "true" {
// 		user := h.GetSessionUser(r)
// 		if user != nil {
// 			conditions = append(conditions, "p.user_id = ?")
// 			args = append(args, user.ID)
// 		}
// 	}

// 	// Фильтр по лайкнутым постам
// 	if r.URL.Query().Get("liked_posts") == "true" {
// 		user := h.GetSessionUser(r)
// 		if user != nil {
// 			conditions = append(conditions, "EXISTS (SELECT 1 FROM reactions r2 WHERE r2.post_id = p.id AND r2.user_id = ? AND r2.type = 'like')")
// 			args = append(args, user.ID)
// 		}
// 	}

// 	// Добавляем условия в запрос
// 	if len(conditions) > 0 {
// 		query += " WHERE " + strings.Join(conditions, " AND ")
// 	}

// 	// Группировка и сортировка
// 	query += `
// 		GROUP BY p.id, p.user_id, p.title, p.content, p.created_at, p.username
// 		ORDER BY p.created_at DESC
// 	`

// 	// Выполняем запрос
// 	rows, err := h.db.Query(query, args...)
// 	if err != nil {
// 		log.Printf("Error querying posts: %v", err)
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var posts []Post
// 	for rows.Next() {
// 		var post Post
// 		err := rows.Scan(
// 			&post.ID,
// 			&post.UserID,
// 			&post.Title,
// 			&post.Content,
// 			&post.CreatedAt,
// 			&post.Username,
// 			&post.Likes,
// 			&post.Dislikes,
// 		)
// 		if err != nil {
// 			log.Printf("Error scanning post: %v", err)
// 			continue
// 		}

// 		// Получаем категории для поста
// 		post.Categories, err = h.getPostCategories(post.ID)
// 		if err != nil {
// 			log.Printf("Error getting categories for post %d: %v", post.ID, err)
// 			continue
// 		}

// 		// Проверяем реакции пользователя
// 		user := h.GetSessionUser(r)
// 		if user != nil {
// 			post.UserLiked = h.hasUserReaction(user.ID, post.ID, "like")
// 			post.UserDisliked = h.hasUserReaction(user.ID, post.ID, "dislike")
// 		}

// 		posts = append(posts, post)
// 	}

// 	if err = rows.Err(); err != nil {
// 		log.Printf("Error iterating posts: %v", err)
// 		return nil, err
// 	}

// 	return posts, nil
// }

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

// func (h *Handler) Categories(w http.ResponseWriter, r *http.Request) {
// 	categories, err := h.getCategories()
// 	if err != nil {
// 		http.Error(w, "Error loading categories", http.StatusInternalServerError)
// 		return
// 	}

// 	data := &TemplateData{
// 		Title:      "Categories",
// 		User:       h.GetSessionUser(r),
// 		Categories: categories,
// 	}
// 	h.templates.ExecuteTemplate(w, "categories.html", data)
// }
