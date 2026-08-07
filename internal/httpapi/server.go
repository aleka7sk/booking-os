package httpapi

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aleka7sk/booking-os/internal/app"
)

//go:embed web/*
var embeddedWeb embed.FS

type Config struct {
	Addr          string
	SessionSecret string
	PublicURL     string
}

type Server struct {
	app        *app.Service
	config     Config
	logger     *slog.Logger
	web        fs.FS
	fileServer http.Handler
}

func New(service *app.Service, config Config, logger *slog.Logger) *Server {
	web, err := fs.Sub(embeddedWeb, "web")
	if err != nil {
		panic(err)
	}
	return &Server{app: service, config: config, logger: logger, web: web, fileServer: http.FileServer(http.FS(web))}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/auth/me", s.requireAuth(s.handleMe))

	mux.HandleFunc("GET /api/public/profile", s.handlePublicProfile)
	mux.HandleFunc("GET /api/public/offerings", s.handlePublicOfferings)
	mux.HandleFunc("POST /api/public/availability", s.handlePublicAvailability)
	mux.HandleFunc("POST /api/public/bookings", s.handlePublicCreateBooking)
	mux.HandleFunc("GET /api/public/bookings/{token}", s.handlePublicBooking)

	mux.HandleFunc("GET /api/dashboard", s.requireAuth(s.handleDashboard))
	mux.HandleFunc("GET /api/locations", s.requireAuth(s.handleLocations))
	mux.HandleFunc("GET /api/resources", s.requireAuth(s.handleResources))
	mux.HandleFunc("POST /api/resources", s.requireAuth(s.handleCreateResource))
	mux.HandleFunc("GET /api/offerings", s.requireAuth(s.handleOfferings))
	mux.HandleFunc("POST /api/offerings", s.requireAuth(s.handleCreateOffering))
	mux.HandleFunc("GET /api/bookings", s.requireAuth(s.handleBookings))
	mux.HandleFunc("POST /api/bookings", s.requireAuth(s.handleCreateBooking))
	mux.HandleFunc("GET /api/bookings/{id}", s.requireAuth(s.handleBooking))
	mux.HandleFunc("POST /api/bookings/{id}/confirm", s.requireAuth(s.handleConfirmBooking))
	mux.HandleFunc("POST /api/bookings/{id}/reject", s.requireAuth(s.handleRejectBooking))
	mux.HandleFunc("POST /api/bookings/{id}/payment", s.requireAuth(s.handlePayment))
	mux.HandleFunc("POST /api/bookings/{id}/cancel", s.requireAuth(s.handleCancelBooking))
	mux.HandleFunc("POST /api/bookings/{id}/reschedule", s.requireAuth(s.handleRescheduleBooking))
	mux.HandleFunc("POST /api/bookings/{id}/complete", s.requireAuth(s.handleCompleteBooking))
	mux.HandleFunc("POST /api/bookings/{id}/no-show", s.requireAuth(s.handleNoShowBooking))
	mux.HandleFunc("POST /api/availability/search", s.requireAuth(s.handleAvailability))
	mux.HandleFunc("GET /api/calendar", s.requireAuth(s.handleCalendar))
	mux.HandleFunc("GET /api/audit", s.requireAuth(s.handleAudit))

	mux.HandleFunc("/", s.serveWeb)
	return s.middleware(mux)
}

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
		if !strings.HasPrefix(r.URL.Path, "/api/health") {
			s.logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(started).String())
		}
	})
}

func (s *Server) serveWeb(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}
	path := strings.TrimPrefix(filepath.Clean(r.URL.Path), "/")
	if path != "." && path != "" && filepath.Ext(path) != "" {
		if _, err := fs.Stat(s.web, path); err == nil {
			if contentType := mime.TypeByExtension(filepath.Ext(path)); contentType != "" {
				w.Header().Set("Content-Type", contentType)
			}
			s.fileServer.ServeHTTP(w, r)
			return
		}
	}
	index, err := fs.ReadFile(s.web, "index.html")
	if err != nil {
		http.Error(w, "web application unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(index)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "booking-os", "time": time.Now().UTC()})
}

type requestError struct {
	code, message string
	cause         error
}

func (e *requestError) Error() string { return e.message }
func (e *requestError) Unwrap() error { return e.cause }

func writeError(w http.ResponseWriter, err error) {
	status, code, message := http.StatusInternalServerError, "internal_error", "Произошла внутренняя ошибка"
	var appErr *app.AppError
	var reqErr *requestError
	switch {
	case errors.As(err, &appErr):
		code, message = appErr.Code, appErr.Message
		switch {
		case errors.Is(appErr, app.ErrNotFound):
			status = http.StatusNotFound
		case errors.Is(appErr, app.ErrConflict):
			status = http.StatusConflict
		case errors.Is(appErr, app.ErrForbidden):
			status = http.StatusForbidden
		default:
			status = http.StatusBadRequest
		}
	case errors.As(err, &reqErr):
		status, code, message = http.StatusBadRequest, reqErr.code, reqErr.message
	}
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		fmt.Printf("encode response: %v\n", err)
	}
}
