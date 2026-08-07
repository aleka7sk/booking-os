package app

import (
	"errors"
	"fmt"
	"time"

	"github.com/aleka7sk/booking-os/internal/domain"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("booking conflict")
	ErrInvalid   = errors.New("invalid input")
	ErrForbidden = errors.New("forbidden")
)

type AppError struct {
	Kind    error
	Code    string
	Message string
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Kind }

func invalid(code, message string) error {
	return &AppError{Kind: ErrInvalid, Code: code, Message: message}
}

func conflict(code, message string) error {
	return &AppError{Kind: ErrConflict, Code: code, Message: message}
}

func notFound(entity string) error {
	return &AppError{Kind: ErrNotFound, Code: entity + "_not_found", Message: fmt.Sprintf("%s не найден", entity)}
}

type AvailabilityInput struct {
	OfferingID  string `json:"offeringId"`
	Date        string `json:"date"`
	GuestCount  int    `json:"guestCount"`
	DurationMin int    `json:"durationMin"`
}

type AvailabilitySlot struct {
	StartAt       time.Time          `json:"startAt"`
	EndAt         time.Time          `json:"endAt"`
	DurationMin   int                `json:"durationMin"`
	Available     bool               `json:"available"`
	ScarcityLabel string             `json:"scarcityLabel,omitempty"`
	TotalAmount   int64              `json:"totalAmount"`
	DepositAmount int64              `json:"depositAmount"`
	PriceLines    []domain.PriceLine `json:"priceLines"`
	ResourceIDs   []string           `json:"resourceIds,omitempty"`
}

type CreateBookingInput struct {
	OfferingID     string        `json:"offeringId"`
	CustomerName   string        `json:"customerName"`
	CustomerPhone  string        `json:"customerPhone"`
	CustomerEmail  string        `json:"customerEmail"`
	StartAt        time.Time     `json:"startAt"`
	DurationMin    int           `json:"durationMin"`
	GuestCount     int           `json:"guestCount"`
	Source         domain.Source `json:"source"`
	Notes          string        `json:"notes"`
	InternalNotes  string        `json:"internalNotes"`
	RequestOnly    bool          `json:"requestOnly"`
	PaidAmount     int64         `json:"paidAmount"`
	AcceptedPolicy bool          `json:"acceptedPolicy"`
}

type RescheduleInput struct {
	StartAt     time.Time `json:"startAt"`
	DurationMin int       `json:"durationMin"`
	Reason      string    `json:"reason"`
}

type CancelInput struct {
	Reason string `json:"reason"`
}

type PaymentInput struct {
	Amount int64  `json:"amount"`
	Note   string `json:"note"`
}

type ResourceInput struct {
	Name       string              `json:"name"`
	LocationID string              `json:"locationId"`
	Pool       string              `json:"pool"`
	Type       domain.ResourceType `json:"type"`
	Capacity   int                 `json:"capacity"`
	Color      string              `json:"color"`
	Notes      string              `json:"notes"`
}

type OfferingInput struct {
	Name                string                  `json:"name"`
	LocationID          string                  `json:"locationId"`
	Category            string                  `json:"category"`
	Description         string                  `json:"description"`
	SchedulingMode      domain.SchedulingMode   `json:"schedulingMode"`
	AllocationMode      domain.AllocationMode   `json:"allocationMode"`
	ConfirmationMode    domain.ConfirmationMode `json:"confirmationMode"`
	PriceMode           domain.PriceMode        `json:"priceMode"`
	BasePrice           int64                   `json:"basePrice"`
	DepositPercent      int                     `json:"depositPercent"`
	DurationMin         int                     `json:"durationMin"`
	MinDurationMin      int                     `json:"minDurationMin"`
	MaxDurationMin      int                     `json:"maxDurationMin"`
	DurationStepMin     int                     `json:"durationStepMin"`
	StartStepMin        int                     `json:"startStepMin"`
	BufferBeforeMin     int                     `json:"bufferBeforeMin"`
	BufferAfterMin      int                     `json:"bufferAfterMin"`
	MinGuests           int                     `json:"minGuests"`
	MaxGuests           int                     `json:"maxGuests"`
	ResourcePool        string                  `json:"resourcePool"`
	ResourcesRequired   int                     `json:"resourcesRequired"`
	CapacityPerResource int                     `json:"capacityPerResource"`
	MinLeadMin          int                     `json:"minLeadMin"`
	MaxAdvanceDays      int                     `json:"maxAdvanceDays"`
	HoldDurationMin     int                     `json:"holdDurationMin"`
	PublicEnabled       bool                    `json:"publicEnabled"`
	Accent              string                  `json:"accent"`
	CancellationPolicy  string                  `json:"cancellationPolicy"`
	FixedSessions       []domain.FixedSession   `json:"fixedSessions"`
}

type BookingView struct {
	Booking   domain.Booking    `json:"booking"`
	Offering  domain.Offering   `json:"offering"`
	Customer  domain.Customer   `json:"customer"`
	Resources []domain.Resource `json:"resources"`
}

type CalendarView struct {
	From      time.Time              `json:"from"`
	To        time.Time              `json:"to"`
	Resources []domain.Resource      `json:"resources"`
	Bookings  []BookingView          `json:"bookings"`
	Blocks    []domain.ResourceBlock `json:"blocks"`
}

type DashboardView struct {
	Date            string          `json:"date"`
	TodayBookings   int             `json:"todayBookings"`
	PendingRequests int             `json:"pendingRequests"`
	ActiveHolds     int             `json:"activeHolds"`
	TodayRevenue    int64           `json:"todayRevenue"`
	MonthRevenue    int64           `json:"monthRevenue"`
	Occupancy       int             `json:"occupancy"`
	Upcoming        []BookingView   `json:"upcoming"`
	WeeklyLoad      []DailyLoad     `json:"weeklyLoad"`
	Sources         []SourceMetric  `json:"sources"`
	ResourcePulse   []ResourcePulse `json:"resourcePulse"`
}

type DailyLoad struct {
	Date      string `json:"date"`
	Label     string `json:"label"`
	Bookings  int    `json:"bookings"`
	Occupancy int    `json:"occupancy"`
}

type SourceMetric struct {
	Source domain.Source `json:"source"`
	Count  int           `json:"count"`
}

type ResourcePulse struct {
	Resource domain.Resource `json:"resource"`
	State    string          `json:"state"`
	Until    *time.Time      `json:"until,omitempty"`
	Booking  *BookingView    `json:"booking,omitempty"`
}
