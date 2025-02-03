package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// В начале файла добавим константы для сессий
const (
	SessionTokenCookie = "session_token"
	SessionDuration    = 24 * time.Hour
	RememberDuration   = 30 * 24 * time.Hour
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
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

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	var user User
	err := h.db.QueryRow(`
		SELECT id, email, username, password_hash, is_admin
		FROM users 
		WHERE email = ?
	`, email).Scan(&user.ID, &user.Email, &user.Username, &user.PasswordHash, &user.IsAdmin)

	if err != nil {
		data := &TemplateData{
			Title: "Login",
			Error: "Invalid email or password",
		}
		h.templates.ExecuteTemplate(w, "login.html", data)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		data := &TemplateData{
			Title: "Login",
			Error: "Invalid email or password",
		}
		h.templates.ExecuteTemplate(w, "login.html", data)
		return
	}

	// Создаем сессию
	sessionToken := uuid.New().String()
	var expiresAt time.Time
	
	// Проверяем флаг "Remember me"
	if r.FormValue("remember_me") == "true" {
		expiresAt = time.Now().Add(RememberDuration)
		log.Printf("Creating long-term session (30 days) for user %d", user.ID)
	} else {
		expiresAt = time.Now().Add(SessionDuration)
		log.Printf("Creating standard session (24 hours) for user %d", user.ID)
	}

	// Начинаем транзакцию
	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Удаляем старые сессии пользователя
	_, err = tx.Exec("DELETE FROM sessions WHERE user_id = ?", user.ID)
	if err != nil {
		log.Printf("Error deleting old sessions: %v", err)
		http.Error(w, "Session error", http.StatusInternalServerError)
		return
	}

	// Создаем новую сессию
	_, err = tx.Exec(`
		INSERT INTO sessions (token, user_id, expires_at)
		VALUES (?, ?, ?)
	`, sessionToken, user.ID, expiresAt)

	if err != nil {
		log.Printf("Error creating session: %v", err)
		http.Error(w, "Session creation error", http.StatusInternalServerError)
		return
	}

	// Завершаем транзакцию
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Устанавливаем cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionTokenCookie,
		Value:    sessionToken,
		Expires:  expiresAt,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})

	log.Printf("Successfully created session for user %d", user.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data := &TemplateData{
			Title: "Register",
		}
		h.templates.ExecuteTemplate(w, "register.html", data)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	// Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Password processing error", http.StatusInternalServerError)
		return
	}

	// Проверяем, есть ли уже пользователи в системе
	var userCount int
	err = h.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Если это первый пользователь, делаем его админом
	isAdmin := userCount == 0

	// Сохраняем пользователя в базу и получаем его ID
	result, err := h.db.Exec(`
		INSERT INTO users (email, username, password_hash, is_admin)
		VALUES (?, ?, ?, ?)
	`, email, username, string(hashedPassword), isAdmin)

	if err != nil {
		data := &TemplateData{
			Title: "Register",
			Error: "User already exists",
		}
		h.templates.ExecuteTemplate(w, "register.html", data)
		return
	}

	// Получаем ID нового пользователя
	userID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error getting new user ID: %v", err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// После успешной регистрации, если выбрана опция "Remember me",
	// сразу создаем сессию
	if r.FormValue("remember_me") == "true" {
		sessionToken := uuid.New().String()
		expiresAt := time.Now().Add(30 * 24 * time.Hour)

		_, err = h.db.Exec(`
			INSERT INTO sessions (token, user_id, expires_at)
			VALUES (?, ?, ?)
		`, sessionToken, userID, expiresAt)

		if err != nil {
			log.Printf("Error creating session: %v", err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    sessionToken,
			Expires:  expiresAt,
			HttpOnly: true,
			Path:     "/",
		})

		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionTokenCookie)
	if err != nil {
		log.Printf("No session cookie found: %v", err)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Удаляем сессию из базы данных
	_, err = h.db.Exec("DELETE FROM sessions WHERE token = ?", cookie.Value)
	if err != nil {
		log.Printf("Error deleting session: %v", err)
	}

	// Удаляем cookie на стороне клиента
	http.SetCookie(w, &http.Cookie{
		Name:     SessionTokenCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) GetSessionUser(r *http.Request) *User {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		log.Printf("No session cookie found: %v", err)
		return nil
	}

	var user User
	err = h.db.QueryRow(`
		SELECT u.id, u.email, u.username, u.is_admin 
		FROM users u
		JOIN sessions s ON u.id = s.user_id
		WHERE s.token = ? AND s.expires_at > CURRENT_TIMESTAMP
	`, cookie.Value).Scan(&user.ID, &user.Email, &user.Username, &user.IsAdmin)

	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("Error getting user from session: %v", err)
		}
		return nil
	}

	log.Printf("Found user in session: %s (ID: %d)", user.Username, user.ID)
	return &user
}
