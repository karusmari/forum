package handlers

import (
	"database/sql"
	"log"
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

	//initializing the categories
	if err := initCategories(db); err != nil {
		return err
	}

	log.Println("Database initialized successfully")
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