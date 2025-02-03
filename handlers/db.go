package handlers

import (
	"database/sql"
	"log"
	"os"
)

func InitDB(db *sql.DB) error {
	// Читаем SQL-скрипт
	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		return err
	}

	// Выполняем SQL-скрипт
	_, err = db.Exec(string(schema))
	if err != nil {
		return err
	}

	log.Println("Database initialized successfully")
	return nil
} 