package handlers

import "time"

type User struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	IsAdmin      bool   `json:"is_admin"`
}

type Post struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	Username     string    `json:"username"`
	Title        string    `json:"title"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
	Categories   []string  `json:"categories"`
	Likes        int       `json:"likes"`
	Dislikes     int       `json:"dislikes"`
	UserLiked    bool      `json:"user_liked"`
	UserDisliked bool      `json:"user_disliked"`
}

type Category struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Comment struct {
	ID           int64     `json:"id"`
	PostID       int64     `json:"post_id"`
	UserID       int64     `json:"user_id"`
	Username     string    `json:"username"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
	Likes        int       `json:"likes"`
	Dislikes     int       `json:"dislikes"`
	UserLiked    bool      `json:"user_liked"`
	UserDisliked bool      `json:"user_disliked"`
}

type TemplateData struct {
	Title            string
	User             *User
	Post             *Post
	Posts            []Post
	Categories       []Category
	Error            string
	SelectedCategory int64
	ShowMyPosts      bool
	ShowLikedPosts   bool
}
