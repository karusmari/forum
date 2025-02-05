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
		data := TemplateData{
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
	var hashedPassword string
	err := h.db.QueryRow(`
		SELECT id, email, username, password_hash, is_admin 
		FROM users 
		WHERE email = ?
	`, email).Scan(&user.ID, &user.Email, &user.Username, &hashedPassword, &user.IsAdmin)

	//if the user is not found, then we will display an error message
	if err != nil {
		if err == sql.ErrNoRows {
			data := TemplateData{
				Title: "Login",
				Error: "Invalid email or password",
			}
			h.templates.ExecuteTemplate(w, "login.html", data)
			return
		}
		log.Printf("Database error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		data := TemplateData{
			Title: "Login",
			Error: "Invalid email or password",
		}
		h.templates.ExecuteTemplate(w, "login.html", data)
		return
	}

	sessionToken := uuid.New().String()
	var expiresAt time.Time
	
	if r.FormValue("remember_me") == "true" {
		expiresAt = time.Now().Add(RememberDuration)
		log.Printf("Creating long-term session (30 days) for user %d", user.ID)
	} else {
		expiresAt = time.Now().Add(SessionDuration)
		log.Printf("Creating standard session (24 hours) for user %d", user.ID)
	}

	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM sessions WHERE user_id = ?", user.ID)
	if err != nil {
		log.Printf("Error deleting old sessions: %v", err)
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(`
		INSERT INTO sessions (token, user_id, expires_at)
		VALUES (?, ?, ?)
	`, sessionToken, user.ID, expiresAt)

	if err != nil {
		log.Printf("Error creating session: %v", err)
		http.Error(w, "Session creation error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

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
		data := TemplateData{
			Title: "Sign Up",
		}
		h.templates.ExecuteTemplate(w, "register.html", data)
		return
	}

	//if the request is not POST, then we will display an error message
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем данные из формы
	email := r.FormValue("email")
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Проверяем, существует ли уже пользователь с таким email
	var exists bool
	err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", email).Scan(&exists)
	if err != nil {
		log.Printf("Error checking email existence: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if exists {
		data := TemplateData{
			Title: "Sign Up",
			Error: "This email address is already registered",
		}
		if err := h.templates.ExecuteTemplate(w, "register.html", data); err != nil {
			log.Printf("Template error: %v", err)
			http.Error(w, "Error rendering page", http.StatusInternalServerError)
		}
		return
	}

	// Проверяем, существует ли пользователь с таким username
	err = h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", username).Scan(&exists)
	if err != nil {
		log.Printf("Error checking username existence: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if exists {
		data := TemplateData{
			Title: "Sign Up",
			Error: "This username is already taken",
		}
		h.templates.ExecuteTemplate(w, "register.html", data)
		return
	}

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Создаем нового пользователя
	_, err = h.db.Exec(`
		INSERT INTO users (email, username, password_hash)
		VALUES (?, ?, ?)
	`, email, username, string(hashedPassword))

	if err != nil {
		log.Printf("Error creating user: %v", err)
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	// Перенаправляем на страницу входа
	http.Redirect(w, r, "/login", http.StatusSeeOther)
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
