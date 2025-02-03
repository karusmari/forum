package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)


const (
	SessionTokenCookie = "session_token" //cookie's name
	SessionDuration    = 24 * time.Hour //duration of session
	RememberDuration   = 30 * 24 * time.Hour //duration of long-term session
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	//if the request is GET(client enters URL), then we will display the login page
	if r.Method == http.MethodGet {
		data := &TemplateData{
			Title: "Login",
		}
		h.templates.ExecuteTemplate(w, "login.html", data)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//this will analyze the form data and parses it
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	//this will get the email and password from the form
	email := r.FormValue("email")
	password := r.FormValue("password")

	//this will get the user from the database
	var user User
	err := h.db.QueryRow(`
		SELECT id, email, username, password_hash, is_admin
		FROM users 
		WHERE email = ?
	`, email).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.IsAdmin)

	//if the user is not found, then we will display an error message
	if err != nil {
		data := &TemplateData{
			Title: "Login",
			Error: "Invalid email or password",
		}
		h.templates.ExecuteTemplate(w, "login.html", data)
		return
	}

	//this will compare the password from the form with the password from the database
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		data := &TemplateData{
			Title: "Login",
			Error: "Invalid email or password",
		}
		h.templates.ExecuteTemplate(w, "login.html", data)
		return
	}

	//creating a new session with unique token
	sessionToken := uuid.New().String()
	var expiresAt time.Time
	
	//if the user wants to remember the session, then we will create a long-term session
	if r.FormValue("remember_me") == "true" {
		expiresAt = time.Now().Add(RememberDuration)
		log.Printf("Creating long-term session (30 days) for user %d", user.ID)
	} else {
		expiresAt = time.Now().Add(SessionDuration)
		log.Printf("Creating standard session (24 hours) for user %d", user.ID)
	}

	//starting a transaction from the database. If there is an error, then we will display an error message
	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	//deleting the old sessions from the database
	_, err = tx.Exec("DELETE FROM sessions WHERE user_id = ?", user.ID)
	if err != nil {
		log.Printf("Error deleting old sessions: %v", err)
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	//new session will be inserted into the database
	_, err = tx.Exec(`
		INSERT INTO sessions (token, user_id, expires_at)
		VALUES (?, ?, ?)
	`, sessionToken, user.ID, expiresAt)

	if err != nil {
		log.Printf("Error creating session: %v", err)
		http.Error(w, "Session creation error", http.StatusInternalServerError)
		return
	}

	//committing the transaction to the database
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	//session will be saved in the cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionTokenCookie,
		Value:    sessionToken,
		Expires:  expiresAt,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	//redirecting the user to the home page after successful login
	log.Printf("Successfully created session for user %d", user.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	//if the request is GET(client enters URL), then we will display the register page
	if r.Method == http.MethodGet {
		data := &TemplateData{
			Title: "Register",
		}
		h.templates.ExecuteTemplate(w, "register.html", data)
		return
	}

	//if the request is not POST, then we will display an error message
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	//this will analyze the form data and parses it
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	//getting the username, email and password from the form
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	//hashing the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Password processing error", http.StatusInternalServerError)
		return
	}

	//checking if the user already exists in the database
	var userCount int
	err = h.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	//if the user is the first user, then we will make them an admin
	isAdmin := userCount == 0

	// inserts the new user into the database
	result, err := h.db.Exec(`
		INSERT INTO users (email, username, password_hash, is_admin)
		VALUES (?, ?, ?, ?)
	`, email, username, string(hashedPassword), isAdmin)

	//if the user already exists, then we will display an error message
	if err != nil {
		data := &TemplateData{
			Title: "Register",
			Error: "User already exists",
		}
		h.templates.ExecuteTemplate(w, "register.html", data)
		return
	}

	//getting the new user ID
	userID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error getting new user ID: %v", err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	//if the user checked the "remember me" checkbox, then we will create a long-term session
	if r.FormValue("remember_me") == "true" {
		//generating a unique token for the session
		sessionToken := uuid.New().String()
		expiresAt := time.Now().Add(30 * 24 * time.Hour)

		//inserting the session into the sessions table, linking it to the user and setting the expiration time
		_, err = h.db.Exec(`
			INSERT INTO sessions (token, user_id, expires_at)
			VALUES (?, ?, ?)
		`, sessionToken, userID, expiresAt)

		//if there is an error, then we will display an error message and redirect the user to the login page
		if err != nil {
			log.Printf("Error creating session: %v", err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		//setting the cookie with the session token in the user's browser with the expiration time
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			Expires:  expiresAt,
			HttpOnly: true,
			Path:     "/",
		})
		//redirecting the user to the home page after a successful registration
		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		//if the user did not check the "remember me" checkbox, then we will redirect the user to the login page
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	//tries to find the session cookie, if not found, then we will redirect the user to the home page
	cookie, err := r.Cookie(SessionTokenCookie)
	if err != nil {
		log.Printf("No session cookie found: %v", err)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	//deleting the session associated with the token from the database
	_, err = h.db.Exec("DELETE FROM sessions WHERE token = ?", cookie.Value)
	if err != nil {
		log.Printf("Error deleting session: %v", err)
	}

	//deleting the session cookie from the user's browser
	http.SetCookie(w, &http.Cookie{
		Name:     SessionTokenCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1, //-1 because we want to delete the cookie immediately, cookie is considered expired
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	//after logging out, the user will be redirected to the home page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

//the purpose of this function is to find the user from the session
func (h *Handler) GetSessionUser(r *http.Request) *User {
	//tries to find the session cookie, if not found, then we will return nil
	cookie, err := r.Cookie("session_token")
	if err != nil {
		log.Printf("No session cookie found: %v", err)
		return nil
	}

	var user User //creating a new object of the User struct
	//SQL query to get the user from the session
	err = h.db.QueryRow(` 
		SELECT u.id, u.email, u.username, u.is_admin 
		FROM users u
		JOIN sessions s ON u.id = s.user_id
		WHERE s.token = ? AND s.expires_at > CURRENT_TIMESTAMP
	`, cookie.Value).Scan(&user.ID, &user.Email, &user.Username, &user.IsAdmin)
	//if the scan was successful, then we will fill the user object with the data

	if err != nil {
		//if we don't find the user, then we will return nil
		if err != sql.ErrNoRows {
			log.Printf("Error getting user from session: %v", err)
		}
		return nil
	}

	//if the user is found, then we will log the user's information
	log.Printf("Found user in session: %s (ID: %d)", user.Username, user.ID)
	return &user
}
