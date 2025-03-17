package handlers

import (
	"log"
	"net/http"
	"strconv"
)

// HomeHandler is a handler for the home page
func (h *Handler) HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		h.ErrorHandler(w, "Page not found", http.StatusNotFound)
		return
	}

	//calling out the getCategories function to get all the categories
	categories, err := h.getCategories()
	if err != nil {
		log.Printf("Server error: %v", err)
		h.ErrorHandler(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}

	//getting the user from the session
	user := h.GetSessionUser(w, r)

	//collecting all the data into a struct

	data := TemplateData{
		User:       user,
		Categories: categories,
	}

	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("Error rendering page: %v", err)
		h.ErrorHandler(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
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
		ORDER BY c.id
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

func (h *Handler) CategoryHandler(w http.ResponseWriter, r *http.Request) {
	//recieve the category ID from the URL
	categoryIDStr := r.URL.Path[len("/category/"):]

	//if the category ID is empty, return a 404 error
	if categoryIDStr == "" {
		h.ErrorHandler(w, "Category not found", http.StatusNotFound)
		return
	}

	//convert the category ID to an int64 because it is a string
	categoryID, err := strconv.ParseInt(categoryIDStr, 10, 64)
	if err != nil {
		h.ErrorHandler(w, "Invalid category ID", http.StatusBadRequest)
		return
	}

	//query the database to get the category with the given ID
	var category Category
	err = h.db.QueryRow("SELECT id, name, description FROM categories WHERE id = ?", categoryID).
		Scan(&category.ID, &category.Name, &category.Description)
	if err != nil {
		h.ErrorHandler(w, "Category not found", http.StatusNotFound)
		return
	}

	//gets the user who has a session right now
	user := h.GetSessionUser(w, r)

	//query to get the posts with the given category ID
	query := `
		SELECT p.id, p.title, p.content, p.username, p.created_at, p.user_id,
		COUNT(DISTINCT cm.id) as comment_count,
		EXISTS(SELECT 1 FROM reactions r WHERE r.post_id = p.id AND r.user_id = ? AND r.type = 'like') as user_liked
		FROM posts p
		INNER JOIN post_categories pc ON p.id = pc.post_id
		LEFT JOIN comments cm ON p.id = cm.post_id
		WHERE pc.category_id = ?
		GROUP BY p.id
		ORDER BY p.created_at DESC
	`

	//creates an user ID (0, if the user is not logged in)
	var userID int64
	if user != nil {
		userID = user.ID
	}

	//starts the query to get the posts with the given category ID
	rows, err := h.db.Query(query, userID, categoryID)
	if err != nil {
		log.Printf("Error getting posts: %v", err)
		h.ErrorHandler(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	//collecting all the posts into a slice
	var posts []Post
	for rows.Next() {
		var p Post
		err := rows.Scan(
			&p.ID, &p.Title, &p.Content, &p.Username, &p.CreatedAt, &p.UserID,
			&p.CommentCount, &p.UserLiked,
		)
		if err != nil {
			continue
		}
		posts = append(posts, p)
	}

	if user == nil {
		user = &User{
			ID:       0,
			Username: "",
			Email:    "",
			IsAdmin:  false,
		}
	}

	//collecting all the data into a struct
	data := TemplateData{
		Title:    category.Name,
		User:     user,
		Category: &category,
		Posts:    posts,
	}

	//render the category.html template with the data
	h.templates.ExecuteTemplate(w, "category.html", data)
}
