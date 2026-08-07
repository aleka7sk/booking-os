# Booking OS API

Все даты передаются в RFC 3339. Денежные суммы — целые тенге в текущем MVP.

## Ошибки

```json
{
  "error": {
    "code": "resources_unavailable",
    "message": "На выбранное время недостаточно свободных ресурсов"
  }
}
```

Типовые HTTP-коды:

- `400` — невалидный ввод или переход статуса;
- `401` — нет действующей сессии;
- `403` — операция запрещена;
- `404` — сущность не найдена;
- `409` — ресурсный/capacity-конфликт;
- `500` — внутренняя ошибка.

## Auth

### `POST /api/auth/login`

```json
{
  "email": "owner@booking.local",
  "password": "demo1234"
}
```

Устанавливает cookie `booking_os_session`.

### `POST /api/auth/logout`

Удаляет сессию.

### `GET /api/auth/me`

Возвращает пользователя, организацию и филиалы.

## Availability

### `POST /api/availability/search`
### `POST /api/public/availability`

```json
{
  "offeringId": "off-bowling",
  "date": "2026-08-08",
  "guestCount": 8,
  "durationMin": 120
}
```

Ответ:

```json
[
  {
    "startAt": "2026-08-08T05:00:00Z",
    "endAt": "2026-08-08T07:00:00Z",
    "durationMin": 120,
    "available": true,
    "totalAmount": 28000,
    "depositAmount": 8400,
    "priceLines": [
      {
        "label": "Ресурс × время",
        "quantity": 2,
        "unitAmount": 7000,
        "amount": 28000
      }
    ],
    "resourceIds": ["res-lane-1", "res-lane-2"]
  }
]
```

Resource IDs в ответе являются предварительным планом. При создании система повторно проверяет занятость.

## Bookings

### `POST /api/bookings`
### `POST /api/public/bookings`

```json
{
  "offeringId": "off-bowling",
  "customerName": "Айдос",
  "customerPhone": "+7 701 000 00 00",
  "customerEmail": "",
  "startAt": "2026-08-08T15:00:00Z",
  "durationMin": 120,
  "guestCount": 8,
  "source": "TWO_GIS",
  "notes": "Нужны соседние дорожки",
  "internalNotes": "",
  "requestOnly": false,
  "paidAmount": 0,
  "acceptedPolicy": true
}
```

Для public endpoint `source` принудительно становится `PUBLIC_BOOKING_PAGE`, `acceptedPolicy=true` обязательно, а статус определяется confirmation mode услуги. Момент согласия и текст правил сохраняются внутри брони.

### `GET /api/bookings`

Query:

```text
from=RFC3339
to=RFC3339
status=REQUESTED
```

### `GET /api/bookings/{id}`

Возвращает booking, offering, customer и allocated resources.

### `POST /api/bookings/{id}/confirm`

- `REQUESTED` получает allocations;
- при недостающем депозите переходит в `HELD`;
- иначе переходит в `CONFIRMED`.

### `POST /api/bookings/{id}/payment`

```json
{
  "amount": 8400,
  "note": "Kaspi перевод"
}
```

### `POST /api/bookings/{id}/reschedule`

```json
{
  "startAt": "2026-08-09T15:00:00Z",
  "durationMin": 120,
  "reason": "Просьба клиента"
}
```

### `POST /api/bookings/{id}/cancel`

```json
{
  "reason": "Клиент отменил"
}
```

### Остальные actions

```text
POST /api/bookings/{id}/reject
POST /api/bookings/{id}/complete
POST /api/bookings/{id}/no-show
```

## Public

```text
GET  /api/public/profile
GET  /api/public/offerings
GET  /api/public/bookings/{publicToken}
```

Публичный token генерируется криптографически случайно и не раскрывает внутренний ID.

## Configuration

```text
GET  /api/locations
GET  /api/resources
POST /api/resources
GET  /api/offerings
POST /api/offerings
```

## Reporting

```text
GET /api/dashboard
GET /api/calendar?from=...&to=...
GET /api/audit
```

## Fixed sessions при создании услуги

Для `schedulingMode=FIXED_SESSION` необходимо передать хотя бы один повторяющийся сеанс:

```json
{
  "name": "Гончарный мастер-класс",
  "locationId": "loc-astana",
  "resourcePool": "WORKSHOP",
  "schedulingMode": "FIXED_SESSION",
  "allocationMode": "SHARED_CAPACITY",
  "confirmationMode": "PAYMENT_GATED",
  "priceMode": "PER_PERSON",
  "basePrice": 9000,
  "depositPercent": 100,
  "durationMin": 90,
  "minDurationMin": 90,
  "maxDurationMin": 90,
  "durationStepMin": 90,
  "minGuests": 1,
  "maxGuests": 12,
  "maxAdvanceDays": 30,
  "holdDurationMin": 15,
  "publicEnabled": true,
  "fixedSessions": [
    {"weekday": 2, "start": "19:00"},
    {"weekday": 4, "start": "19:00"},
    {"weekday": 6, "start": "15:00"}
  ]
}
```

`weekday` использует стандарт Go/JavaScript: воскресенье — `0`, понедельник — `1`, суббота — `6`. Дубликаты нормализуются.
