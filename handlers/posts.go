package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
)

// ables the user to create a new post
func (h *Handler) CreatePost(w http.ResponseWriter, r *http.Request) {
	//checking if the user is logged in
	user := h.GetSessionUser(r)
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


	result, err := h.db.Exec(`
		INSERT INTO posts (user_id, title, content, username)
		SELECT ?, ?, ?, username FROM users WHERE id = ?
	`, user.ID, title, content, user.ID)
	if err != nil {
		log.Printf("Error creating post: %v", err)
		h.ErrorHandler(w, "Error creating post", http.StatusInternalServerError)
		return
	}

	//getting the ID of the post
	postID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error getting post ID: %v", err)
		h.ErrorHandler(w, "Error getting post ID", http.StatusInternalServerError)
		return
	}

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

	// check if title or content are empty after trimming spaces
	if title == "" || content == "" {
		h.ErrorHandler(w, "Title and content cannot be empty", http.StatusBadRequest)
		return
	}

	// //adding the post to the database
	// result, err := h.db.Exec(`
	// 	INSERT INTO posts (user_id, title, content, username)
	// 	SELECT ?, ?, ?, username FROM users WHERE id = ?
	// `, user.ID, title, content, user.ID)

	// if err != nil {
	// 	log.Printf("Error creating post: %v", err)
	// 	h.ErrorHandler(w, "Error creating post", http.StatusInternalServerError)
	// 	return
	// }


	//redirecting the user to the newly created post page
	http.Redirect(w, r, "/post/"+strconv.FormatInt(postID, 10), http.StatusSeeOther)
}


func (h *Handler) EditPost(w http.ResponseWriter, r *http.Request) {
	//removes the /post/edit/ from the URL to get the post ID
	postID := strings.TrimPrefix(r.URL.Path, "/post/edit/")
	if postID == "" {
		log.Printf("Empty post ID")
		h.ErrorHandler(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	//getting the category ID from the URL
	catId := r.URL.Query().Get("cat")

	//converting the post ID to an integer
	Category := Category{}
	Category.ID, _ = strconv.ParseInt(catId, 10, 64)

	user := h.GetSessionUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	//getting the post from the database
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
		h.ErrorHandler(w, "Post not found", http.StatusNotFound)
		return
	}

	//checking if the user is the owner of the post or an admin
	if post.UserID != user.ID && !user.IsAdmin {
		h.ErrorHandler(w, "Not authorized to edit this post", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodGet {
		//if the method is GET then recieve the categories
		categories, err := h.getCategories()
		if err != nil {
			h.ErrorHandler(w, "Error loading categories", http.StatusInternalServerError)
			return
		}

		//getting the category of the post
		post.Categories, err = h.getPostCategories(post.ID)
		if err != nil {
			h.ErrorHandler(w, "Error loading post categories", http.StatusInternalServerError)
			return
		}
		//creating the data to be displayed on the page
		data := &TemplateData{
			Title:      "Edit Post",
			User:       user,
			Post:       &post,
			Categories: categories,
			Category:   &post.Category,
		}
		//executing the template and displaying the page
		h.templates.ExecuteTemplate(w, "edit_post.html", data)
		return
	}

	//if the method is not POST, display an error message
	if r.Method != http.MethodPost {
		h.ErrorHandler(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//if the method is POST, parse the form
	if err := r.ParseForm(); err != nil {
		h.ErrorHandler(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	//beginning the transaction from the db
	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		h.ErrorHandler(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	//updating the post in the database
	_, err = tx.Exec(`
		UPDATE posts 
		SET title = ?, content = ?
		WHERE id = ?
	`, r.FormValue("title"), r.FormValue("content"), postID)

	if err != nil {
		log.Printf("Error updating post: %v", err)
		h.ErrorHandler(w, "Error updating post", http.StatusInternalServerError)
		return
	}

	//deleting the old categories
	_, err = tx.Exec("DELETE FROM post_categories WHERE post_id = ?", postID)
	if err != nil {
		log.Printf("Error deleting old categories: %v", err)
		h.ErrorHandler(w, "Error updating categories", http.StatusInternalServerError)
		return
	}

	//adding the new category
	categoryID := r.FormValue("category")
	_, err = tx.Exec(`
    INSERT INTO post_categories (post_id, category_id)
    VALUES (?, ?)
`, postID, categoryID)
	if err != nil {
		log.Printf("Error adding category %s to post %s: %v", categoryID, postID, err)
		h.ErrorHandler(w, "Error updating categories", http.StatusInternalServerError)
		return
	}

	//committing the transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		h.ErrorHandler(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/category/"+catId, http.StatusSeeOther)
}

func (h *Handler) DeletePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.ErrorHandler(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := h.GetSessionUser(r)
	if user == nil {
		h.ErrorHandler(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	postID := r.FormValue("post_id")

	//querying the database to check if the user is the owner of the post
	var userID int64
	err := h.db.QueryRow("SELECT user_id FROM posts WHERE id = ?", postID).Scan(&userID)
	if err != nil {
		log.Printf("Error checking post ownership: %v", err)
		h.ErrorHandler(w, "Post not found", http.StatusNotFound)
		return
	}

	//only the owner of the post or an admin can delete the post
	if userID != user.ID && !user.IsAdmin {
		h.ErrorHandler(w, "Not authorized to delete this post", http.StatusForbidden)
		return
	}

	//starting a transaction
	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		h.ErrorHandler(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	//delete the comments
	_, err = tx.Exec("DELETE FROM comments WHERE post_id = ?", postID)
	if err != nil {
		log.Printf("Error deleting comments: %v", err)
		http.Error(w, "Error deleting post", http.StatusInternalServerError)
		return
	}

	//deleting the category links
	_, err = tx.Exec("DELETE FROM post_categories WHERE post_id = ?", postID)
	if err != nil {
		log.Printf("Error deleting category links: %v", err)
		h.ErrorHandler(w, "Error deleting post", http.StatusInternalServerError)
		return
	}

	//
	_, err = tx.Exec("DELETE FROM reactions WHERE post_id = ?", postID)
	if err != nil {
		log.Printf("Error deleting reactions: %v", err)
		h.ErrorHandler(w, "Error deleting post", http.StatusInternalServerError)
		return
	}

	//committing the transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		h.ErrorHandler(w, "Error disabling post", http.StatusInternalServerError)
		return
	}

	//redirecting the user to the home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
