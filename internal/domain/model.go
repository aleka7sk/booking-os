package domain

import "time"

type BookingStatus string
type PaymentStatus string
type AttendanceStatus string
type SchedulingMode string
type AllocationMode string
type ConfirmationMode string
type PriceMode string
type Source string
type ResourceType string
type AllocationStatus string

const (
	BookingDraft     BookingStatus = "DRAFT"
	BookingRequested BookingStatus = "REQUESTED"
	BookingHeld      BookingStatus = "HELD"
	BookingConfirmed BookingStatus = "CONFIRMED"
	BookingCancelled BookingStatus = "CANCELLED"
	BookingRejected  BookingStatus = "REJECTED"
	BookingExpired   BookingStatus = "EXPIRED"
	BookingCompleted BookingStatus = "COMPLETED"
)

const (
	PaymentNotRequired       PaymentStatus = "NOT_REQUIRED"
	PaymentUnpaid            PaymentStatus = "UNPAID"
	PaymentPartiallyPaid     PaymentStatus = "PARTIALLY_PAID"
	PaymentPaid              PaymentStatus = "PAID"
	PaymentFailed            PaymentStatus = "PAYMENT_FAILED"
	PaymentPartiallyRefunded PaymentStatus = "PARTIALLY_REFUNDED"
	PaymentRefunded          PaymentStatus = "REFUNDED"
)

const (
	AttendanceNotStarted AttendanceStatus = "NOT_STARTED"
	AttendanceCheckedIn  AttendanceStatus = "CHECKED_IN"
	AttendanceAttended   AttendanceStatus = "ATTENDED"
	AttendanceNoShow     AttendanceStatus = "NO_SHOW"
)

const (
	SchedulingAppointment SchedulingMode = "APPOINTMENT"
	SchedulingRental      SchedulingMode = "RENTAL"
	SchedulingFixed       SchedulingMode = "FIXED_SESSION"
	SchedulingRequest     SchedulingMode = "REQUEST_WINDOW"
)

const (
	AllocationExclusive AllocationMode = "EXCLUSIVE"
	AllocationShared    AllocationMode = "SHARED_CAPACITY"
)

const (
	ConfirmationInstant            ConfirmationMode = "INSTANT"
	ConfirmationManagerApproval    ConfirmationMode = "MANAGER_APPROVAL"
	ConfirmationPaymentGated       ConfirmationMode = "PAYMENT_GATED"
	ConfirmationApprovalAndPayment ConfirmationMode = "APPROVAL_AND_PAYMENT"
	ConfirmationManualQuote        ConfirmationMode = "MANUAL_QUOTE"
)

const (
	PriceFixed           PriceMode = "FIXED"
	PricePerPerson       PriceMode = "PER_PERSON"
	PricePerResource     PriceMode = "PER_RESOURCE"
	PricePerDuration     PriceMode = "PER_DURATION"
	PricePerResourceHour PriceMode = "PER_RESOURCE_DURATION"
	PricePartyTier       PriceMode = "PARTY_TIER"
	PriceCustomQuote     PriceMode = "CUSTOM_QUOTE"
)

const (
	SourcePublicPage Source = "PUBLIC_BOOKING_PAGE"
	SourceWhatsApp   Source = "WHATSAPP"
	SourcePhone      Source = "PHONE"
	SourceInstagram  Source = "INSTAGRAM"
	SourceTwoGIS     Source = "TWO_GIS"
	SourceWalkIn     Source = "WALK_IN"
	SourceManager    Source = "MANAGER"
	SourceImport     Source = "IMPORT"
	SourceAPI        Source = "API"
	SourceOther      Source = "OTHER"
)

const (
	ResourcePerson    ResourceType = "PERSON"
	ResourceSpace     ResourceType = "SPACE"
	ResourceAsset     ResourceType = "ASSET"
	ResourceEquipment ResourceType = "EQUIPMENT"
	ResourceVirtual   ResourceType = "VIRTUAL"
)

const (
	AllocationActive    AllocationStatus = "ACTIVE"
	AllocationCancelled AllocationStatus = "CANCELLED"
	AllocationExpired   AllocationStatus = "EXPIRED"
)

type State struct {
	SchemaVersion  int             `json:"schemaVersion"`
	Organization   Organization    `json:"organization"`
	Users          []User          `json:"users"`
	Locations      []Location      `json:"locations"`
	Resources      []Resource      `json:"resources"`
	Offerings      []Offering      `json:"offerings"`
	Customers      []Customer      `json:"customers"`
	Bookings       []Booking       `json:"bookings"`
	Allocations    []Allocation    `json:"allocations"`
	ResourceBlocks []ResourceBlock `json:"resourceBlocks"`
	AuditEvents    []AuditEvent    `json:"auditEvents"`
	NextReference  int             `json:"nextReference"`
}

type Organization struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	LegalName      string    `json:"legalName,omitempty"`
	Currency       string    `json:"currency"`
	Timezone       string    `json:"timezone"`
	Phone          string    `json:"phone"`
	Email          string    `json:"email"`
	BrandColor     string    `json:"brandColor"`
	PublicSlug     string    `json:"publicSlug"`
	PublicTitle    string    `json:"publicTitle"`
	PublicSubtitle string    `json:"publicSubtitle"`
	CreatedAt      time.Time `json:"createdAt"`
}

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	PasswordSalt string    `json:"passwordSalt"`
	PasswordHash string    `json:"passwordHash"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Location struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Address     string             `json:"address"`
	Timezone    string             `json:"timezone"`
	Phone       string             `json:"phone"`
	Active      bool               `json:"active"`
	WeeklyHours map[string][]Hours `json:"weeklyHours"`
	CreatedAt   time.Time          `json:"createdAt"`
}

type Hours struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type Resource struct {
	ID         string       `json:"id"`
	LocationID string       `json:"locationId"`
	Name       string       `json:"name"`
	Pool       string       `json:"pool"`
	Type       ResourceType `json:"type"`
	Capacity   int          `json:"capacity"`
	Color      string       `json:"color"`
	Active     bool         `json:"active"`
	Notes      string       `json:"notes,omitempty"`
	CreatedAt  time.Time    `json:"createdAt"`
	UpdatedAt  time.Time    `json:"updatedAt"`
}

type PriceTier struct {
	MinGuests int   `json:"minGuests"`
	MaxGuests int   `json:"maxGuests"`
	Amount    int64 `json:"amount"`
}

type FixedSession struct {
	Weekday int    `json:"weekday"`
	Start   string `json:"start"`
}

type Offering struct {
	ID                  string           `json:"id"`
	LocationID          string           `json:"locationId"`
	Name                string           `json:"name"`
	Category            string           `json:"category"`
	Description         string           `json:"description"`
	SchedulingMode      SchedulingMode   `json:"schedulingMode"`
	AllocationMode      AllocationMode   `json:"allocationMode"`
	ConfirmationMode    ConfirmationMode `json:"confirmationMode"`
	PriceMode           PriceMode        `json:"priceMode"`
	BasePrice           int64            `json:"basePrice"`
	DepositPercent      int              `json:"depositPercent"`
	DurationMin         int              `json:"durationMin"`
	MinDurationMin      int              `json:"minDurationMin"`
	MaxDurationMin      int              `json:"maxDurationMin"`
	DurationStepMin     int              `json:"durationStepMin"`
	StartStepMin        int              `json:"startStepMin"`
	BufferBeforeMin     int              `json:"bufferBeforeMin"`
	BufferAfterMin      int              `json:"bufferAfterMin"`
	MinGuests           int              `json:"minGuests"`
	MaxGuests           int              `json:"maxGuests"`
	ResourcePool        string           `json:"resourcePool"`
	ResourcesRequired   int              `json:"resourcesRequired"`
	CapacityPerResource int              `json:"capacityPerResource"`
	MinLeadMin          int              `json:"minLeadMin"`
	MaxAdvanceDays      int              `json:"maxAdvanceDays"`
	HoldDurationMin     int              `json:"holdDurationMin"`
	PublicEnabled       bool             `json:"publicEnabled"`
	Active              bool             `json:"active"`
	Accent              string           `json:"accent"`
	PriceTiers          []PriceTier      `json:"priceTiers,omitempty"`
	FixedSessions       []FixedSession   `json:"fixedSessions,omitempty"`
	CancellationPolicy  string           `json:"cancellationPolicy"`
	CreatedAt           time.Time        `json:"createdAt"`
	UpdatedAt           time.Time        `json:"updatedAt"`
}

type Customer struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email,omitempty"`
	Language  string    `json:"language"`
	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Booking struct {
	ID                   string           `json:"id"`
	Reference            string           `json:"reference"`
	OrganizationID       string           `json:"organizationId"`
	LocationID           string           `json:"locationId"`
	OfferingID           string           `json:"offeringId"`
	CustomerID           string           `json:"customerId"`
	Status               BookingStatus    `json:"status"`
	PaymentStatus        PaymentStatus    `json:"paymentStatus"`
	AttendanceStatus     AttendanceStatus `json:"attendanceStatus"`
	Source               Source           `json:"source"`
	StartAt              time.Time        `json:"startAt"`
	EndAt                time.Time        `json:"endAt"`
	OccupiedStartAt      time.Time        `json:"occupiedStartAt"`
	OccupiedEndAt        time.Time        `json:"occupiedEndAt"`
	GuestCount           int              `json:"guestCount"`
	TotalAmount          int64            `json:"totalAmount"`
	DepositAmount        int64            `json:"depositAmount"`
	PaidAmount           int64            `json:"paidAmount"`
	HoldExpiresAt        *time.Time       `json:"holdExpiresAt,omitempty"`
	Notes                string           `json:"notes,omitempty"`
	InternalNotes        string           `json:"internalNotes,omitempty"`
	CancellationReason   string           `json:"cancellationReason,omitempty"`
	CancellationSnapshot string           `json:"cancellationSnapshot"`
	PolicyAcceptedAt     *time.Time       `json:"policyAcceptedAt,omitempty"`
	PriceSnapshot        []PriceLine      `json:"priceSnapshot"`
	PublicToken          string           `json:"publicToken"`
	CreatedBy            string           `json:"createdBy"`
	CreatedAt            time.Time        `json:"createdAt"`
	UpdatedAt            time.Time        `json:"updatedAt"`
	ConfirmedAt          *time.Time       `json:"confirmedAt,omitempty"`
	CancelledAt          *time.Time       `json:"cancelledAt,omitempty"`
	CompletedAt          *time.Time       `json:"completedAt,omitempty"`
}

type PriceLine struct {
	Label      string `json:"label"`
	Quantity   int    `json:"quantity"`
	UnitAmount int64  `json:"unitAmount"`
	Amount     int64  `json:"amount"`
}

type Allocation struct {
	ID         string           `json:"id"`
	BookingID  string           `json:"bookingId"`
	ResourceID string           `json:"resourceId"`
	StartAt    time.Time        `json:"startAt"`
	EndAt      time.Time        `json:"endAt"`
	Quantity   int              `json:"quantity"`
	Status     AllocationStatus `json:"status"`
	CreatedAt  time.Time        `json:"createdAt"`
}

type ResourceBlock struct {
	ID         string    `json:"id"`
	ResourceID string    `json:"resourceId"`
	StartAt    time.Time `json:"startAt"`
	EndAt      time.Time `json:"endAt"`
	Reason     string    `json:"reason"`
	CreatedAt  time.Time `json:"createdAt"`
}

type AuditEvent struct {
	ID        string         `json:"id"`
	ActorID   string         `json:"actorId"`
	Action    string         `json:"action"`
	Entity    string         `json:"entity"`
	EntityID  string         `json:"entityId"`
	Summary   string         `json:"summary"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

func (s BookingStatus) OccupiesResource() bool {
	return s == BookingHeld || s == BookingConfirmed || s == BookingCompleted
}
