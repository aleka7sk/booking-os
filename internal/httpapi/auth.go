package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aleka7sk/booking-os/internal/domain"
	"github.com/aleka7sk/booking-os/internal/security"
)

type contextKey string

const userContextKey contextKey = "booking-os-user"
const sessionCookieName = "booking_os_session"

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var input loginRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	user, err := s.app.FindUserByEmail(strings.TrimSpace(input.Email))
	if err != nil || !security.VerifyPassword(input.Password, user.PasswordSalt, user.PasswordHash) {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"code": "invalid_credentials", "message": "Неверный email или пароль"}})
		return
	}
	expires := time.Now().UTC().Add(14 * 24 * time.Hour)
	token := security.SignSession(s.config.SessionSecret, user.ID, expires)
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: token, Path: "/", Expires: expires, MaxAge: int((14 * 24 * time.Hour).Seconds()),
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: requestIsSecure(r),
	})
	writeJSON(w, http.StatusOK, map[string]any{"user": publicUser(user), "organization": s.app.Organization()})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: requestIsSecure(r)})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"code": "unauthorized", "message": "Требуется вход"}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": publicUser(user), "organization": s.app.Organization(), "locations": s.app.Locations()})
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"code": "unauthorized", "message": "Требуется вход"}})
			return
		}
		userID, err := security.VerifySession(s.config.SessionSecret, cookie.Value, time.Now().UTC())
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"code": "session_expired", "message": "Сессия истекла. Войдите снова"}})
			return
		}
		user, err := s.app.FindUserByID(userID)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"code": "unauthorized", "message": "Пользователь не найден"}})
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userContextKey, user)))
	}
}

func userFromContext(ctx context.Context) (domain.User, bool) {
	user, ok := ctx.Value(userContextKey).(domain.User)
	return user, ok
}

func publicUser(user domain.User) map[string]any {
	return map[string]any{"id": user.ID, "name": user.Name, "email": user.Email, "role": user.Role}
}

func requestIsSecure(r *http.Request) bool {
	return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return &requestError{code: "invalid_json", message: "Не удалось прочитать данные запроса", cause: err}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return &requestError{code: "invalid_json", message: "JSON-запрос должен содержать ровно один объект", cause: err}
	}
	return nil
}
