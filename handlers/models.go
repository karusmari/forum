package handlers

import (
	"time"
)

type User struct {
	ID           int64  
	Email        string 
	Username     string 
	PasswordHash string 
	IsAdmin      bool   
}

type Post struct {
	ID           int64     
	UserID       int64    
	Username     string   
	Title        string   
	Content      string    
	CreatedAt    time.Time 
	Categories   []string  
	Likes        int       `json:"likes"`
	Dislikes     int       `json:"dislikes"`
	UserLiked    bool      `json:"user_liked"`
	UserDisliked bool      `json:"user_disliked"`
	Comments     []Comment 
	CommentCount int       
	Category    Category  
}

type Category struct {
	ID          int64  
	Name        string 
	Description string 
	PostCount   int 
}

type Comment struct {
	ID           int64     
	PostID       int64     
	UserID       int64    
	Username     string    
	Content      string    
	CreatedAt    time.Time 
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
