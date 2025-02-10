package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"forum/handlers"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	//open the database
	db, err := sql.Open("sqlite3", "./database/forum.db?_loc=auto")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := handlers.InitDB(db); err != nil {
		log.Fatal("Failed to initialize the database:", err)
	}

	// Create handlers
	h := handlers.NewHandler(db)

	// Setup routes
	http.HandleFunc("/", h.HomeHandler)
	// http.HandleFunc("/categories", h.Categories)
	http.HandleFunc("/rules", h.Rules)
	http.HandleFunc("/register", h.SignUp)
	http.HandleFunc("/login", h.HandleLogin)
	http.HandleFunc("/logout", h.LogoutHandler)
	http.HandleFunc("/post/new", h.CreatePost)
	http.HandleFunc("/post/", h.GetPost)
	http.HandleFunc("/category/", h.CategoryHandler)
	http.HandleFunc("/api/react", h.PostReaction)
	http.HandleFunc("/api/comment", h.AddComment)
	http.HandleFunc("/api/comment/react", h.HandleCommentReaction)

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	fmt.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil)) // Logs the error and exits.
}
