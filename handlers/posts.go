package handlers

import (
	"database/sql"
	"fmt"
	"log"
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
			log.Printf("Error loading catgories: %v", err)
			h.ErrorHandler(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
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
		log.Printf("Error creating post: %v", err)
		h.ErrorHandler(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}

	//getting the ID of the post
	postID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error getting post id: %v", err)
		h.ErrorHandler(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}

	
	// If no categories were selected, use category ID = 1
	if len(categories) == 0 {
		categories = append(categories, "1") // Use category ID 1 if no category is selected
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

// a function to prepare the data for the post.html template
func (h *Handler) GetPost(w http.ResponseWriter, r *http.Request) {
	//get the post ID from the URL by cutting the /post/ from the URL
	postID := r.URL.Path[len("/post/"):]
	//recieve the category ID from the URL
	catId := r.URL.Query().Get("cat")

	Category := Category{}
	Category.ID, _ = strconv.ParseInt(catId, 10, 64)

	//get the user from the session
	user := h.GetSessionUser(w, r)

	//calling out the getPostByID function to get the post with the given ID
	post, err := h.getPostByID(postID)
	if err != nil {
		h.ErrorHandler(w, "Post not found", http.StatusNotFound)
		return
	}

	//if user has a session, check if the user has liked or disliked the post
	if user != nil {
		post.UserLiked = h.hasUserReaction(user.ID, post.ID, "like")
		post.UserDisliked = h.hasUserReaction(user.ID, post.ID, "dislike")
	}

	//calling out the getComments function to get the comments of the post
	comments, err := h.getComments(post.ID)
	if err != nil {
		log.Printf("Error loading comments: %v", err)
		h.ErrorHandler(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
		return
	}

	//checking if the user has liked or disliked the comments
	if user != nil {
		for _, comment := range comments {
			comment.UserLiked = h.hasCommentReaction(user.ID, comment.ID, "like")
			comment.UserDisliked = h.hasCommentReaction(user.ID, comment.ID, "dislike")
		}
	}

	// prepare data for the template
	var commentDataList []CommentData
	for _, comment := range comments {
		//each comment will have a user and a post
		commentData := CommentData{
			Comment: comment,
			User:    user,
			Post:    post,
		}
		commentDataList = append(commentDataList, commentData)
	}

	//collecting all the data into a struct
	data := TemplateData{
		Title:           post.Title,
		Post:            post,
		User:            user,
		Comments:        comments,
		CommentDataList: commentDataList,
		Category:        &Category,
	}

	//render the post.html template with the data
	if err := h.templates.ExecuteTemplate(w, "post.html", data); err != nil {
		log.Printf("Error rendering page: %v", err)
		h.ErrorHandler(w, "Something went wrong. Please try again later.", http.StatusInternalServerError)
	}
}

// a function to get a specific post from the database
func (h *Handler) getPostByID(postID string) (*Post, error) {
	var post Post
	err := h.db.QueryRow(`
		SELECT p.id, p.user_id, p.title, p.content, p.created_at, u.username,
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

	//recieve the categories of the post
	post.Categories, err = h.getPostCategories(post.ID)
	if err != nil {
		return nil, err
	}

	//recieve the comment count of the post
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

// getting the posts from the database with the post ID
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
