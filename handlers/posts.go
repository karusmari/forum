package handlers

import (
	"net/http"
	"strconv"
	"strings"
)

// ables the user to create a new post
func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	//checking if the user is logged in
	user := h.GetSessionUser(w, r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther) //if they are not logged in, redirect them to the login page
		return
	}

	//if the request method is GET, the user wants to create a post and it will display the create post page
	if r.Method == http.MethodGet {
		//loading the categories to choose from
		categories, err := h.getCategories()
		if err != nil {
			h.ErrorHandler(w, "Error loading categories", http.StatusInternalServerError)
			return
		}

		//creating the data to be displayed on the page
		data := &TemplateData{
			Title:      "Create Post",
			User:       user,
			Categories: categories,
		}
		//executing the template and displaying the page
		h.templates.ExecuteTemplate(w, "new_post.html", data)
		return
	}

	//if the request method is not POST, display an error message
	if r.Method != http.MethodPost {
		h.ErrorHandler(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//parsing the form
	if err := r.ParseForm(); err != nil {
		h.ErrorHandler(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	//getting the title and content of the post
	title := strings.TrimSpace(r.FormValue("title"))
	content := strings.TrimSpace(r.FormValue("content"))
	categories := r.Form["categories"] //can choose multiple categories

	// check if title or content are empty after trimming spaces
	if title == "" || content == "" {
		h.ErrorHandler(w, "Title and content cannot be empty", http.StatusBadRequest)
		return
	}

	result, err := h.db.Exec(`
		INSERT INTO posts (user_id, title, content, username)
		SELECT ?, ?, ?, username FROM users WHERE id = ?
	`, user.ID, title, content, user.ID)
	if err != nil {
		h.ErrorHandler(w, "Error creating post", http.StatusInternalServerError)
		return
	}

	//getting the ID of the post
	postID, err := result.LastInsertId()
	if err != nil {
		h.ErrorHandler(w, "Error getting post ID", http.StatusInternalServerError)
		return
	}

	for _, categoryID := range categories {
		_, err = h.db.Exec(`
			INSERT INTO post_categories (post_id, category_id)
			VALUES (?, ?)
		`, postID, categoryID)
		if err != nil {
			continue
		}
	}

	//redirecting the user to the newly created post page
	http.Redirect(w, r, "/post/"+strconv.FormatInt(postID, 10), http.StatusSeeOther)
}
