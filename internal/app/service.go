package app

import (
	"crypto/rand"
	"encoding/base32"
	"sort"
	"strings"
	"time"

	"github.com/aleka7sk/booking-os/internal/domain"
	"github.com/aleka7sk/booking-os/internal/store"
)

type Service struct {
	store *store.JSONStore
	now   func() time.Time
}

func New(st *store.JSONStore) *Service {
	return &Service{store: st, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Service) State() domain.State { return s.store.Snapshot() }

func (s *Service) FindUserByEmail(email string) (domain.User, error) {
	state := s.store.Snapshot()
	for _, user := range state.Users {
		if strings.EqualFold(strings.TrimSpace(user.Email), strings.TrimSpace(email)) {
			return user, nil
		}
	}
	return domain.User{}, notFound("user")
}

func (s *Service) FindUserByID(id string) (domain.User, error) {
	state := s.store.Snapshot()
	for _, user := range state.Users {
		if user.ID == id {
			return user, nil
		}
	}
	return domain.User{}, notFound("user")
}

func (s *Service) Organization() domain.Organization { return s.store.Snapshot().Organization }

func (s *Service) Locations() []domain.Location {
	state := s.store.Snapshot()
	return append([]domain.Location(nil), state.Locations...)
}

func (s *Service) Resources() []domain.Resource {
	state := s.store.Snapshot()
	out := append([]domain.Resource(nil), state.Resources...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Pool == out[j].Pool {
			return out[i].Name < out[j].Name
		}
		return out[i].Pool < out[j].Pool
	})
	return out
}

func (s *Service) Offerings(publicOnly bool) []domain.Offering {
	state := s.store.Snapshot()
	out := make([]domain.Offering, 0, len(state.Offerings))
	for _, offering := range state.Offerings {
		if !offering.Active {
			continue
		}
		if publicOnly && !offering.PublicEnabled {
			continue
		}
		out = append(out, offering)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *Service) AddResource(input ResourceInput, actorID string) (domain.Resource, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Pool = strings.ToUpper(strings.TrimSpace(input.Pool))
	if input.Name == "" || input.Pool == "" {
		return domain.Resource{}, invalid("resource_fields_required", "Укажите название и пул ресурса")
	}
	if input.Capacity < 1 {
		input.Capacity = 1
	}
	if input.Capacity > 10000 {
		return domain.Resource{}, invalid("resource_capacity_too_large", "Вместимость ресурса не может превышать 10 000 в MVP")
	}
	if input.LocationID == "" {
		input.LocationID = "loc-astana"
	}
	if input.Type == "" {
		input.Type = domain.ResourceAsset
	}
	if !validResourceType(input.Type) {
		return domain.Resource{}, invalid("resource_type_invalid", "Выбран неподдерживаемый тип ресурса")
	}
	if input.Color == "" {
		input.Color = "#7868F2"
	}
	if !validHexColor(input.Color) {
		return domain.Resource{}, invalid("resource_color_invalid", "Цвет ресурса должен быть в формате #RRGGBB")
	}
	now := s.now()
	resource := domain.Resource{ID: newID("res"), LocationID: input.LocationID, Name: input.Name, Pool: input.Pool, Type: input.Type, Capacity: input.Capacity, Color: input.Color, Active: true, Notes: input.Notes, CreatedAt: now, UpdatedAt: now}
	err := s.store.Update(func(state *domain.State) error {
		if _, ok := findLocation(*state, resource.LocationID); !ok {
			return notFound("location")
		}
		state.Resources = append(state.Resources, resource)
		appendAudit(state, actorID, "resource.created", "resource", resource.ID, "Создан ресурс «"+resource.Name+"»", nil, now)
		return nil
	})
	return resource, err
}

func (s *Service) AddOffering(input OfferingInput, actorID string) (domain.Offering, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.ResourcePool = strings.ToUpper(strings.TrimSpace(input.ResourcePool))
	input.Category = strings.TrimSpace(input.Category)
	input.Description = strings.TrimSpace(input.Description)
	input.CancellationPolicy = strings.TrimSpace(input.CancellationPolicy)
	if input.Name == "" || input.ResourcePool == "" {
		return domain.Offering{}, invalid("offering_fields_required", "Укажите название и пул ресурсов")
	}
	if input.LocationID == "" {
		input.LocationID = "loc-astana"
	}
	if input.SchedulingMode == "" {
		input.SchedulingMode = domain.SchedulingRental
	}
	if !validSchedulingMode(input.SchedulingMode) {
		return domain.Offering{}, invalid("scheduling_mode_invalid", "Выбран неподдерживаемый тип бронирования")
	}
	if input.SchedulingMode == domain.SchedulingRequest {
		return domain.Offering{}, invalid("request_window_not_available", "Запрос диапазона появится после пилота; используйте запись, аренду или фиксированные сеансы")
	}
	if input.AllocationMode == "" {
		input.AllocationMode = domain.AllocationExclusive
	}
	if !validAllocationMode(input.AllocationMode) {
		return domain.Offering{}, invalid("allocation_mode_invalid", "Выбран неподдерживаемый способ использования ресурса")
	}
	if input.ConfirmationMode == "" {
		input.ConfirmationMode = domain.ConfirmationManagerApproval
	}
	if !validConfirmationMode(input.ConfirmationMode) {
		return domain.Offering{}, invalid("confirmation_mode_invalid", "Выбран неподдерживаемый способ подтверждения")
	}
	if input.PriceMode == "" {
		input.PriceMode = domain.PriceFixed
	}
	if !validPriceMode(input.PriceMode) {
		return domain.Offering{}, invalid("price_mode_invalid", "Выбрана неподдерживаемая модель цены")
	}
	if input.BasePrice < 0 {
		return domain.Offering{}, invalid("base_price_invalid", "Цена не может быть отрицательной")
	}
	if input.DepositPercent < 0 || input.DepositPercent > 100 {
		return domain.Offering{}, invalid("deposit_percent_invalid", "Предоплата должна быть от 0 до 100%")
	}
	if input.ConfirmationMode == domain.ConfirmationPaymentGated && input.DepositPercent == 0 {
		return domain.Offering{}, invalid("deposit_required", "Для подтверждения после предоплаты укажите процент предоплаты")
	}
	if input.PriceMode == domain.PriceCustomQuote && input.DepositPercent > 0 {
		return domain.Offering{}, invalid("custom_quote_deposit_invalid", "Предоплата для индивидуального расчёта задаётся после согласования цены")
	}
	if input.DurationMin <= 0 {
		input.DurationMin = 60
	}
	if input.MinDurationMin <= 0 {
		input.MinDurationMin = input.DurationMin
	}
	if input.MaxDurationMin < input.MinDurationMin {
		input.MaxDurationMin = input.MinDurationMin
	}
	if input.DurationMin < input.MinDurationMin || input.DurationMin > input.MaxDurationMin {
		return domain.Offering{}, invalid("default_duration_invalid", "Стандартная длительность должна находиться между минимумом и максимумом")
	}
	if input.DurationStepMin <= 0 {
		input.DurationStepMin = 30
	}
	if (input.DurationMin-input.MinDurationMin)%input.DurationStepMin != 0 {
		return domain.Offering{}, invalid("default_duration_step_mismatch", "Стандартная длительность должна соответствовать шагу длительности")
	}
	if input.StartStepMin <= 0 {
		input.StartStepMin = 30
	}
	if input.BufferBeforeMin < 0 || input.BufferAfterMin < 0 || input.MinLeadMin < 0 {
		return domain.Offering{}, invalid("time_rules_invalid", "Буферы и минимальный срок записи не могут быть отрицательными")
	}
	if input.MinGuests <= 0 {
		input.MinGuests = 1
	}
	if input.MaxGuests < input.MinGuests {
		input.MaxGuests = input.MinGuests
	}
	if input.ResourcesRequired <= 0 {
		input.ResourcesRequired = 1
	}
	if input.CapacityPerResource <= 0 {
		input.CapacityPerResource = input.MaxGuests
	}
	if input.MaxAdvanceDays <= 0 {
		input.MaxAdvanceDays = 60
	}
	if input.MaxAdvanceDays > 365 {
		return domain.Offering{}, invalid("booking_horizon_too_large", "Горизонт онлайн-записи не может превышать 365 дней в MVP")
	}
	if input.HoldDurationMin <= 0 {
		input.HoldDurationMin = 15
	}
	if input.Accent == "" {
		input.Accent = "violet"
	}
	if input.CancellationPolicy == "" {
		input.CancellationPolicy = "Условия отмены и переноса уточняются у площадки до подтверждения брони."
	}

	fixedSessions, err := normalizeFixedSessions(input.SchedulingMode, input.FixedSessions)
	if err != nil {
		return domain.Offering{}, err
	}
	now := s.now()
	offering := domain.Offering{
		ID: newID("off"), LocationID: input.LocationID, Name: input.Name, Category: input.Category,
		Description: input.Description, SchedulingMode: input.SchedulingMode, AllocationMode: input.AllocationMode,
		ConfirmationMode: input.ConfirmationMode, PriceMode: input.PriceMode, BasePrice: input.BasePrice,
		DepositPercent: input.DepositPercent, DurationMin: input.DurationMin, MinDurationMin: input.MinDurationMin,
		MaxDurationMin: input.MaxDurationMin, DurationStepMin: input.DurationStepMin, StartStepMin: input.StartStepMin,
		BufferBeforeMin: input.BufferBeforeMin, BufferAfterMin: input.BufferAfterMin, MinGuests: input.MinGuests,
		MaxGuests: input.MaxGuests, ResourcePool: input.ResourcePool, ResourcesRequired: input.ResourcesRequired,
		CapacityPerResource: input.CapacityPerResource, MinLeadMin: input.MinLeadMin, MaxAdvanceDays: input.MaxAdvanceDays,
		HoldDurationMin: input.HoldDurationMin, PublicEnabled: input.PublicEnabled, Active: true, Accent: input.Accent,
		FixedSessions: fixedSessions, CancellationPolicy: input.CancellationPolicy, CreatedAt: now, UpdatedAt: now,
	}
	err = s.store.Update(func(state *domain.State) error {
		if _, ok := findLocation(*state, offering.LocationID); !ok {
			return notFound("location")
		}
		hasPool := false
		for _, resource := range state.Resources {
			if resource.Active && resource.LocationID == offering.LocationID && resource.Pool == offering.ResourcePool {
				hasPool = true
				break
			}
		}
		if !hasPool {
			return invalid("resource_pool_empty", "В выбранном пуле нет активных ресурсов")
		}
		state.Offerings = append(state.Offerings, offering)
		appendAudit(state, actorID, "offering.created", "offering", offering.ID, "Создана услуга «"+offering.Name+"»", nil, now)
		return nil
	})
	return offering, err
}

func normalizeFixedSessions(mode domain.SchedulingMode, sessions []domain.FixedSession) ([]domain.FixedSession, error) {
	if mode != domain.SchedulingFixed {
		return nil, nil
	}
	if len(sessions) == 0 {
		return nil, invalid("fixed_sessions_required", "Добавьте хотя бы один день и время фиксированного сеанса")
	}
	seen := make(map[string]bool, len(sessions))
	out := make([]domain.FixedSession, 0, len(sessions))
	for _, session := range sessions {
		if session.Weekday < int(time.Sunday) || session.Weekday > int(time.Saturday) {
			return nil, invalid("fixed_session_weekday_invalid", "Некорректный день недели фиксированного сеанса")
		}
		parsed, err := time.Parse("15:04", strings.TrimSpace(session.Start))
		if err != nil {
			return nil, invalid("fixed_session_time_invalid", "Время фиксированного сеанса должно быть в формате ЧЧ:ММ")
		}
		start := parsed.Format("15:04")
		key := string(rune('0'+session.Weekday)) + "@" + start
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, domain.FixedSession{Weekday: session.Weekday, Start: start})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Weekday == out[j].Weekday {
			return out[i].Start < out[j].Start
		}
		return out[i].Weekday < out[j].Weekday
	})
	return out, nil
}

func validResourceType(value domain.ResourceType) bool {
	switch value {
	case domain.ResourcePerson, domain.ResourceSpace, domain.ResourceAsset, domain.ResourceEquipment, domain.ResourceVirtual:
		return true
	default:
		return false
	}
}

func validHexColor(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, char := range value[1:] {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

func validSchedulingMode(value domain.SchedulingMode) bool {
	return value == domain.SchedulingAppointment || value == domain.SchedulingRental || value == domain.SchedulingFixed || value == domain.SchedulingRequest
}

func validAllocationMode(value domain.AllocationMode) bool {
	return value == domain.AllocationExclusive || value == domain.AllocationShared
}

func validConfirmationMode(value domain.ConfirmationMode) bool {
	switch value {
	case domain.ConfirmationInstant, domain.ConfirmationManagerApproval, domain.ConfirmationPaymentGated, domain.ConfirmationApprovalAndPayment, domain.ConfirmationManualQuote:
		return true
	default:
		return false
	}
}

func validPriceMode(value domain.PriceMode) bool {
	switch value {
	case domain.PriceFixed, domain.PricePerPerson, domain.PricePerResource, domain.PricePerDuration, domain.PricePerResourceHour, domain.PriceCustomQuote:
		return true
	default:
		return false
	}
}

func newID(prefix string) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return prefix + "-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	return prefix + "-" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf))
}

func appendAudit(state *domain.State, actorID, action, entity, entityID, summary string, metadata map[string]any, at time.Time) {
	state.AuditEvents = append(state.AuditEvents, domain.AuditEvent{ID: newID("audit"), ActorID: actorID, Action: action, Entity: entity, EntityID: entityID, Summary: summary, Metadata: metadata, CreatedAt: at})
	if len(state.AuditEvents) > 2000 {
		state.AuditEvents = state.AuditEvents[len(state.AuditEvents)-2000:]
	}
}

func findLocation(state domain.State, id string) (domain.Location, bool) {
	for _, item := range state.Locations {
		if item.ID == id {
			return item, true
		}
	}
	return domain.Location{}, false
}
func findResource(state domain.State, id string) (domain.Resource, bool) {
	for _, item := range state.Resources {
		if item.ID == id {
			return item, true
		}
	}
	return domain.Resource{}, false
}
func findOffering(state domain.State, id string) (domain.Offering, bool) {
	for _, item := range state.Offerings {
		if item.ID == id {
			return item, true
		}
	}
	return domain.Offering{}, false
}
func findCustomer(state domain.State, id string) (domain.Customer, bool) {
	for _, item := range state.Customers {
		if item.ID == id {
			return item, true
		}
	}
	return domain.Customer{}, false
}
func findBooking(state domain.State, id string) (domain.Booking, int, bool) {
	for i, item := range state.Bookings {
		if item.ID == id {
			return item, i, true
		}
	}
	return domain.Booking{}, -1, false
}
