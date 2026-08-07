package httpapi

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/aleka7sk/booking-os/internal/app"
	"github.com/aleka7sk/booking-os/internal/domain"
)

func (s *Server) handlePublicProfile(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"organization": s.app.Organization(), "locations": s.app.Locations()})
}

func (s *Server) handlePublicOfferings(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.app.Offerings(true))
}

func (s *Server) handlePublicAvailability(w http.ResponseWriter, r *http.Request) {
	var input app.AvailabilityInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	slots, err := s.app.SearchAvailability(input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, slots)
}

func (s *Server) handlePublicCreateBooking(w http.ResponseWriter, r *http.Request) {
	var input app.CreateBookingInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	input.Source = domain.SourcePublicPage
	booking, err := s.app.CreateBooking(input, "public", true)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, booking)
}

func (s *Server) handlePublicBooking(w http.ResponseWriter, r *http.Request) {
	booking, err := s.app.PublicBooking(r.PathValue("token"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, booking)
}

func (s *Server) handleDashboard(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.app.Dashboard())
}

func (s *Server) handleLocations(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.app.Locations())
}

func (s *Server) handleResources(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.app.Resources())
}

func (s *Server) handleCreateResource(w http.ResponseWriter, r *http.Request) {
	var input app.ResourceInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	user, _ := userFromContext(r.Context())
	resource, err := s.app.AddResource(input, user.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resource)
}

func (s *Server) handleOfferings(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.app.Offerings(false))
}

func (s *Server) handleCreateOffering(w http.ResponseWriter, r *http.Request) {
	var input app.OfferingInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	user, _ := userFromContext(r.Context())
	offering, err := s.app.AddOffering(input, user.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, offering)
}

func (s *Server) handleBookings(w http.ResponseWriter, r *http.Request) {
	var from, to *time.Time
	if raw := r.URL.Query().Get("from"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, &requestError{code: "invalid_from", message: "Некорректная начальная дата", cause: err})
			return
		}
		from = &parsed
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, &requestError{code: "invalid_to", message: "Некорректная конечная дата", cause: err})
			return
		}
		to = &parsed
	}
	status := domain.BookingStatus(strings.ToUpper(r.URL.Query().Get("status")))
	writeJSON(w, http.StatusOK, s.app.ListBookings(from, to, status))
}

func (s *Server) handleCreateBooking(w http.ResponseWriter, r *http.Request) {
	var input app.CreateBookingInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	user, _ := userFromContext(r.Context())
	booking, err := s.app.CreateBooking(input, user.ID, false)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, booking)
}

func (s *Server) handleBooking(w http.ResponseWriter, r *http.Request) {
	booking, err := s.app.Booking(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, booking)
}

func (s *Server) handleConfirmBooking(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	booking, err := s.app.ConfirmBooking(r.PathValue("id"), user.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, booking)
}

func (s *Server) handleRejectBooking(w http.ResponseWriter, r *http.Request) {
	var input app.CancelInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	user, _ := userFromContext(r.Context())
	booking, err := s.app.RejectBooking(r.PathValue("id"), user.ID, input.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, booking)
}

func (s *Server) handlePayment(w http.ResponseWriter, r *http.Request) {
	var input app.PaymentInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	user, _ := userFromContext(r.Context())
	booking, err := s.app.RecordPayment(r.PathValue("id"), user.ID, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, booking)
}

func (s *Server) handleCancelBooking(w http.ResponseWriter, r *http.Request) {
	var input app.CancelInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	user, _ := userFromContext(r.Context())
	booking, err := s.app.CancelBooking(r.PathValue("id"), user.ID, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, booking)
}

func (s *Server) handleRescheduleBooking(w http.ResponseWriter, r *http.Request) {
	var input app.RescheduleInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	user, _ := userFromContext(r.Context())
	booking, err := s.app.RescheduleBooking(r.PathValue("id"), user.ID, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, booking)
}

func (s *Server) handleCompleteBooking(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	booking, err := s.app.CompleteBooking(r.PathValue("id"), user.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, booking)
}

func (s *Server) handleNoShowBooking(w http.ResponseWriter, r *http.Request) {
	user, _ := userFromContext(r.Context())
	booking, err := s.app.NoShowBooking(r.PathValue("id"), user.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, booking)
}

func (s *Server) handleAvailability(w http.ResponseWriter, r *http.Request) {
	var input app.AvailabilityInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	slots, err := s.app.SearchAvailability(input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, slots)
}

func (s *Server) handleCalendar(w http.ResponseWriter, r *http.Request) {
	from, to, err := calendarRange(r)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.app.Calendar(from, to))
}

func calendarRange(r *http.Request) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	to := from.Add(7 * 24 * time.Hour)
	var err error
	if raw := r.URL.Query().Get("from"); raw != "" {
		from, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return time.Time{}, time.Time{}, &requestError{code: "invalid_from", message: "Некорректная начальная дата", cause: err}
		}
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		to, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return time.Time{}, time.Time{}, &requestError{code: "invalid_to", message: "Некорректная конечная дата", cause: err}
		}
	}
	if !to.After(from) || to.Sub(from) > 31*24*time.Hour {
		return time.Time{}, time.Time{}, &requestError{code: "invalid_range", message: "Диапазон календаря должен быть от 1 до 31 дня"}
	}
	return from, to, nil
}

func (s *Server) handleAudit(w http.ResponseWriter, _ *http.Request) {
	state := s.app.State()
	events := append([]domain.AuditEvent(nil), state.AuditEvents...)
	sort.Slice(events, func(i, j int) bool { return events[i].CreatedAt.After(events[j].CreatedAt) })
	if len(events) > 100 {
		events = events[:100]
	}
	writeJSON(w, http.StatusOK, events)
}
