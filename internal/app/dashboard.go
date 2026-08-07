package app

import (
	"sort"
	"strings"
	"time"

	"github.com/aleka7sk/booking-os/internal/domain"
)

func (s *Service) Dashboard() DashboardView {
	state := s.store.Snapshot()
	now := s.now()
	loc := loadLocation(state.Organization.Timezone)
	localNow := now.In(loc)
	dayStartLocal := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, loc)
	dayEndLocal := dayStartLocal.AddDate(0, 0, 1)
	dayStart, dayEnd := dayStartLocal.UTC(), dayEndLocal.UTC()
	monthStart := time.Date(localNow.Year(), localNow.Month(), 1, 0, 0, 0, 0, loc).UTC()

	view := DashboardView{Date: dayStartLocal.Format("2006-01-02")}
	upcoming := make([]BookingView, 0)
	sourceCounts := map[domain.Source]int{}
	for _, booking := range state.Bookings {
		if booking.Status == domain.BookingRequested {
			view.PendingRequests++
		}
		if booking.Status == domain.BookingHeld && booking.HoldExpiresAt != nil && booking.HoldExpiresAt.After(now) {
			view.ActiveHolds++
		}
		if booking.StartAt.Before(dayEnd) && booking.EndAt.After(dayStart) && booking.Status != domain.BookingCancelled && booking.Status != domain.BookingRejected && booking.Status != domain.BookingExpired {
			view.TodayBookings++
			view.TodayRevenue += booking.PaidAmount
		}
		if booking.CreatedAt.After(monthStart) {
			view.MonthRevenue += booking.PaidAmount
			sourceCounts[booking.Source]++
		}
		if booking.StartAt.After(now.Add(-30*time.Minute)) && booking.Status != domain.BookingCancelled && booking.Status != domain.BookingRejected && booking.Status != domain.BookingExpired && booking.Status != domain.BookingCompleted {
			upcoming = append(upcoming, buildBookingView(state, booking))
		}
	}
	sort.Slice(upcoming, func(i, j int) bool { return upcoming[i].Booking.StartAt.Before(upcoming[j].Booking.StartAt) })
	if len(upcoming) > 8 {
		upcoming = upcoming[:8]
	}
	view.Upcoming = upcoming
	view.Occupancy = occupancyForRange(state, dayStart, dayEnd, now)

	for i := 0; i < 7; i++ {
		startLocal := dayStartLocal.AddDate(0, 0, i)
		endLocal := startLocal.AddDate(0, 0, 1)
		start, end := startLocal.UTC(), endLocal.UTC()
		count := 0
		for _, booking := range state.Bookings {
			if booking.StartAt.Before(end) && booking.EndAt.After(start) && booking.Status != domain.BookingCancelled && booking.Status != domain.BookingRejected && booking.Status != domain.BookingExpired {
				count++
			}
		}
		view.WeeklyLoad = append(view.WeeklyLoad, DailyLoad{Date: startLocal.Format("2006-01-02"), Label: russianWeekday(startLocal.Weekday()), Bookings: count, Occupancy: occupancyForRange(state, start, end, now)})
	}

	for source, count := range sourceCounts {
		view.Sources = append(view.Sources, SourceMetric{Source: source, Count: count})
	}
	sort.Slice(view.Sources, func(i, j int) bool { return view.Sources[i].Count > view.Sources[j].Count })

	for _, resource := range state.Resources {
		if !resource.Active {
			continue
		}
		pulse := ResourcePulse{Resource: resource, State: "FREE"}
		for _, allocation := range state.Allocations {
			if allocation.ResourceID != resource.ID || allocation.Status != domain.AllocationActive || !allocation.StartAt.Before(now) || !allocation.EndAt.After(now) {
				continue
			}
			booking, _, ok := findBooking(state, allocation.BookingID)
			if !ok || !bookingOccupies(booking, now) {
				continue
			}
			until := booking.EndAt
			bookingView := buildBookingView(state, booking)
			pulse.State, pulse.Until, pulse.Booking = "BUSY", &until, &bookingView
			break
		}
		view.ResourcePulse = append(view.ResourcePulse, pulse)
	}
	sort.Slice(view.ResourcePulse, func(i, j int) bool {
		if view.ResourcePulse[i].State != view.ResourcePulse[j].State {
			return view.ResourcePulse[i].State == "BUSY"
		}
		return view.ResourcePulse[i].Resource.Name < view.ResourcePulse[j].Resource.Name
	})
	return view
}

func occupancyForRange(state domain.State, start, end, now time.Time) int {
	if !end.After(start) {
		return 0
	}
	location, ok := findLocation(state, "loc-astana")
	if !ok && len(state.Locations) > 0 {
		location = state.Locations[0]
		ok = true
	}
	if !ok {
		return 0
	}
	loc := loadLocation(location.Timezone)
	localStart := start.In(loc)
	hours := location.WeeklyHours[strings.ToLower(localStart.Weekday().String())]
	openMinutes := 0
	for _, window := range hours {
		ws, ok1 := combineDateAndClock(time.Date(localStart.Year(), localStart.Month(), localStart.Day(), 0, 0, 0, 0, loc), window.Start)
		we, ok2 := combineDateAndClock(time.Date(localStart.Year(), localStart.Month(), localStart.Day(), 0, 0, 0, 0, loc), window.End)
		if !ok1 || !ok2 {
			continue
		}
		if !we.After(ws) {
			we = we.Add(24 * time.Hour)
		}
		openMinutes += int(we.Sub(ws).Minutes())
	}
	capacityUnits := 0
	resourceCapacity := map[string]int{}
	for _, resource := range state.Resources {
		if !resource.Active || resource.LocationID != location.ID {
			continue
		}
		capacity := resource.Capacity
		if capacity < 1 {
			capacity = 1
		}
		resourceCapacity[resource.ID] = capacity
		capacityUnits += capacity
	}
	if openMinutes == 0 || capacityUnits == 0 {
		return 0
	}
	bookedUnitMinutes := 0.0
	bookings := map[string]domain.Booking{}
	for _, booking := range state.Bookings {
		bookings[booking.ID] = booking
	}
	for _, allocation := range state.Allocations {
		if allocation.Status != domain.AllocationActive {
			continue
		}
		booking, ok := bookings[allocation.BookingID]
		if !ok || !bookingOccupies(booking, now) {
			continue
		}
		overlapStart := maxTime(allocation.StartAt, start)
		overlapEnd := minTime(allocation.EndAt, end)
		if !overlapEnd.After(overlapStart) {
			continue
		}
		quantity := allocation.Quantity
		if quantity < 1 {
			quantity = 1
		}
		if _, ok := resourceCapacity[allocation.ResourceID]; !ok {
			continue
		}
		bookedUnitMinutes += overlapEnd.Sub(overlapStart).Minutes() * float64(quantity)
	}
	value := int(bookedUnitMinutes * 100 / float64(openMinutes*capacityUnits))
	if value > 100 {
		value = 100
	}
	if value < 0 {
		value = 0
	}
	return value
}

func russianWeekday(day time.Weekday) string {
	labels := []string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"}
	return labels[int(day)]
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
