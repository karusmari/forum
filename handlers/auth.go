package handlers

import (
	"database/sql"
	"net/http"
	"time"
	"github.com/gofrs/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	SessionTokenCookie = "session_token"     //cookie's name
	SessionDuration    = 24 * time.Hour      //duration of session
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
		h.ErrorHandler(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	//this will analyze the form data and parses it
	if err := r.ParseForm(); err != nil {
		h.ErrorHandler(w, "Failed to parse form", http.StatusBadRequest)
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
		h.ErrorHandler(w, "Database error", http.StatusInternalServerError)
		return
	}
	//this will compare the password from the form with the password from the database
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		data := TemplateData{
			Title: "Login",
			Error: "Invalid email or password",
		}
		h.templates.ExecuteTemplate(w, "login.html", data)
		return
	}
	//creating a new session with unique token
	sessionUUID, err := uuid.NewV4() // Generate a new UUID
	if err != nil {
		return
	}

	sessionToken := sessionUUID.String()

	var expiresAt time.Time

	//if the user wants to remember the session, then we will create a long-term session
	if r.FormValue("remember_me") == "true" {
		expiresAt = time.Now().Add(RememberDuration)
	} else {
		expiresAt = time.Now().Add(SessionDuration)
	}

	//starting a transaction from the database. If there is an error, then we will display an error message
	tx, err := h.db.Begin()
	if err != nil {
		h.ErrorHandler(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	//deleting the old sessions from the database
	_, err = tx.Exec("DELETE FROM sessions WHERE user_id = ?", user.ID)
	if err != nil {
		h.ErrorHandler(w, "Session error", http.StatusInternalServerError)
		return
	}

	//new session will be inserted into the database
	_, err = tx.Exec(`
		INSERT INTO sessions (token, user_id, expires_at)
		VALUES (?, ?, ?)
	`, sessionToken, user.ID, expiresAt)

	if err != nil {
		h.ErrorHandler(w, "Session creation error", http.StatusInternalServerError)
		return
	}

	//committing the transaction to the database
	if err := tx.Commit(); err != nil {
		h.ErrorHandler(w, "Database error", http.StatusInternalServerError)
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
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	// If request is GET (client enters URL), display the register page
	if r.Method == http.MethodGet {
		data := TemplateData{
			Title: "Sign Up",
		}
		h.templates.ExecuteTemplate(w, "register.html", data)
		return
	}

	// If request is not POST, display an error message
	if r.Method != http.MethodPost {
		h.ErrorHandler(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get username, email and password from the form
	email := r.FormValue("email")
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Check if user exists with this email
	var exists bool
	err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", email).Scan(&exists)
	if err != nil {
		h.ErrorHandler(w, "Database error", http.StatusInternalServerError)
		return
	}

	if exists {
		data := TemplateData{
			Title: "Sign Up",
			Error: "This email address is already registered",
		}
		if err := h.templates.ExecuteTemplate(w, "register.html", data); err != nil {
			h.ErrorHandler(w, "Error rendering page", http.StatusInternalServerError)
		}
		return
	}

	// Check if username is taken
	err = h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", username).Scan(&exists)
	if err != nil {
		h.ErrorHandler(w, "Database error", http.StatusInternalServerError)
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

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		h.ErrorHandler(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create new user
	_, err = h.db.Exec(`
		INSERT INTO users (email, username, password_hash)
		VALUES (?, ?, ?)
	`, email, username, string(hashedPassword))

	if err != nil {
		h.ErrorHandler(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	// Redirect to login page
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	//tries to find the session cookie, if not found, then we will redirect the user to the home page
	cookie, err := r.Cookie(SessionTokenCookie)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	//deleting the session associated with the token from the database
	_, err = h.db.Exec("DELETE FROM sessions WHERE token = ?", cookie.Value)
	if err != nil {
		h.ErrorHandler(w, "Error deleting session", http.StatusInternalServerError)
		return
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

// the purpose of this function is to find the user from the session
func (h *Handler) GetSessionUser(r *http.Request) *User {
	//tries to find the session cookie, if not found, then we will return nil
	cookie, err := r.Cookie("session_token")
	if err != nil {
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
			return nil
		}
	}
	//if the user is found, then we will log the user's information
	return &user
}