package app

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/aleka7sk/booking-os/internal/domain"
	"github.com/aleka7sk/booking-os/internal/security"
)

func (s *Service) CreateBooking(input CreateBookingInput, actorID string, public bool) (BookingView, error) {
	input.CustomerName = strings.TrimSpace(input.CustomerName)
	input.CustomerPhone = normalizePhone(input.CustomerPhone)
	if input.CustomerName == "" {
		return BookingView{}, invalid("customer_name_required", "Укажите имя клиента")
	}
	if len(input.CustomerPhone) < 10 {
		return BookingView{}, invalid("customer_phone_invalid", "Укажите корректный номер телефона")
	}
	if input.StartAt.IsZero() {
		return BookingView{}, invalid("start_required", "Укажите дату и время")
	}
	if input.Source == "" {
		if public {
			input.Source = domain.SourcePublicPage
		} else {
			input.Source = domain.SourceManager
		}
	}

	var createdID string
	err := s.store.Update(func(state *domain.State) error {
		now := s.now()
		offering, ok := findOffering(*state, input.OfferingID)
		if !ok || !offering.Active {
			return notFound("offering")
		}
		if public && !offering.PublicEnabled {
			return &AppError{Kind: ErrForbidden, Code: "offering_not_public", Message: "Онлайн-бронирование этой услуги отключено"}
		}
		if public && !input.AcceptedPolicy {
			return invalid("policy_not_accepted", "Подтвердите согласие с условиями отмены и переноса")
		}
		duration, err := normalizeDuration(offering, input.DurationMin)
		if err != nil {
			return err
		}
		guests, err := normalizeGuests(offering, input.GuestCount)
		if err != nil {
			return err
		}
		start := input.StartAt.UTC().Truncate(time.Minute)
		end := start.Add(time.Duration(duration) * time.Minute)
		if public {
			if err := validatePublicTime(*state, offering, start, duration, now); err != nil {
				return err
			}
		}

		status := domain.BookingConfirmed
		if input.RequestOnly {
			status = domain.BookingRequested
		}
		if public {
			switch offering.ConfirmationMode {
			case domain.ConfirmationManagerApproval, domain.ConfirmationApprovalAndPayment, domain.ConfirmationManualQuote:
				status = domain.BookingRequested
			case domain.ConfirmationPaymentGated:
				status = domain.BookingHeld
			default:
				status = domain.BookingConfirmed
			}
		}

		occupiedStart := start.Add(-time.Duration(offering.BufferBeforeMin) * time.Minute)
		occupiedEnd := end.Add(time.Duration(offering.BufferAfterMin) * time.Minute)
		var plan []domain.Allocation
		meta := allocationMeta{ResourceCount: requiredResourceCount(offering, guests)}
		if status.OccupiesResource() {
			plan, meta, err = planAllocations(*state, offering, occupiedStart, occupiedEnd, guests, "", now)
			if err != nil {
				return err
			}
		}
		total, deposit, lines := calculatePrice(offering, duration, guests, meta.ResourceCount)
		if input.PaidAmount < 0 || input.PaidAmount > total {
			return invalid("paid_amount_invalid", "Сумма оплаты не может быть больше стоимости брони")
		}

		customerID := ""
		for i := range state.Customers {
			if normalizePhone(state.Customers[i].Phone) == input.CustomerPhone {
				customerID = state.Customers[i].ID
				state.Customers[i].Name = input.CustomerName
				if input.CustomerEmail != "" {
					state.Customers[i].Email = strings.TrimSpace(input.CustomerEmail)
				}
				state.Customers[i].UpdatedAt = now
				break
			}
		}
		if customerID == "" {
			customerID = newID("cus")
			state.Customers = append(state.Customers, domain.Customer{ID: customerID, Name: input.CustomerName, Phone: input.CustomerPhone, Email: strings.TrimSpace(input.CustomerEmail), Language: "ru", CreatedAt: now, UpdatedAt: now})
		}

		publicToken, tokenErr := security.RandomToken(24)
		if tokenErr != nil {
			return tokenErr
		}
		ref := fmt.Sprintf("B-%04d", state.NextReference)
		state.NextReference++
		var policyAcceptedAt *time.Time
		if public {
			acceptedAt := now
			policyAcceptedAt = &acceptedAt
		}
		booking := domain.Booking{
			ID: newID("book"), Reference: ref, OrganizationID: state.Organization.ID, LocationID: offering.LocationID,
			OfferingID: offering.ID, CustomerID: customerID, Status: status,
			PaymentStatus: paymentStatus(input.PaidAmount, total, deposit), AttendanceStatus: domain.AttendanceNotStarted,
			Source: input.Source, StartAt: start, EndAt: end, OccupiedStartAt: occupiedStart, OccupiedEndAt: occupiedEnd,
			GuestCount: guests, TotalAmount: total, DepositAmount: deposit, PaidAmount: input.PaidAmount,
			Notes: strings.TrimSpace(input.Notes), InternalNotes: strings.TrimSpace(input.InternalNotes),
			CancellationSnapshot: offering.CancellationPolicy, PolicyAcceptedAt: policyAcceptedAt, PriceSnapshot: lines, PublicToken: publicToken,
			CreatedBy: actorID, CreatedAt: now, UpdatedAt: now,
		}
		if status == domain.BookingHeld {
			expires := now.Add(time.Duration(maxInt(offering.HoldDurationMin, 10)) * time.Minute)
			booking.HoldExpiresAt = &expires
		}
		if status == domain.BookingConfirmed {
			confirmed := now
			booking.ConfirmedAt = &confirmed
		}
		state.Bookings = append(state.Bookings, booking)
		createdID = booking.ID
		for _, allocation := range plan {
			allocation.ID = newID("alloc")
			allocation.BookingID = booking.ID
			allocation.CreatedAt = now
			state.Allocations = append(state.Allocations, allocation)
		}
		appendAudit(state, actorID, "booking.created", "booking", booking.ID, "Создана бронь "+booking.Reference, map[string]any{"status": booking.Status, "source": booking.Source}, now)
		return nil
	})
	if err != nil {
		return BookingView{}, err
	}
	return s.Booking(createdID)
}

func (s *Service) Booking(id string) (BookingView, error) {
	state := s.store.Snapshot()
	booking, _, ok := findBooking(state, id)
	if !ok {
		return BookingView{}, notFound("booking")
	}
	return buildBookingView(state, booking), nil
}

func (s *Service) PublicBooking(token string) (BookingView, error) {
	state := s.store.Snapshot()
	for _, booking := range state.Bookings {
		if booking.PublicToken == token {
			return buildBookingView(state, booking), nil
		}
	}
	return BookingView{}, notFound("booking")
}

func (s *Service) ListBookings(from, to *time.Time, status domain.BookingStatus) []BookingView {
	state := s.store.Snapshot()
	out := make([]BookingView, 0, len(state.Bookings))
	for _, booking := range state.Bookings {
		if from != nil && booking.EndAt.Before(*from) {
			continue
		}
		if to != nil && !booking.StartAt.Before(*to) {
			continue
		}
		if status != "" && booking.Status != status {
			continue
		}
		out = append(out, buildBookingView(state, booking))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Booking.StartAt.Before(out[j].Booking.StartAt) })
	return out
}

func (s *Service) Calendar(from, to time.Time) CalendarView {
	state := s.store.Snapshot()
	resources := append([]domain.Resource(nil), state.Resources...)
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].Pool == resources[j].Pool {
			return resources[i].Name < resources[j].Name
		}
		return resources[i].Pool < resources[j].Pool
	})
	bookings := make([]BookingView, 0)
	for _, booking := range state.Bookings {
		if booking.EndAt.After(from) && booking.StartAt.Before(to) && booking.Status != domain.BookingCancelled && booking.Status != domain.BookingRejected && booking.Status != domain.BookingExpired {
			bookings = append(bookings, buildBookingView(state, booking))
		}
	}
	blocks := make([]domain.ResourceBlock, 0)
	for _, block := range state.ResourceBlocks {
		if block.EndAt.After(from) && block.StartAt.Before(to) {
			blocks = append(blocks, block)
		}
	}
	return CalendarView{From: from, To: to, Resources: resources, Bookings: bookings, Blocks: blocks}
}

func (s *Service) ConfirmBooking(id, actorID string) (BookingView, error) {
	expired := false
	err := s.store.Update(func(state *domain.State) error {
		now := s.now()
		booking, index, ok := findBooking(*state, id)
		if !ok {
			return notFound("booking")
		}
		if expireHeldBooking(state, &booking, index, now) {
			expired = true
			return nil
		}
		if booking.Status != domain.BookingRequested && booking.Status != domain.BookingHeld {
			return invalid("booking_cannot_confirm", "Эту бронь нельзя подтвердить в текущем статусе")
		}
		offering, ok := findOffering(*state, booking.OfferingID)
		if !ok {
			return notFound("offering")
		}
		if booking.Status == domain.BookingRequested {
			plan, _, err := planAllocations(*state, offering, booking.OccupiedStartAt, booking.OccupiedEndAt, booking.GuestCount, booking.ID, now)
			if err != nil {
				return err
			}
			for _, allocation := range plan {
				allocation.ID, allocation.BookingID, allocation.CreatedAt = newID("alloc"), booking.ID, now
				state.Allocations = append(state.Allocations, allocation)
			}
		}
		if booking.DepositAmount > booking.PaidAmount {
			booking.Status = domain.BookingHeld
			expires := now.Add(time.Duration(maxInt(offering.HoldDurationMin, 10)) * time.Minute)
			booking.HoldExpiresAt = &expires
		} else {
			booking.Status = domain.BookingConfirmed
			booking.HoldExpiresAt = nil
			confirmed := now
			booking.ConfirmedAt = &confirmed
		}
		booking.UpdatedAt = now
		state.Bookings[index] = booking
		appendAudit(state, actorID, "booking.confirmed", "booking", booking.ID, "Подтверждена бронь "+booking.Reference, map[string]any{"status": booking.Status}, now)
		return nil
	})
	if err != nil {
		return BookingView{}, err
	}
	if expired {
		return BookingView{}, invalid("hold_expired", "Срок удержания истёк. Создайте новую бронь или выберите другое время")
	}
	return s.Booking(id)
}

func (s *Service) RejectBooking(id, actorID, reason string) (BookingView, error) {
	err := s.store.Update(func(state *domain.State) error {
		now := s.now()
		booking, index, ok := findBooking(*state, id)
		if !ok {
			return notFound("booking")
		}
		if booking.Status != domain.BookingRequested {
			return invalid("booking_cannot_reject", "Отклонить можно только новый запрос")
		}
		booking.Status = domain.BookingRejected
		booking.CancellationReason = strings.TrimSpace(reason)
		booking.UpdatedAt = now
		state.Bookings[index] = booking
		appendAudit(state, actorID, "booking.rejected", "booking", booking.ID, "Отклонена бронь "+booking.Reference, map[string]any{"reason": reason}, now)
		return nil
	})
	if err != nil {
		return BookingView{}, err
	}
	return s.Booking(id)
}

func (s *Service) RecordPayment(id, actorID string, input PaymentInput) (BookingView, error) {
	if input.Amount <= 0 {
		return BookingView{}, invalid("payment_amount_invalid", "Сумма оплаты должна быть больше нуля")
	}
	expired := false
	err := s.store.Update(func(state *domain.State) error {
		now := s.now()
		booking, index, ok := findBooking(*state, id)
		if !ok {
			return notFound("booking")
		}
		if expireHeldBooking(state, &booking, index, now) {
			expired = true
			return nil
		}
		if booking.Status == domain.BookingCancelled || booking.Status == domain.BookingRejected || booking.Status == domain.BookingExpired {
			return invalid("booking_payment_closed", "Нельзя принять оплату по закрытой брони")
		}
		if booking.PaidAmount+input.Amount > booking.TotalAmount {
			return invalid("payment_exceeds_total", "Оплата превышает остаток по брони")
		}
		booking.PaidAmount += input.Amount
		booking.PaymentStatus = paymentStatus(booking.PaidAmount, booking.TotalAmount, booking.DepositAmount)
		if booking.Status == domain.BookingHeld && booking.PaidAmount >= booking.DepositAmount {
			booking.Status = domain.BookingConfirmed
			booking.HoldExpiresAt = nil
			confirmed := now
			booking.ConfirmedAt = &confirmed
		}
		booking.UpdatedAt = now
		state.Bookings[index] = booking
		appendAudit(state, actorID, "payment.recorded", "booking", booking.ID, fmt.Sprintf("Получена оплата %d ₸ по %s", input.Amount, booking.Reference), map[string]any{"amount": input.Amount, "note": input.Note}, now)
		return nil
	})
	if err != nil {
		return BookingView{}, err
	}
	if expired {
		return BookingView{}, invalid("hold_expired", "Оплата не зафиксирована: срок удержания уже истёк")
	}
	return s.Booking(id)
}

func (s *Service) CancelBooking(id, actorID string, input CancelInput) (BookingView, error) {
	expired := false
	err := s.store.Update(func(state *domain.State) error {
		now := s.now()
		booking, index, ok := findBooking(*state, id)
		if !ok {
			return notFound("booking")
		}
		if expireHeldBooking(state, &booking, index, now) {
			expired = true
			return nil
		}
		switch booking.Status {
		case domain.BookingCancelled, domain.BookingRejected, domain.BookingExpired, domain.BookingCompleted:
			return invalid("booking_cannot_cancel", "Эту бронь нельзя отменить в текущем статусе")
		}
		booking.Status = domain.BookingCancelled
		booking.CancellationReason = strings.TrimSpace(input.Reason)
		booking.UpdatedAt = now
		booking.HoldExpiresAt = nil
		booking.CancelledAt = &now
		state.Bookings[index] = booking
		deactivateAllocations(state, booking.ID, domain.AllocationCancelled)
		appendAudit(state, actorID, "booking.cancelled", "booking", booking.ID, "Отменена бронь "+booking.Reference, map[string]any{"reason": input.Reason}, now)
		return nil
	})
	if err != nil {
		return BookingView{}, err
	}
	if expired {
		return BookingView{}, invalid("hold_expired", "Срок удержания уже истёк; отменять бронь не требуется")
	}
	return s.Booking(id)
}

func (s *Service) RescheduleBooking(id, actorID string, input RescheduleInput) (BookingView, error) {
	if input.StartAt.IsZero() {
		return BookingView{}, invalid("start_required", "Укажите новое время")
	}
	expired := false
	err := s.store.Update(func(state *domain.State) error {
		now := s.now()
		booking, index, ok := findBooking(*state, id)
		if !ok {
			return notFound("booking")
		}
		if expireHeldBooking(state, &booking, index, now) {
			expired = true
			return nil
		}
		if booking.Status == domain.BookingCancelled || booking.Status == domain.BookingRejected || booking.Status == domain.BookingExpired || booking.Status == domain.BookingCompleted {
			return invalid("booking_cannot_reschedule", "Эту бронь нельзя перенести")
		}
		offering, ok := findOffering(*state, booking.OfferingID)
		if !ok {
			return notFound("offering")
		}
		duration, err := normalizeDuration(offering, input.DurationMin)
		if err != nil {
			return err
		}
		start := input.StartAt.UTC().Truncate(time.Minute)
		end := start.Add(time.Duration(duration) * time.Minute)
		occupiedStart := start.Add(-time.Duration(offering.BufferBeforeMin) * time.Minute)
		occupiedEnd := end.Add(time.Duration(offering.BufferAfterMin) * time.Minute)
		var plan []domain.Allocation
		meta := allocationMeta{ResourceCount: requiredResourceCount(offering, booking.GuestCount)}
		if booking.Status.OccupiesResource() {
			plan, meta, err = planAllocations(*state, offering, occupiedStart, occupiedEnd, booking.GuestCount, booking.ID, now)
			if err != nil {
				return err
			}
		}
		oldStart := booking.StartAt
		booking.StartAt, booking.EndAt = start, end
		booking.OccupiedStartAt, booking.OccupiedEndAt = occupiedStart, occupiedEnd
		booking.TotalAmount, booking.DepositAmount, booking.PriceSnapshot = calculatePrice(offering, duration, booking.GuestCount, meta.ResourceCount)
		booking.PaymentStatus = paymentStatus(booking.PaidAmount, booking.TotalAmount, booking.DepositAmount)
		booking.UpdatedAt = now
		state.Bookings[index] = booking
		if booking.Status.OccupiesResource() {
			deactivateAllocations(state, booking.ID, domain.AllocationCancelled)
			for _, allocation := range plan {
				allocation.ID, allocation.BookingID, allocation.CreatedAt = newID("alloc"), booking.ID, now
				state.Allocations = append(state.Allocations, allocation)
			}
		}
		appendAudit(state, actorID, "booking.rescheduled", "booking", booking.ID, "Перенесена бронь "+booking.Reference, map[string]any{"from": oldStart, "to": start, "reason": input.Reason}, now)
		return nil
	})
	if err != nil {
		return BookingView{}, err
	}
	if expired {
		return BookingView{}, invalid("hold_expired", "Срок удержания истёк. Для переноса создайте новую бронь")
	}
	return s.Booking(id)
}

func (s *Service) CompleteBooking(id, actorID string) (BookingView, error) {
	return s.finishBooking(id, actorID, false)
}

func (s *Service) NoShowBooking(id, actorID string) (BookingView, error) {
	return s.finishBooking(id, actorID, true)
}

func (s *Service) finishBooking(id, actorID string, noShow bool) (BookingView, error) {
	err := s.store.Update(func(state *domain.State) error {
		now := s.now()
		booking, index, ok := findBooking(*state, id)
		if !ok {
			return notFound("booking")
		}
		if booking.Status != domain.BookingConfirmed {
			return invalid("booking_cannot_finish", "Завершить можно только подтверждённую бронь")
		}
		booking.Status = domain.BookingCompleted
		if noShow {
			booking.AttendanceStatus = domain.AttendanceNoShow
		} else {
			booking.AttendanceStatus = domain.AttendanceAttended
		}
		booking.CompletedAt, booking.UpdatedAt = &now, now
		state.Bookings[index] = booking
		action := "booking.completed"
		summary := "Завершена бронь " + booking.Reference
		if noShow {
			action, summary = "booking.no_show", "Отмечена неявка по "+booking.Reference
		}
		appendAudit(state, actorID, action, "booking", booking.ID, summary, nil, now)
		return nil
	})
	if err != nil {
		return BookingView{}, err
	}
	return s.Booking(id)
}

func (s *Service) ExpireHolds() (int, error) {
	now := s.now()
	hasExpired := false
	for _, booking := range s.store.Snapshot().Bookings {
		if booking.Status == domain.BookingHeld && booking.HoldExpiresAt != nil && !booking.HoldExpiresAt.After(now) {
			hasExpired = true
			break
		}
	}
	if !hasExpired {
		return 0, nil
	}

	expired := 0
	err := s.store.Update(func(state *domain.State) error {
		for i := range state.Bookings {
			booking := state.Bookings[i]
			if expireHeldBooking(state, &booking, i, now) {
				expired++
			}
		}
		return nil
	})
	return expired, err
}

func expireHeldBooking(state *domain.State, booking *domain.Booking, index int, now time.Time) bool {
	if booking.Status != domain.BookingHeld || booking.HoldExpiresAt == nil || booking.HoldExpiresAt.After(now) {
		return false
	}
	booking.Status, booking.UpdatedAt = domain.BookingExpired, now
	state.Bookings[index] = *booking
	deactivateAllocations(state, booking.ID, domain.AllocationExpired)
	appendAudit(state, "system", "booking.expired", "booking", booking.ID, "Истёк срок удержания "+booking.Reference, nil, now)
	return true
}

func buildBookingView(state domain.State, booking domain.Booking) BookingView {
	offering, _ := findOffering(state, booking.OfferingID)
	customer, _ := findCustomer(state, booking.CustomerID)
	resources := make([]domain.Resource, 0)
	seen := map[string]bool{}
	for _, allocation := range state.Allocations {
		if allocation.BookingID != booking.ID || allocation.Status != domain.AllocationActive || seen[allocation.ResourceID] {
			continue
		}
		if resource, ok := findResource(state, allocation.ResourceID); ok {
			resources = append(resources, resource)
			seen[resource.ID] = true
		}
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].Name < resources[j].Name })
	return BookingView{Booking: booking, Offering: offering, Customer: customer, Resources: resources}
}

func deactivateAllocations(state *domain.State, bookingID string, status domain.AllocationStatus) {
	for i := range state.Allocations {
		if state.Allocations[i].BookingID == bookingID && state.Allocations[i].Status == domain.AllocationActive {
			state.Allocations[i].Status = status
		}
	}
}

func normalizePhone(value string) string {
	var digits []rune
	for _, r := range value {
		if unicode.IsDigit(r) {
			digits = append(digits, r)
		}
	}
	if len(digits) == 10 {
		digits = append([]rune{'7'}, digits...)
	}
	if len(digits) == 11 && digits[0] == '8' {
		digits[0] = '7'
	}
	if len(digits) == 11 {
		return "+" + string(digits)
	}
	return string(digits)
}

func requiredResourceCount(offering domain.Offering, guests int) int {
	count := maxInt(offering.ResourcesRequired, 1)
	if offering.AllocationMode == domain.AllocationExclusive && offering.CapacityPerResource > 0 {
		calculated := (guests + offering.CapacityPerResource - 1) / offering.CapacityPerResource
		if calculated > count {
			count = calculated
		}
	}
	return count
}

func paymentStatus(paid, total, deposit int64) domain.PaymentStatus {
	if paid >= total && total > 0 {
		return domain.PaymentPaid
	}
	if paid > 0 {
		return domain.PaymentPartiallyPaid
	}
	if deposit > 0 {
		return domain.PaymentUnpaid
	}
	return domain.PaymentNotRequired
}

func validatePublicTime(state domain.State, offering domain.Offering, start time.Time, duration int, now time.Time) error {
	location, ok := findLocation(state, offering.LocationID)
	if !ok {
		return notFound("location")
	}
	if start.Before(now.Add(time.Duration(offering.MinLeadMin) * time.Minute)) {
		return invalid("booking_too_soon", "Это время уже недоступно для онлайн-бронирования")
	}
	if start.After(now.AddDate(0, 0, offering.MaxAdvanceDays)) {
		return invalid("booking_too_far", "Выбранная дата находится за пределами доступного периода")
	}
	loc := loadLocation(location.Timezone)
	localStart := start.In(loc)
	day := time.Date(localStart.Year(), localStart.Month(), localStart.Day(), 0, 0, 0, 0, loc)
	for _, candidate := range candidateStarts(location, offering, day, duration) {
		if candidate.UTC().Equal(start) {
			return nil
		}
	}
	return invalid("slot_not_in_schedule", "Выбранное время не входит в расписание услуги")
}
