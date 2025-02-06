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
	db, err := sql.Open("sqlite3", "./forum.db?_loc=auto")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := handlers.InitDB(db); err != nil {
		log.Fatal("Failed to initialize the database:", err)
	}

	// // Устанавливаем часовой пояс UTC+2
	// _, err = db.Exec("PRAGMA timezone = '+02:00'")
	// if err != nil {
	// 	log.Fatal("Error setting timezone:", err)
	// }

	// // Проверяем существование базы данных
	// if _, err := os.Stat("./forum.db"); os.IsNotExist(err) {
	// 	log.Println("Initializing new database...")

	// 	// Получаем текущую директорию
	// 	currentDir, err := os.Getwd()
	// 	if err != nil {
	// 		log.Fatal("Failed to get current directory:", err)
	// 	}
	// 	log.Printf("Current directory: %s", currentDir)

	// 	// Читаем SQL файл
	// 	sqlFile, err := os.ReadFile("./database/schema.sql")
	// 	if err != nil {
	// 		// Пробуем альтернативный путь
	// 		sqlFile, err = os.ReadFile("../database/schema.sql")
	// 		if err != nil {
	// 			log.Fatal("Failed to read schema.sql:", err)
	// 		}
	// 	}

	// 	// Выполняем SQL запросы
	// 	_, err = db.Exec(string(sqlFile))
	// 	if err != nil {
	// 		log.Printf("SQL Error: %v", err)
	// 		log.Fatal("Failed to initialize database:", err)
	// 	}

	// 	// Добавляем базовые категории
	// 	if err := initCategories(db); err != nil {
	// 		log.Fatal("Failed to initialize categories:", err)
	// 	}

	// 	log.Println("Database initialized successfully")
	// }

	// // Проверяем существование таблиц
	// tables := []string{"users", "posts", "categories", "comments", "reactions", "sessions"}
	// for _, table := range tables {
	// 	var count int
	// 	err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count)
	// 	if err != nil {
	// 		log.Fatal("Error checking tables:", err)
	// 	}
	// 	if count == 0 {
	// 		log.Printf("Table %s does not exist, reinitializing database", table)

	// 		// Пересоздаем базу данных
	// 		sqlFile, err := os.ReadFile("./database/schema.sql")
	// 		if err != nil {
	// 			sqlFile, err = os.ReadFile("../database/schema.sql")
	// 			if err != nil {
	// 				log.Fatal("Failed to read schema.sql:", err)
	// 			}
	// 		}

	// 		_, err = db.Exec(string(sqlFile))
	// 		if err != nil {
	// 			log.Fatal("Failed to reinitialize database:", err)
	// 		}

	// 		// Добавляем категории после пересоздания базы
	// 		if err := initCategories(db); err != nil {
	// 			log.Fatal("Failed to initialize categories after reinit:", err)
	// 		}

	// 		log.Println("Database reinitialized successfully")
	// 		break
	// 	}
	// }

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
	http.HandleFunc("/api/react", h.HandleReaction)
	http.HandleFunc("/api/comment", h.AddComment)
	http.HandleFunc("/api/comment/delete", h.DeleteComment)
	http.HandleFunc("/api/post/delete", h.DeletePost)
	http.HandleFunc("/post/edit/", h.EditPost)
	http.HandleFunc("/api/comment/react", h.HandleCommentReaction)
	http.HandleFunc("/api/comment/edit", h.EditComment)

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	fmt.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil)) // Logs the error and exits.
}
