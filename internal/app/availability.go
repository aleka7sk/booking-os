package app

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/aleka7sk/booking-os/internal/domain"
)

func (s *Service) SearchAvailability(input AvailabilityInput) ([]AvailabilitySlot, error) {
	state := s.store.Snapshot()
	offering, ok := findOffering(state, input.OfferingID)
	if !ok || !offering.Active {
		return nil, notFound("offering")
	}
	location, ok := findLocation(state, offering.LocationID)
	if !ok || !location.Active {
		return nil, notFound("location")
	}

	duration, err := normalizeDuration(offering, input.DurationMin)
	if err != nil {
		return nil, err
	}
	guests, err := normalizeGuests(offering, input.GuestCount)
	if err != nil {
		return nil, err
	}
	loc := loadLocation(location.Timezone)
	day, err := time.ParseInLocation("2006-01-02", input.Date, loc)
	if err != nil {
		return nil, invalid("invalid_date", "Дата должна быть в формате YYYY-MM-DD")
	}

	starts := candidateStarts(location, offering, day, duration)
	now := s.now()
	maxAt := now.AddDate(0, 0, offering.MaxAdvanceDays)
	slots := make([]AvailabilitySlot, 0, len(starts))
	for _, startLocal := range starts {
		start := startLocal.UTC()
		end := start.Add(time.Duration(duration) * time.Minute)
		if start.Before(now.Add(time.Duration(offering.MinLeadMin) * time.Minute)) {
			continue
		}
		if start.After(maxAt) {
			continue
		}

		occupiedStart := start.Add(-time.Duration(offering.BufferBeforeMin) * time.Minute)
		occupiedEnd := end.Add(time.Duration(offering.BufferAfterMin) * time.Minute)
		plan, meta, planErr := planAllocations(state, offering, occupiedStart, occupiedEnd, guests, "", now)
		if planErr != nil {
			continue
		}
		total, deposit, lines := calculatePrice(offering, duration, guests, meta.ResourceCount)
		ids := make([]string, 0, len(plan))
		for _, allocation := range plan {
			ids = append(ids, allocation.ResourceID)
		}
		slots = append(slots, AvailabilitySlot{
			StartAt: start, EndAt: end, DurationMin: duration, Available: true,
			ScarcityLabel: meta.ScarcityLabel, TotalAmount: total, DepositAmount: deposit,
			PriceLines: lines, ResourceIDs: ids,
		})
	}
	return slots, nil
}

func normalizeDuration(offering domain.Offering, requested int) (int, error) {
	if requested <= 0 {
		requested = offering.DurationMin
	}
	min := offering.MinDurationMin
	if min <= 0 {
		min = offering.DurationMin
	}
	max := offering.MaxDurationMin
	if max <= 0 {
		max = min
	}
	if requested < min || requested > max {
		return 0, invalid("duration_out_of_range", fmt.Sprintf("Продолжительность должна быть от %d до %d минут", min, max))
	}
	step := offering.DurationStepMin
	if step <= 0 {
		step = min
	}
	if (requested-min)%step != 0 {
		return 0, invalid("duration_step_mismatch", fmt.Sprintf("Продолжительность меняется с шагом %d минут", step))
	}
	return requested, nil
}

func normalizeGuests(offering domain.Offering, requested int) (int, error) {
	if requested <= 0 {
		requested = offering.MinGuests
	}
	if requested < offering.MinGuests || requested > offering.MaxGuests {
		return 0, invalid("guest_count_out_of_range", fmt.Sprintf("Количество гостей должно быть от %d до %d", offering.MinGuests, offering.MaxGuests))
	}
	return requested, nil
}

func candidateStarts(location domain.Location, offering domain.Offering, day time.Time, duration int) []time.Time {
	if offering.SchedulingMode == domain.SchedulingFixed {
		var out []time.Time
		for _, session := range offering.FixedSessions {
			if session.Weekday != int(day.Weekday()) {
				continue
			}
			if start, ok := combineDateAndClock(day, session.Start); ok {
				out = append(out, start)
			}
		}
		return out
	}

	key := strings.ToLower(day.Weekday().String())
	hours := location.WeeklyHours[key]
	step := offering.StartStepMin
	if step <= 0 {
		step = 30
	}
	var out []time.Time
	for _, window := range hours {
		windowStart, okStart := combineDateAndClock(day, window.Start)
		windowEnd, okEnd := combineDateAndClock(day, window.End)
		if !okStart || !okEnd {
			continue
		}
		if !windowEnd.After(windowStart) {
			windowEnd = windowEnd.Add(24 * time.Hour)
		}
		latestStart := windowEnd.Add(-time.Duration(duration+offering.BufferAfterMin) * time.Minute)
		for cursor := windowStart; !cursor.After(latestStart); cursor = cursor.Add(time.Duration(step) * time.Minute) {
			out = append(out, cursor)
		}
	}
	return out
}

func combineDateAndClock(day time.Time, clock string) (time.Time, bool) {
	parsed, err := time.Parse("15:04", clock)
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(day.Year(), day.Month(), day.Day(), parsed.Hour(), parsed.Minute(), 0, 0, day.Location()), true
}

func loadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err == nil {
		return loc
	}
	return time.FixedZone("UTC+5", 5*60*60)
}

type allocationMeta struct {
	ResourceCount int
	ScarcityLabel string
}

func planAllocations(state domain.State, offering domain.Offering, start, end time.Time, guests int, excludeBookingID string, now time.Time) ([]domain.Allocation, allocationMeta, error) {
	resources := make([]domain.Resource, 0)
	for _, resource := range state.Resources {
		if resource.Active && resource.LocationID == offering.LocationID && resource.Pool == offering.ResourcePool {
			resources = append(resources, resource)
		}
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].Name < resources[j].Name })
	if len(resources) == 0 {
		return nil, allocationMeta{}, conflict("no_resources", "Для услуги не настроены активные ресурсы")
	}

	if offering.AllocationMode == domain.AllocationShared {
		remaining := guests
		plan := make([]domain.Allocation, 0, len(resources))
		totalFree := 0
		for _, resource := range resources {
			free := resource.Capacity - overlappingQuantity(state, resource.ID, start, end, excludeBookingID, now)
			if blocked(state, resource.ID, start, end) {
				free = 0
			}
			if free < 0 {
				free = 0
			}
			totalFree += free
			if remaining <= 0 || free == 0 {
				continue
			}
			quantity := free
			if quantity > remaining {
				quantity = remaining
			}
			plan = append(plan, domain.Allocation{ResourceID: resource.ID, StartAt: start, EndAt: end, Quantity: quantity, Status: domain.AllocationActive})
			remaining -= quantity
		}
		if remaining > 0 {
			return nil, allocationMeta{}, conflict("capacity_unavailable", "На выбранное время недостаточно свободных мест")
		}
		after := totalFree - guests
		label := ""
		if after <= 2 {
			label = fmt.Sprintf("Осталось %d мест", maxInt(after, 0))
		}
		return plan, allocationMeta{ResourceCount: len(plan), ScarcityLabel: label}, nil
	}

	required := offering.ResourcesRequired
	if required <= 0 {
		required = 1
	}
	if offering.CapacityPerResource > 0 {
		calculated := int(math.Ceil(float64(guests) / float64(offering.CapacityPerResource)))
		if calculated > required {
			required = calculated
		}
	}
	freeResources := make([]domain.Resource, 0, len(resources))
	for _, resource := range resources {
		if blocked(state, resource.ID, start, end) {
			continue
		}
		if overlappingQuantity(state, resource.ID, start, end, excludeBookingID, now) == 0 {
			freeResources = append(freeResources, resource)
		}
	}
	if len(freeResources) < required {
		return nil, allocationMeta{}, conflict("resources_unavailable", "На выбранное время недостаточно свободных ресурсов")
	}
	plan := make([]domain.Allocation, 0, required)
	for _, resource := range freeResources[:required] {
		plan = append(plan, domain.Allocation{ResourceID: resource.ID, StartAt: start, EndAt: end, Quantity: 1, Status: domain.AllocationActive})
	}
	label := ""
	remaining := len(freeResources) - required
	if remaining <= 1 {
		label = fmt.Sprintf("Остался %d вариант", maxInt(remaining, 0)+1)
	}
	return plan, allocationMeta{ResourceCount: required, ScarcityLabel: label}, nil
}

func overlappingQuantity(state domain.State, resourceID string, start, end time.Time, excludeBookingID string, now time.Time) int {
	bookings := make(map[string]domain.Booking, len(state.Bookings))
	for _, booking := range state.Bookings {
		bookings[booking.ID] = booking
	}
	used := 0
	for _, allocation := range state.Allocations {
		if allocation.ResourceID != resourceID || allocation.BookingID == excludeBookingID || allocation.Status != domain.AllocationActive {
			continue
		}
		if !overlaps(allocation.StartAt, allocation.EndAt, start, end) {
			continue
		}
		booking, ok := bookings[allocation.BookingID]
		if !ok || !bookingOccupies(booking, now) {
			continue
		}
		used += allocation.Quantity
	}
	return used
}

func bookingOccupies(booking domain.Booking, now time.Time) bool {
	if !booking.Status.OccupiesResource() {
		return false
	}
	if booking.Status == domain.BookingHeld && booking.HoldExpiresAt != nil && !booking.HoldExpiresAt.After(now) {
		return false
	}
	return true
}

func blocked(state domain.State, resourceID string, start, end time.Time) bool {
	for _, block := range state.ResourceBlocks {
		if block.ResourceID == resourceID && overlaps(block.StartAt, block.EndAt, start, end) {
			return true
		}
	}
	return false
}

func overlaps(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}

func calculatePrice(offering domain.Offering, durationMin, guests, resourceCount int) (int64, int64, []domain.PriceLine) {
	if resourceCount <= 0 {
		resourceCount = maxInt(offering.ResourcesRequired, 1)
	}
	var total int64
	var label string
	var quantity int
	var unit int64
	switch offering.PriceMode {
	case domain.PricePerPerson:
		quantity, unit, label = guests, offering.BasePrice, "Участники"
		total = int64(quantity) * unit
	case domain.PricePerResource:
		quantity, unit, label = resourceCount, offering.BasePrice, "Ресурсы"
		total = int64(quantity) * unit
	case domain.PricePerDuration:
		quantity, unit, label = durationMin, offering.BasePrice, "Аренда"
		total = offering.BasePrice * int64(durationMin) / 60
	case domain.PricePerResourceHour:
		quantity, unit, label = resourceCount, offering.BasePrice, "Ресурс × время"
		total = offering.BasePrice * int64(resourceCount) * int64(durationMin) / 60
	case domain.PricePartyTier:
		label = "Пакет"
		quantity = 1
		for _, tier := range offering.PriceTiers {
			if guests >= tier.MinGuests && guests <= tier.MaxGuests {
				total, unit = tier.Amount, tier.Amount
				break
			}
		}
		if total == 0 {
			total, unit = offering.BasePrice, offering.BasePrice
		}
	case domain.PriceCustomQuote:
		label, quantity, unit, total = "Индивидуальный расчёт", 1, 0, 0
	default:
		label, quantity, unit, total = "Услуга", 1, offering.BasePrice, offering.BasePrice
	}
	lines := []domain.PriceLine{{Label: label, Quantity: quantity, UnitAmount: unit, Amount: total}}
	deposit := total * int64(offering.DepositPercent) / 100
	return total, deposit, lines
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
