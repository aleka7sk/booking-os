package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/aleka7sk/booking-os/internal/domain"
	"github.com/aleka7sk/booking-os/internal/store"
)

var testNow = time.Date(2026, time.August, 7, 10, 0, 0, 0, time.UTC)

func newTestService(t *testing.T, resources []domain.Resource, offerings []domain.Offering) *Service {
	t.Helper()
	weekly := map[string][]domain.Hours{}
	for _, day := range []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"} {
		weekly[day] = []domain.Hours{{Start: "08:00", End: "23:00"}}
	}
	st, err := store.Open(filepath.Join(t.TempDir(), "state.json"), func() (domain.State, error) {
		return domain.State{
			SchemaVersion: 1,
			Organization:  domain.Organization{ID: "org-test", Name: "Test", Currency: "KZT", Timezone: "UTC"},
			Locations:     []domain.Location{{ID: "loc-test", Name: "Test location", Timezone: "UTC", Active: true, WeeklyHours: weekly}},
			Resources:     resources,
			Offerings:     offerings,
			NextReference: 1,
		}, nil
	})
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	service := New(st)
	service.now = func() time.Time { return testNow }
	return service
}

func lane(id string) domain.Resource {
	return domain.Resource{ID: id, LocationID: "loc-test", Name: id, Pool: "BOWLING", Type: domain.ResourceAsset, Capacity: 1, Active: true, CreatedAt: testNow, UpdatedAt: testNow}
}

func bowlingOffering(confirmation domain.ConfirmationMode) domain.Offering {
	return domain.Offering{
		ID: "off-bowling", LocationID: "loc-test", Name: "Боулинг", SchedulingMode: domain.SchedulingRental,
		AllocationMode: domain.AllocationExclusive, ConfirmationMode: confirmation, PriceMode: domain.PricePerResourceHour,
		BasePrice: 7000, DepositPercent: 30, DurationMin: 120, MinDurationMin: 60, MaxDurationMin: 240,
		DurationStepMin: 60, StartStepMin: 30, MinGuests: 1, MaxGuests: 24, ResourcePool: "BOWLING",
		ResourcesRequired: 1, CapacityPerResource: 6, MaxAdvanceDays: 60, HoldDurationMin: 15,
		PublicEnabled: true, Active: true, CancellationPolicy: "24 часа", CreatedAt: testNow, UpdatedAt: testNow,
	}
}

func TestAvailabilityPlansTwoLanesAndPriceSnapshot(t *testing.T) {
	service := newTestService(t,
		[]domain.Resource{lane("lane-1"), lane("lane-2"), lane("lane-3"), lane("lane-4")},
		[]domain.Offering{bowlingOffering(domain.ConfirmationApprovalAndPayment)},
	)

	slots, err := service.SearchAvailability(AvailabilityInput{OfferingID: "off-bowling", Date: "2026-08-08", GuestCount: 8, DurationMin: 120})
	if err != nil {
		t.Fatalf("search availability: %v", err)
	}
	if len(slots) == 0 {
		t.Fatal("expected available slots")
	}
	got := slots[0]
	if len(got.ResourceIDs) != 2 {
		t.Fatalf("expected 2 lanes, got %d", len(got.ResourceIDs))
	}
	if got.TotalAmount != 28000 || got.DepositAmount != 8400 {
		t.Fatalf("unexpected quote total=%d deposit=%d", got.TotalAmount, got.DepositAmount)
	}
}

func TestRequestApprovalDepositFlow(t *testing.T) {
	service := newTestService(t,
		[]domain.Resource{lane("lane-1"), lane("lane-2")},
		[]domain.Offering{bowlingOffering(domain.ConfirmationApprovalAndPayment)},
	)
	start := time.Date(2026, time.August, 8, 18, 0, 0, 0, time.UTC)

	created, err := service.CreateBooking(CreateBookingInput{
		OfferingID: "off-bowling", CustomerName: "Алия", CustomerPhone: "+7 701 000 00 01",
		StartAt: start, DurationMin: 120, GuestCount: 8, Source: domain.SourcePublicPage, AcceptedPolicy: true,
	}, "public", true)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if created.Booking.Status != domain.BookingRequested {
		t.Fatalf("expected REQUESTED, got %s", created.Booking.Status)
	}
	if len(created.Resources) != 0 {
		t.Fatalf("requested booking must not occupy resources, got %d", len(created.Resources))
	}

	approved, err := service.ConfirmBooking(created.Booking.ID, "manager-1")
	if err != nil {
		t.Fatalf("confirm booking: %v", err)
	}
	if approved.Booking.Status != domain.BookingHeld || approved.Booking.HoldExpiresAt == nil {
		t.Fatalf("expected HELD with expiry, got %s", approved.Booking.Status)
	}
	if len(approved.Resources) != 2 {
		t.Fatalf("expected 2 allocated lanes, got %d", len(approved.Resources))
	}

	paid, err := service.RecordPayment(created.Booking.ID, "manager-1", PaymentInput{Amount: approved.Booking.DepositAmount})
	if err != nil {
		t.Fatalf("record deposit: %v", err)
	}
	if paid.Booking.Status != domain.BookingConfirmed || paid.Booking.PaymentStatus != domain.PaymentPartiallyPaid {
		t.Fatalf("expected confirmed partially paid booking, got status=%s payment=%s", paid.Booking.Status, paid.Booking.PaymentStatus)
	}
	if paid.Booking.HoldExpiresAt != nil {
		t.Fatal("confirmed booking must not retain hold expiry")
	}
}

func TestSharedCapacityNeverOverbooks(t *testing.T) {
	resource := domain.Resource{ID: "hall", LocationID: "loc-test", Name: "Hall", Pool: "WORKSHOP", Type: domain.ResourceSpace, Capacity: 12, Active: true}
	offering := domain.Offering{
		ID: "off-workshop", LocationID: "loc-test", Name: "Workshop", SchedulingMode: domain.SchedulingRental,
		AllocationMode: domain.AllocationShared, ConfirmationMode: domain.ConfirmationPaymentGated, PriceMode: domain.PricePerPerson,
		BasePrice: 9000, DepositPercent: 100, DurationMin: 90, MinDurationMin: 90, MaxDurationMin: 90,
		DurationStepMin: 90, StartStepMin: 30, MinGuests: 1, MaxGuests: 12, ResourcePool: "WORKSHOP",
		ResourcesRequired: 1, MaxAdvanceDays: 60, HoldDurationMin: 15, PublicEnabled: true, Active: true,
	}
	service := newTestService(t, []domain.Resource{resource}, []domain.Offering{offering})
	start := time.Date(2026, time.August, 8, 15, 0, 0, 0, time.UTC)

	first, err := service.CreateBooking(CreateBookingInput{OfferingID: offering.ID, CustomerName: "First", CustomerPhone: "+77010000001", StartAt: start, DurationMin: 90, GuestCount: 9, AcceptedPolicy: true}, "public", true)
	if err != nil {
		t.Fatalf("create first booking: %v", err)
	}
	if first.Booking.Status != domain.BookingHeld {
		t.Fatalf("expected held first booking, got %s", first.Booking.Status)
	}

	_, err = service.CreateBooking(CreateBookingInput{OfferingID: offering.ID, CustomerName: "Second", CustomerPhone: "+77010000002", StartAt: start, DurationMin: 90, GuestCount: 4, AcceptedPolicy: true}, "public", true)
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected capacity conflict, got %v", err)
	}

	third, err := service.CreateBooking(CreateBookingInput{OfferingID: offering.ID, CustomerName: "Third", CustomerPhone: "+77010000003", StartAt: start, DurationMin: 90, GuestCount: 3, AcceptedPolicy: true}, "public", true)
	if err != nil {
		t.Fatalf("create remaining capacity: %v", err)
	}
	if len(third.Resources) != 1 {
		t.Fatalf("expected one shared resource, got %d", len(third.Resources))
	}
}

func TestConcurrentLastResourceHasSingleWinner(t *testing.T) {
	service := newTestService(t, []domain.Resource{lane("lane-only")}, []domain.Offering{bowlingOffering(domain.ConfirmationInstant)})
	start := time.Date(2026, time.August, 8, 20, 0, 0, 0, time.UTC)

	const contenders = 20
	var wg sync.WaitGroup
	results := make(chan error, contenders)
	for i := 0; i < contenders; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := service.CreateBooking(CreateBookingInput{
				OfferingID: "off-bowling", CustomerName: fmt.Sprintf("Client %d", index),
				CustomerPhone: fmt.Sprintf("+7701000%04d", index), StartAt: start, DurationMin: 120, GuestCount: 2,
			}, "manager", false)
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)

	successes, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent result: %v", err)
		}
	}
	if successes != 1 || conflicts != contenders-1 {
		t.Fatalf("expected one winner and %d conflicts, got successes=%d conflicts=%d", contenders-1, successes, conflicts)
	}
}

func TestFailedRescheduleKeepsOriginalAllocation(t *testing.T) {
	service := newTestService(t, []domain.Resource{lane("lane-only")}, []domain.Offering{bowlingOffering(domain.ConfirmationInstant)})
	firstStart := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	secondStart := time.Date(2026, time.August, 8, 16, 0, 0, 0, time.UTC)

	first, err := service.CreateBooking(CreateBookingInput{OfferingID: "off-bowling", CustomerName: "First", CustomerPhone: "+77010000001", StartAt: firstStart, DurationMin: 120, GuestCount: 2}, "manager", false)
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	_, err = service.CreateBooking(CreateBookingInput{OfferingID: "off-bowling", CustomerName: "Second", CustomerPhone: "+77010000002", StartAt: secondStart, DurationMin: 120, GuestCount: 2}, "manager", false)
	if err != nil {
		t.Fatalf("create second: %v", err)
	}

	_, err = service.RescheduleBooking(first.Booking.ID, "manager", RescheduleInput{StartAt: secondStart, DurationMin: 120, Reason: "conflict test"})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected reschedule conflict, got %v", err)
	}
	unchanged, err := service.Booking(first.Booking.ID)
	if err != nil {
		t.Fatalf("reload first: %v", err)
	}
	if !unchanged.Booking.StartAt.Equal(firstStart) || len(unchanged.Resources) != 1 {
		t.Fatalf("original booking was damaged: start=%s resources=%d", unchanged.Booking.StartAt, len(unchanged.Resources))
	}
}

func TestExpiredHoldReleasesResource(t *testing.T) {
	service := newTestService(t, []domain.Resource{lane("lane-only")}, []domain.Offering{bowlingOffering(domain.ConfirmationPaymentGated)})
	start := time.Date(2026, time.August, 8, 20, 0, 0, 0, time.UTC)
	created, err := service.CreateBooking(CreateBookingInput{OfferingID: "off-bowling", CustomerName: "First", CustomerPhone: "+77010000001", StartAt: start, DurationMin: 120, GuestCount: 2, AcceptedPolicy: true}, "public", true)
	if err != nil {
		t.Fatalf("create held booking: %v", err)
	}
	if created.Booking.Status != domain.BookingHeld {
		t.Fatalf("expected HELD, got %s", created.Booking.Status)
	}

	service.now = func() time.Time { return testNow.Add(16 * time.Minute) }
	count, err := service.ExpireHolds()
	if err != nil {
		t.Fatalf("expire holds: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one expired hold, got %d", count)
	}

	second, err := service.CreateBooking(CreateBookingInput{OfferingID: "off-bowling", CustomerName: "Second", CustomerPhone: "+77010000002", StartAt: start, DurationMin: 120, GuestCount: 2}, "manager", false)
	if err != nil {
		t.Fatalf("resource was not released: %v", err)
	}
	if second.Booking.Status != domain.BookingConfirmed {
		t.Fatalf("expected confirmed replacement, got %s", second.Booking.Status)
	}
}

func TestLatePaymentCannotReviveExpiredHold(t *testing.T) {
	service := newTestService(t, []domain.Resource{lane("lane-only")}, []domain.Offering{bowlingOffering(domain.ConfirmationPaymentGated)})
	start := time.Date(2026, time.August, 8, 20, 0, 0, 0, time.UTC)
	oldHold, err := service.CreateBooking(CreateBookingInput{OfferingID: "off-bowling", CustomerName: "Late", CustomerPhone: "+77010000001", StartAt: start, DurationMin: 120, GuestCount: 2, AcceptedPolicy: true}, "public", true)
	if err != nil {
		t.Fatalf("create hold: %v", err)
	}

	service.now = func() time.Time { return testNow.Add(16 * time.Minute) }
	replacement, err := service.CreateBooking(CreateBookingInput{OfferingID: "off-bowling", CustomerName: "Replacement", CustomerPhone: "+77010000002", StartAt: start, DurationMin: 120, GuestCount: 2}, "manager", false)
	if err != nil {
		t.Fatalf("create replacement after hold expiry: %v", err)
	}

	_, err = service.RecordPayment(oldHold.Booking.ID, "manager", PaymentInput{Amount: oldHold.Booking.DepositAmount})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected late-payment rejection, got %v", err)
	}
	oldView, err := service.Booking(oldHold.Booking.ID)
	if err != nil {
		t.Fatalf("load old hold: %v", err)
	}
	if oldView.Booking.Status != domain.BookingExpired || oldView.Booking.PaidAmount != 0 || len(oldView.Resources) != 0 {
		t.Fatalf("expired hold was revived: status=%s paid=%d resources=%d", oldView.Booking.Status, oldView.Booking.PaidAmount, len(oldView.Resources))
	}
	newView, err := service.Booking(replacement.Booking.ID)
	if err != nil {
		t.Fatalf("load replacement: %v", err)
	}
	if newView.Booking.Status != domain.BookingConfirmed || len(newView.Resources) != 1 {
		t.Fatalf("replacement booking was damaged: status=%s resources=%d", newView.Booking.Status, len(newView.Resources))
	}
}
