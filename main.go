package main

import (
	"database/sql"
	"fmt"
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
		"Moving to Åland",
		"Living in Åland",
		"Housing in Åland",
		"Studying in Åland",
		"Jobs and entrepreneurship in Åland",
		"Family life in Åland",
		"Culture and leisure in Åland",
		"For sale and wanted in Åland",
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
		{"Moving to Åland", "Get insights and practical tips on relocating to Åland, including shipping, immigration procedures, residence permits, visas, and firsthand experiences."},
		{"Living in Åland", "Explore all aspects of life in Åland, from finding local services to understanding daily living, including essential advice for settling in and getting connected."},
		{"Housing in Åland", "Guidance on finding housing in Åland, whether you're renting, buying, or selling. Includes advice on neighborhoods, utilities, tenant rights, and housing trends."},
		{"Studying in Åland", "Discover everything you need to know about studying in Åland, from admissions to universities, student visas, courses, and tips for a successful student life."},
		{"Jobs and entrepreneurship in Åland", "Helpful information about job opportunities, career advice, work permits, and entrepreneurship in Åland. Find job openings, work culture, and resources for business owners."},
		{"Family life in Åland", "Support and resources for families in Åland, including schooling, childcare, language education, and family-friendly activities. Connect with other families and share experiences."},
		{"Culture and leisure in Åland", "Explore cultural events, leisure activities, and entertainment options in Åland. Find out about local festivals, museums, outdoor activities, and ways to enjoy your free time."},
		{"For sale and wanted in Åland", "Browse listings for items for sale, trades, and wanted ads in Åland. Whether you're buying or selling, find a variety of goods and services available in the local community."},
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
	// Добавляем параметр _loc=auto для использования локального времени
	db, err := sql.Open("sqlite3", "./forum.db?_loc=auto")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Устанавливаем часовой пояс UTC+2
	_, err = db.Exec("PRAGMA timezone = '+02:00'")
	if err != nil {
		log.Fatal("Error setting timezone:", err)
	}

	// Проверяем существование базы данных
	if _, err := os.Stat("./forum.db"); os.IsNotExist(err) {
		log.Println("Initializing new database...")

		// Получаем текущую директорию
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatal("Failed to get current directory:", err)
		}
		log.Printf("Current directory: %s", currentDir)

		// Читаем SQL файл
		sqlFile, err := os.ReadFile("./database/schema.sql")
		if err != nil {
			// Пробуем альтернативный путь
			sqlFile, err = os.ReadFile("../database/schema.sql")
			if err != nil {
				log.Fatal("Failed to read schema.sql:", err)
			}
		}

		// Выполняем SQL запросы
		_, err = db.Exec(string(sqlFile))
		if err != nil {
			log.Printf("SQL Error: %v", err)
			log.Fatal("Failed to initialize database:", err)
		}

		// Добавляем базовые категории
		if err := initCategories(db); err != nil {
			log.Fatal("Failed to initialize categories:", err)
		}

		log.Println("Database initialized successfully")
	}

	// Проверяем существование таблиц
	tables := []string{"users", "posts", "categories", "comments", "reactions", "sessions"}
	for _, table := range tables {
		var count int
		err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count)
		if err != nil {
			log.Fatal("Error checking tables:", err)
		}
		if count == 0 {
			log.Printf("Table %s does not exist, reinitializing database", table)

			// Пересоздаем базу данных
			sqlFile, err := os.ReadFile("./database/schema.sql")
			if err != nil {
				sqlFile, err = os.ReadFile("../database/schema.sql")
				if err != nil {
					log.Fatal("Failed to read schema.sql:", err)
				}
			}

			_, err = db.Exec(string(sqlFile))
			if err != nil {
				log.Fatal("Failed to reinitialize database:", err)
			}

			// Добавляем категории после пересоздания базы
			if err := initCategories(db); err != nil {
				log.Fatal("Failed to initialize categories after reinit:", err)
			}

			log.Println("Database reinitialized successfully")
			break
		}
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
	http.HandleFunc("/api/react", h.ReactToPost)
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
