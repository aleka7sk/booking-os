package app

import (
	"errors"
	"testing"
	"time"

	"github.com/aleka7sk/booking-os/internal/domain"
)

func requireAppErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError %q, got %v", code, err)
	}
	if appErr.Code != code {
		t.Fatalf("expected error code %q, got %q", code, appErr.Code)
	}
}

func TestPublicBookingRequiresAndRecordsPolicyAcceptance(t *testing.T) {
	service := newTestService(t, []domain.Resource{lane("lane-only")}, []domain.Offering{bowlingOffering(domain.ConfirmationManagerApproval)})
	start := time.Date(2026, time.August, 8, 18, 0, 0, 0, time.UTC)
	input := CreateBookingInput{
		OfferingID: "off-bowling", CustomerName: "Policy Client", CustomerPhone: "+77010000001",
		StartAt: start, DurationMin: 120, GuestCount: 2,
	}

	_, err := service.CreateBooking(input, "public", true)
	requireAppErrorCode(t, err, "policy_not_accepted")

	input.AcceptedPolicy = true
	created, err := service.CreateBooking(input, "public", true)
	if err != nil {
		t.Fatalf("create accepted public booking: %v", err)
	}
	if created.Booking.PolicyAcceptedAt == nil {
		t.Fatal("public booking must record policy acceptance time")
	}
	if created.Booking.CancellationSnapshot != "24 часа" {
		t.Fatalf("unexpected cancellation snapshot: %q", created.Booking.CancellationSnapshot)
	}
}

func TestCreateFixedSessionOfferingAndFindConfiguredSlot(t *testing.T) {
	hall := domain.Resource{
		ID: "hall", LocationID: "loc-test", Name: "Workshop hall", Pool: "WORKSHOP",
		Type: domain.ResourceSpace, Capacity: 12, Active: true, Color: "#7868F2",
	}
	service := newTestService(t, []domain.Resource{hall}, nil)

	offering, err := service.AddOffering(OfferingInput{
		Name: "Ceramics", LocationID: "loc-test", ResourcePool: "WORKSHOP",
		SchedulingMode: domain.SchedulingFixed, AllocationMode: domain.AllocationShared,
		ConfirmationMode: domain.ConfirmationPaymentGated, PriceMode: domain.PricePerPerson,
		BasePrice: 9000, DepositPercent: 100, DurationMin: 90, MinDurationMin: 90,
		MaxDurationMin: 90, DurationStepMin: 90, StartStepMin: 30,
		MinGuests: 1, MaxGuests: 12, ResourcesRequired: 1, CapacityPerResource: 1,
		MaxAdvanceDays: 30, HoldDurationMin: 15, PublicEnabled: true,
		FixedSessions: []domain.FixedSession{
			{Weekday: int(time.Saturday), Start: "15:00"},
			{Weekday: int(time.Tuesday), Start: "19:00"},
			{Weekday: int(time.Saturday), Start: "15:00"}, // duplicate must be removed
		},
	}, "owner")
	if err != nil {
		t.Fatalf("add fixed offering: %v", err)
	}
	if len(offering.FixedSessions) != 2 {
		t.Fatalf("expected two normalized sessions, got %d", len(offering.FixedSessions))
	}

	slots, err := service.SearchAvailability(AvailabilityInput{OfferingID: offering.ID, Date: "2026-08-08", GuestCount: 3, DurationMin: 90})
	if err != nil {
		t.Fatalf("search fixed sessions: %v", err)
	}
	if len(slots) != 1 || slots[0].StartAt.Hour() != 15 || slots[0].StartAt.Minute() != 0 {
		t.Fatalf("expected Saturday 15:00 slot, got %+v", slots)
	}

	noSlots, err := service.SearchAvailability(AvailabilityInput{OfferingID: offering.ID, Date: "2026-08-09", GuestCount: 3, DurationMin: 90})
	if err != nil {
		t.Fatalf("search non-session day: %v", err)
	}
	if len(noSlots) != 0 {
		t.Fatalf("expected no Sunday slots, got %d", len(noSlots))
	}
}

func TestOfferingConfigurationRejectsBrokenWorkflows(t *testing.T) {
	hall := domain.Resource{ID: "hall", LocationID: "loc-test", Name: "Hall", Pool: "WORKSHOP", Type: domain.ResourceSpace, Capacity: 12, Active: true}
	service := newTestService(t, []domain.Resource{hall}, nil)
	base := OfferingInput{
		Name: "Workshop", LocationID: "loc-test", ResourcePool: "WORKSHOP",
		AllocationMode: domain.AllocationShared, PriceMode: domain.PricePerPerson,
		BasePrice: 9000, DurationMin: 90, MinDurationMin: 90, MaxDurationMin: 90,
		DurationStepMin: 90, StartStepMin: 30, MinGuests: 1, MaxGuests: 12,
		ResourcesRequired: 1, CapacityPerResource: 1, MaxAdvanceDays: 30,
	}

	missingSessions := base
	missingSessions.SchedulingMode = domain.SchedulingFixed
	missingSessions.ConfirmationMode = domain.ConfirmationManagerApproval
	_, err := service.AddOffering(missingSessions, "owner")
	requireAppErrorCode(t, err, "fixed_sessions_required")

	paymentWithoutDeposit := base
	paymentWithoutDeposit.SchedulingMode = domain.SchedulingRental
	paymentWithoutDeposit.ConfirmationMode = domain.ConfirmationPaymentGated
	_, err = service.AddOffering(paymentWithoutDeposit, "owner")
	requireAppErrorCode(t, err, "deposit_required")

	requestWindow := base
	requestWindow.SchedulingMode = domain.SchedulingRequest
	requestWindow.ConfirmationMode = domain.ConfirmationManagerApproval
	_, err = service.AddOffering(requestWindow, "owner")
	requireAppErrorCode(t, err, "request_window_not_available")
}

func TestResourceConfigurationRejectsUnsafeValues(t *testing.T) {
	service := newTestService(t, nil, nil)
	_, err := service.AddResource(ResourceInput{Name: "Bad", LocationID: "loc-test", Pool: "X", Type: "SCRIPT", Capacity: 1, Color: "#7868F2"}, "owner")
	requireAppErrorCode(t, err, "resource_type_invalid")

	_, err = service.AddResource(ResourceInput{Name: "Bad", LocationID: "loc-test", Pool: "X", Type: domain.ResourceAsset, Capacity: 1, Color: "red;display:none"}, "owner")
	requireAppErrorCode(t, err, "resource_color_invalid")
}
