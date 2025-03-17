package main

import (
	"fmt"
	"forum/handlers"
	"html/template"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Initialize database
	db, err := handlers.InitDB()
	if err != nil {
		log.Fatal("Failed to initialize the database:", err)
	}
	defer db.Close()

	templates, err := template.ParseGlob("templates/*.html")
	if err != nil {
		log.Fatal("Failed to parse templates:", err)
	}

	// Create connection to the database
	h := handlers.NewHandler(db, templates)

	// Setup routes
	http.HandleFunc("/", h.HomeHandler)
	http.HandleFunc("/rules", h.Rules)
	http.HandleFunc("/register", h.HandleRegister)
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
