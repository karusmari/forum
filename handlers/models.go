package handlers

import (
	"time"
)

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
	Comments     []Comment `json:"comments"`
	CommentCount int       `json:"comment_count"`
}

type Category struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PostCount   int 
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
	User             *User
	Post             *Post
	Posts            []Post
	Categories       []Category
	Category         *Category
	Comments         []*Comment
	CommentDataList  []CommentData
	SelectedCategory int64
	ShowMyPosts      bool
	ShowLikedPosts   bool
	Title            string
	Error            string
}

type CommentData struct {
	Comment *Comment
	User    *User
	Post    *Post
}

type ErrorData struct {
    ErrorMessage string
    ErrorCode    string
}
