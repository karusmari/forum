package handlers

import (
	"database/sql"
	"os"
)

func InitDB()(*sql.DB, error) {
	// Open the database
	db, err := sql.Open("sqlite3", "./database/forum.db")
	if err != nil {
		return nil, err
}

	//read the schema from the file
	schema, err := os.ReadFile("database/schema.sql")
	if err != nil {
		return nil, err
	}

	//filling the database with the schema
	_, err = db.Exec(string(schema))
	if err != nil {
		return nil, err
	}
	return db, nil
}
