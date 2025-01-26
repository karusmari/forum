package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)

	_, err = h.db.Exec(`
		INSERT INTO sessions (id, user_id, expires_at)
		VALUES (?, ?, ?)
	`, sessionID, user.ID, expiresAt)

	if err != nil {
		http.Error(w, "Session creation error", http.StatusInternalServerError)
		return
	}

	// Устанавливаем cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    sessionID,
		Expires:  expiresAt,
		HttpOnly: true,
		Path:     "/",
	})

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

	// Сохраняем пользователя в базу
	_, err = h.db.Exec(`
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

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}

	// Удаляем сессию из базы
	_, err = h.db.Exec("DELETE FROM sessions WHERE id = ?", cookie.Value)
	if err != nil {
		http.Error(w, "Ошибка при выходе", http.StatusInternalServerError)
		return
	}

	// Удаляем cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetSessionUser(r *http.Request) *User {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil
	}

	var user User
	err = h.db.QueryRow(`
		SELECT u.id, u.email, u.username, u.is_admin
		FROM users u
		JOIN sessions s ON s.user_id = u.id
		WHERE s.id = ? AND s.expires_at > ?
	`, cookie.Value, time.Now()).Scan(&user.ID, &user.Email, &user.Username, &user.IsAdmin)

	if err != nil {
		return nil
	}

	return &user
}
