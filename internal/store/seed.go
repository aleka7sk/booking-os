package store

import (
	"fmt"
	"time"

	"github.com/aleka7sk/booking-os/internal/domain"
	"github.com/aleka7sk/booking-os/internal/security"
)

func DemoSeed(adminEmail, adminPassword string) Seeder {
	return func() (domain.State, error) {
		now := time.Now().UTC().Truncate(time.Second)
		loc, err := time.LoadLocation("Asia/Almaty")
		if err != nil {
			loc = time.FixedZone("UTC+5", 5*60*60)
		}
		localNow := now.In(loc)
		today := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, loc)

		salt, hash, err := security.HashPassword(adminPassword)
		if err != nil {
			return domain.State{}, err
		}

		weeklyHours := map[string][]domain.Hours{}
		for _, day := range []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"} {
			weeklyHours[day] = []domain.Hours{{Start: "10:00", End: "23:30"}}
		}

		resources := []domain.Resource{
			{ID: "res-lane-1", LocationID: "loc-astana", Name: "Дорожка 1", Pool: "BOWLING", Type: domain.ResourceAsset, Capacity: 1, Color: "#7C6CF2", Active: true, CreatedAt: now, UpdatedAt: now},
			{ID: "res-lane-2", LocationID: "loc-astana", Name: "Дорожка 2", Pool: "BOWLING", Type: domain.ResourceAsset, Capacity: 1, Color: "#9184F5", Active: true, CreatedAt: now, UpdatedAt: now},
			{ID: "res-lane-3", LocationID: "loc-astana", Name: "Дорожка 3", Pool: "BOWLING", Type: domain.ResourceAsset, Capacity: 1, Color: "#AA9FFA", Active: true, CreatedAt: now, UpdatedAt: now},
			{ID: "res-lane-4", LocationID: "loc-astana", Name: "Дорожка 4", Pool: "BOWLING", Type: domain.ResourceAsset, Capacity: 1, Color: "#C3BBFC", Active: true, CreatedAt: now, UpdatedAt: now},
			{ID: "res-vip-1", LocationID: "loc-astana", Name: "VIP-комната Noir", Pool: "VIP_ROOM", Type: domain.ResourceSpace, Capacity: 1, Color: "#E57A65", Active: true, CreatedAt: now, UpdatedAt: now},
			{ID: "res-vip-2", LocationID: "loc-astana", Name: "VIP-комната Pearl", Pool: "VIP_ROOM", Type: domain.ResourceSpace, Capacity: 1, Color: "#EEA192", Active: true, CreatedAt: now, UpdatedAt: now},
			{ID: "res-studio-1", LocationID: "loc-astana", Name: "Студия Light Hall", Pool: "PHOTO_STUDIO", Type: domain.ResourceSpace, Capacity: 1, Color: "#39A98F", Active: true, CreatedAt: now, UpdatedAt: now},
			{ID: "res-workshop-1", LocationID: "loc-astana", Name: "Творческий зал", Pool: "WORKSHOP", Type: domain.ResourceSpace, Capacity: 12, Color: "#E5B84B", Active: true, CreatedAt: now, UpdatedAt: now},
		}

		offerings := []domain.Offering{
			{
				ID: "off-bowling", LocationID: "loc-astana", Name: "Боулинг", Category: "Активный отдых",
				Description: "Дорожка для компании до 6 гостей. Обувь включена.", SchedulingMode: domain.SchedulingRental,
				AllocationMode: domain.AllocationExclusive, ConfirmationMode: domain.ConfirmationApprovalAndPayment,
				PriceMode: domain.PricePerResourceHour, BasePrice: 7000, DepositPercent: 30, DurationMin: 120,
				MinDurationMin: 60, MaxDurationMin: 240, DurationStepMin: 60, StartStepMin: 30,
				BufferBeforeMin: 0, BufferAfterMin: 15, MinGuests: 1, MaxGuests: 24,
				ResourcePool: "BOWLING", ResourcesRequired: 1, CapacityPerResource: 6, MinLeadMin: 30,
				MaxAdvanceDays: 45, HoldDurationMin: 15, PublicEnabled: true, Active: true, Accent: "violet",
				CancellationPolicy: "Бесплатная отмена не позднее чем за 24 часа. После этого предоплата не возвращается.",
				CreatedAt:          now, UpdatedAt: now,
			},
			{
				ID: "off-vip", LocationID: "loc-astana", Name: "VIP-комната", Category: "Частные события",
				Description: "Приватная комната с караоке, экраном и обслуживанием.", SchedulingMode: domain.SchedulingRental,
				AllocationMode: domain.AllocationExclusive, ConfirmationMode: domain.ConfirmationApprovalAndPayment,
				PriceMode: domain.PricePerDuration, BasePrice: 22000, DepositPercent: 50, DurationMin: 120,
				MinDurationMin: 120, MaxDurationMin: 360, DurationStepMin: 60, StartStepMin: 30,
				BufferBeforeMin: 30, BufferAfterMin: 30, MinGuests: 2, MaxGuests: 16,
				ResourcePool: "VIP_ROOM", ResourcesRequired: 1, CapacityPerResource: 16, MinLeadMin: 120,
				MaxAdvanceDays: 90, HoldDurationMin: 20, PublicEnabled: true, Active: true, Accent: "coral",
				CancellationPolicy: "Бесплатная отмена не позднее чем за 48 часов. Индивидуальные заказы рассчитываются отдельно.",
				CreatedAt:          now, UpdatedAt: now,
			},
			{
				ID: "off-studio", LocationID: "loc-astana", Name: "Фотостудия Light Hall", Category: "Аренда пространства",
				Description: "Светлый зал, циклорама и базовое оборудование.", SchedulingMode: domain.SchedulingRental,
				AllocationMode: domain.AllocationExclusive, ConfirmationMode: domain.ConfirmationInstant,
				PriceMode: domain.PricePerDuration, BasePrice: 18000, DepositPercent: 0, DurationMin: 60,
				MinDurationMin: 60, MaxDurationMin: 300, DurationStepMin: 60, StartStepMin: 30,
				BufferBeforeMin: 15, BufferAfterMin: 15, MinGuests: 1, MaxGuests: 12,
				ResourcePool: "PHOTO_STUDIO", ResourcesRequired: 1, CapacityPerResource: 12, MinLeadMin: 60,
				MaxAdvanceDays: 60, HoldDurationMin: 10, PublicEnabled: true, Active: true, Accent: "emerald",
				CancellationPolicy: "Бесплатная отмена не позднее чем за 24 часа.", CreatedAt: now, UpdatedAt: now,
			},
			{
				ID: "off-workshop", LocationID: "loc-astana", Name: "Гончарный мастер-класс", Category: "Впечатления",
				Description: "Групповое занятие для новичков. Все материалы включены.", SchedulingMode: domain.SchedulingFixed,
				AllocationMode: domain.AllocationShared, ConfirmationMode: domain.ConfirmationPaymentGated,
				PriceMode: domain.PricePerPerson, BasePrice: 9000, DepositPercent: 100, DurationMin: 90,
				MinDurationMin: 90, MaxDurationMin: 90, DurationStepMin: 90, StartStepMin: 30,
				BufferBeforeMin: 30, BufferAfterMin: 30, MinGuests: 1, MaxGuests: 12,
				ResourcePool: "WORKSHOP", ResourcesRequired: 1, CapacityPerResource: 1, MinLeadMin: 180,
				MaxAdvanceDays: 30, HoldDurationMin: 15, PublicEnabled: true, Active: true, Accent: "amber",
				FixedSessions:      []domain.FixedSession{{Weekday: int(time.Tuesday), Start: "19:00"}, {Weekday: int(time.Thursday), Start: "19:00"}, {Weekday: int(time.Saturday), Start: "15:00"}},
				CancellationPolicy: "Перенос возможен не позднее чем за 12 часов до начала.", CreatedAt: now, UpdatedAt: now,
			},
		}

		customers := []domain.Customer{
			{ID: "cus-aigerim", Name: "Айгерим С.", Phone: "+7 701 482 19 27", Language: "ru", CreatedAt: now, UpdatedAt: now},
			{ID: "cus-daniyar", Name: "Данияр К.", Phone: "+7 707 113 42 80", Language: "ru", CreatedAt: now, UpdatedAt: now},
			{ID: "cus-madina", Name: "Мадина А.", Phone: "+7 775 901 05 31", Language: "ru", CreatedAt: now, UpdatedAt: now},
			{ID: "cus-arman", Name: "Арман Т.", Phone: "+7 747 665 18 44", Language: "ru", CreatedAt: now, UpdatedAt: now},
			{ID: "cus-saule", Name: "Сауле Н.", Phone: "+7 702 303 71 62", Language: "ru", CreatedAt: now, UpdatedAt: now},
		}

		at := func(dayOffset, hour, minute int) time.Time {
			return today.AddDate(0, 0, dayOffset).Add(time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute).UTC()
		}
		confirmedAt := now.Add(-2 * time.Hour)
		holdExpiry := now.Add(17 * time.Minute)
		bookings := []domain.Booking{
			booking("book-1", "B-1041", "off-bowling", "cus-aigerim", domain.BookingConfirmed, domain.PaymentPartiallyPaid, domain.SourceTwoGIS, at(0, 12, 0), at(0, 14, 0), 6, 14000, 4200, 4200, &confirmedAt, nil, "День рождения, нужна детская дорожка", now.Add(-24*time.Hour)),
			booking("book-2", "B-1042", "off-studio", "cus-madina", domain.BookingConfirmed, domain.PaymentPaid, domain.SourceInstagram, at(0, 15, 0), at(0, 16, 0), 4, 18000, 0, 18000, &confirmedAt, nil, "Контент-съёмка", now.Add(-18*time.Hour)),
			booking("book-3", "B-1043", "off-workshop", "cus-saule", domain.BookingConfirmed, domain.PaymentPaid, domain.SourcePublicPage, at(0, 19, 0), at(0, 20, 30), 4, 36000, 36000, 36000, &confirmedAt, nil, "", now.Add(-12*time.Hour)),
			booking("book-4", "B-1044", "off-bowling", "cus-daniyar", domain.BookingRequested, domain.PaymentUnpaid, domain.SourceWhatsApp, at(0, 20, 0), at(0, 22, 0), 8, 28000, 8400, 0, nil, nil, "Просит две соседние дорожки", now.Add(-35*time.Minute)),
			booking("book-5", "B-1045", "off-bowling", "cus-arman", domain.BookingHeld, domain.PaymentUnpaid, domain.SourcePublicPage, at(0, 21, 0), at(0, 23, 0), 5, 14000, 4200, 0, nil, &holdExpiry, "Ожидаем предоплату", now.Add(-4*time.Minute)),
			booking("book-6", "B-1046", "off-vip", "cus-aigerim", domain.BookingConfirmed, domain.PaymentPartiallyPaid, domain.SourcePhone, at(1, 20, 0), at(1, 23, 0), 10, 66000, 33000, 33000, &confirmedAt, nil, "Юбилей", now.Add(-48*time.Hour)),
		}
		for i := range bookings {
			bookings[i].OrganizationID = "org-demo"
			bookings[i].LocationID = "loc-astana"
			bookings[i].PublicToken = fmt.Sprintf("demo-%s", bookings[i].ID)
		}

		allocations := []domain.Allocation{
			allocation("alloc-1", "book-1", "res-lane-1", at(0, 12, 0), at(0, 14, 15), 1, now),
			allocation("alloc-2", "book-2", "res-studio-1", at(0, 14, 45), at(0, 16, 15), 1, now),
			allocation("alloc-3", "book-3", "res-workshop-1", at(0, 18, 30), at(0, 21, 0), 4, now),
			allocation("alloc-4", "book-5", "res-lane-2", at(0, 21, 0), at(0, 23, 15), 1, now),
			allocation("alloc-5", "book-6", "res-vip-1", at(1, 19, 30), at(1, 23, 30), 1, now),
		}

		return domain.State{
			SchemaVersion: 1,
			Organization: domain.Organization{
				ID: "org-demo", Name: "ASTRA Space", LegalName: "ТОО ASTRA Space", Currency: "KZT",
				Timezone: "Asia/Almaty", Phone: "+7 7172 55 48 20", Email: "hello@astraspace.kz",
				BrandColor: "#7868F2", PublicSlug: "astra-space", PublicTitle: "Выберите своё впечатление",
				PublicSubtitle: "Забронируйте время онлайн — без звонков и ожидания ответа.", CreatedAt: now,
			},
			Users:     []domain.User{{ID: "user-owner", Name: "Алишер", Email: adminEmail, Role: "OWNER", PasswordSalt: salt, PasswordHash: hash, CreatedAt: now}},
			Locations: []domain.Location{{ID: "loc-astana", Name: "ASTRA Space · Esil", Address: "Астана, проспект Мәңгілік Ел, 55", Timezone: "Asia/Almaty", Phone: "+7 7172 55 48 20", Active: true, WeeklyHours: weeklyHours, CreatedAt: now}},
			Resources: resources, Offerings: offerings, Customers: customers, Bookings: bookings, Allocations: allocations,
			AuditEvents:   []domain.AuditEvent{{ID: "audit-seed", ActorID: "system", Action: "workspace.seeded", Entity: "organization", EntityID: "org-demo", Summary: "Создано демонстрационное пространство", CreatedAt: now}},
			NextReference: 1047,
		}, nil
	}
}

func booking(id, ref, offeringID, customerID string, status domain.BookingStatus, payment domain.PaymentStatus, source domain.Source, start, end time.Time, guests int, total, deposit, paid int64, confirmedAt, holdExpiresAt *time.Time, notes string, createdAt time.Time) domain.Booking {
	return domain.Booking{
		ID: id, Reference: ref, OfferingID: offeringID, CustomerID: customerID, Status: status,
		PaymentStatus: payment, AttendanceStatus: domain.AttendanceNotStarted, Source: source,
		StartAt: start, EndAt: end, OccupiedStartAt: start, OccupiedEndAt: end, GuestCount: guests,
		TotalAmount: total, DepositAmount: deposit, PaidAmount: paid, HoldExpiresAt: holdExpiresAt,
		Notes: notes, CancellationSnapshot: "Условия зафиксированы при создании брони", PriceSnapshot: []domain.PriceLine{{Label: "Итого", Quantity: 1, UnitAmount: total, Amount: total}}, CreatedBy: "user-owner",
		CreatedAt: createdAt, UpdatedAt: createdAt, ConfirmedAt: confirmedAt,
	}
}

func allocation(id, bookingID, resourceID string, start, end time.Time, quantity int, createdAt time.Time) domain.Allocation {
	return domain.Allocation{ID: id, BookingID: bookingID, ResourceID: resourceID, StartAt: start, EndAt: end, Quantity: quantity, Status: domain.AllocationActive, CreatedAt: createdAt}
}
