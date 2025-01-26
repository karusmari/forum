package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"forum/handlers"

	_ "github.com/mattn/go-sqlite3"
)

func initDB(db *sql.DB) error {
	// Читаем SQL файл
	sqlFile, err := os.ReadFile("database/schema.sql")
	if err != nil {
		return err
	}

	// Выполняем SQL запросы
	_, err = db.Exec(string(sqlFile))
	return err
}

func checkDB(db *sql.DB) {
	// Проверка таблицы users
	var userCount int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		log.Printf("Error checking users table: %v", err)
	} else {
		log.Printf("Users in database: %d", userCount)
	}

	// Проверка таблицы posts
	var postCount int
	err = db.QueryRow("SELECT COUNT(*) FROM posts").Scan(&postCount)
	if err != nil {
		log.Printf("Error checking posts table: %v", err)
	} else {
		log.Printf("Posts in database: %d", postCount)
	}

	// Проверка таблицы categories
	var categoryCount int
	err = db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&categoryCount)
	if err != nil {
		log.Printf("Error checking categories table: %v", err)
	} else {
		log.Printf("Categories in database: %d", categoryCount)
	}
}

func addTestData(db *sql.DB) error {
	// Добавляем тестовые категории
	categories := []string{
		"Technology",
		"Science",
		"Programming",
		"Gaming",
		"Movies",
	}

	for _, cat := range categories {
		_, err := db.Exec(`
			INSERT INTO categories (name, description)
			VALUES (?, ?)
			ON CONFLICT(name) DO NOTHING
		`, cat, "Description for "+cat)
		if err != nil {
			return err
		}
	}

	// Делаем первого пользователя админом
	_, err := db.Exec(`
		UPDATE users SET is_admin = TRUE WHERE id = 1
	`)
	if err != nil {
		return err
	}

	log.Println("Test data added successfully")
	return nil
}

func initCategories(db *sql.DB) error {
	categories := []struct {
		name        string
		description string
	}{
		{"Technology", "Discussions about latest tech trends"},
		{"Science", "Scientific discoveries and research"},
		{"Programming", "Programming languages and development"},
		{"Gaming", "Video games and gaming culture"},
		{"Movies", "Film discussions and reviews"},
	}

	for _, cat := range categories {
		_, err := db.Exec(`
			INSERT INTO categories (name, description)
			VALUES (?, ?)
			ON CONFLICT(name) DO NOTHING
		`, cat.name, cat.description)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	db, err := sql.Open("sqlite3", "./forum.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize database schema
	if err := initDB(db); err != nil {
		log.Fatal(err)
	}

	// Initialize default categories
	if err := initCategories(db); err != nil {
		log.Fatal(err)
	}

	// Check database
	checkDB(db)

	// Add test data
	if err := addTestData(db); err != nil {
		log.Printf("Error adding test data: %v", err)
	}

	// Create handlers
	h := handlers.NewHandler(db)

	// Setup routes
	http.HandleFunc("/", h.Home)
	http.HandleFunc("/categories", h.Categories)
	http.HandleFunc("/rules", h.Rules)
	http.HandleFunc("/register", h.SignUp)
	http.HandleFunc("/login", h.HandleLogin)
	http.HandleFunc("/logout", h.LogoutHandler)
	http.HandleFunc("/post/new", h.CreatePost)
	http.HandleFunc("/post/", h.GetPost)
	http.HandleFunc("/category/", h.GetPostsByCategory)
	http.HandleFunc("/api/react", h.ReactToPost)

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
