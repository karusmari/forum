package handlers

import (
	"database/sql"
	"os"
)

func InitDB(db *sql.DB) error {
	//read the schema from the file
	schema, err := os.ReadFile("database/schema.sql")
	if err != nil {
		return err
	}

	//filling the database with the schema
	_, err = db.Exec(string(schema))
	if err != nil {
		return err
	}

	//setting the timezone
	_, err = db.Exec("PRAGMA timezone = '+02:00'")
	if err != nil {
		return err
	}

	return nil
}
