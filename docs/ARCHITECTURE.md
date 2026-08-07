# Архитектура Booking OS MVP

## 1. Цель архитектуры

MVP должен одновременно выполнить две задачи:

1. дать бизнесу работающий продукт, который можно запустить одним процессом;
2. не смешать пользовательский интерфейс с критическими правилами доступности, чтобы позже заменить локальное хранилище PostgreSQL без переписывания домена.

Поэтому приложение реализовано как **modular monolith** на Go.

## 2. Слои

### `internal/domain`

Не содержит HTTP и файловой системы. Здесь находятся:

- `Organization`, `Location`;
- `Offering`;
- `Resource`;
- `Booking`;
- `Allocation`;
- `Customer`;
- `ResourceBlock`;
- `AuditEvent`;
- статусы брони, оплаты и посещения.

### `internal/app`

Оркестрирует бизнес-операции:

- поиск доступности;
- нормализацию длительности и количества гостей;
- подбор ресурсов;
- расчёт цены и депозита;
- создание и изменение броней;
- переходы статусов;
- календарь и dashboard;
- аудит.

Все критические операции выполняются внутри `JSONStore.Update`, то есть над одной рабочей копией состояния под mutex. Только после успешной валидации копия атомарно заменяет прежнее состояние.

### `internal/store`

`JSONStore`:

- сериализует конкурентные записи;
- клонирует состояние перед транзакцией;
- пишет временный файл;
- атомарно выполняет `rename`;
- откатывает in-memory состояние при ошибке диска.

Это осознанная граница MVP. Формат не предназначен для нескольких реплик приложения.

### `internal/httpapi`

- REST-маршруты на стандартном `net/http`;
- JSON validation с `DisallowUnknownFields`;
- auth middleware;
- безопасные cookie;
- security headers;
- embedded frontend и SPA fallback.

### `internal/httpapi/web`

Vanilla ES modules без runtime-зависимостей:

- административная PWA;
- публичный мастер бронирования;
- premium responsive design;
- API adapter;
- UI primitives.

## 3. Модель времени

- абсолютное время сохраняется как UTC;
- филиал хранит IANA timezone;
- поиск слотов строится в локальном времени филиала;
- пользовательская длительность и занятость ресурса различаются;
- `OccupiedStartAt` и `OccupiedEndAt` включают буферы.

Пример:

```text
клиент видит:      20:00–22:00
buffer before:       15 минут
buffer after:        30 минут
ресурс занят:      19:45–22:30
```

## 4. Движок доступности

Для каждого кандидата времени система:

1. проверяет расписание точки и услуги;
2. применяет lead time и booking horizon;
3. вычисляет фактический интервал с буферами;
4. собирает ресурсы нужного пула;
5. исключает блокировки;
6. считает пересекающиеся active allocations;
7. игнорирует просроченные holds;
8. распределяет эксклюзивные ресурсы или shared capacity;
9. рассчитывает price snapshot;
10. возвращает слот и предварительный набор resource IDs.

Результат availability search является рекомендацией. Перед созданием брони подбор выполняется повторно внутри атомарной записи.

## 5. Allocation как источник правды

`Booking` описывает коммерческую договорённость. `Allocation` описывает занятость ресурса.

```text
Booking B-1047
  ├─ Allocation → Дорожка 1, 20:00–22:15
  └─ Allocation → Дорожка 2, 20:00–22:15
```

Для shared capacity `Quantity` показывает расход вместимости.

```text
Творческий зал capacity=12
  ├─ Booking A quantity=4
  ├─ Booking B quantity=5
  └─ осталось 3
```

## 6. Статусы

### Booking lifecycle

```text
DRAFT
REQUESTED
HELD
CONFIRMED
CANCELLED
REJECTED
EXPIRED
COMPLETED
```

### Payment lifecycle

```text
NOT_REQUIRED
UNPAID
PARTIALLY_PAID
PAID
PAYMENT_FAILED
PARTIALLY_REFUNDED
REFUNDED
```

### Attendance lifecycle

```text
NOT_STARTED
CHECKED_IN
ATTENDED
NO_SHOW
```

Эти состояния независимы. Например:

```text
Booking=CANCELLED
Payment=PARTIALLY_REFUNDED
Attendance=NOT_STARTED
```

## 7. Перенос

Перенос не освобождает старый слот заранее.

В одной атомарной операции:

1. строится план новых allocations с исключением текущей брони;
2. рассчитывается новая цена;
3. старые allocations деактивируются;
4. новые allocations добавляются;
5. booking и snapshots обновляются;
6. создаётся audit event.

Любая ошибка оставляет исходное состояние нетронутым.

## 8. Безопасность MVP

- пароли: PBKDF2-HMAC-SHA256, индивидуальная соль, 120 000 итераций;
- сессии: HMAC-SHA256, срок 14 дней;
- cookie: `HttpOnly`, `SameSite=Lax`, `Secure` за HTTPS/reverse proxy;
- request body: максимум 1 MiB;
- неизвестные JSON-поля отклоняются;
- публичная бронь требует согласия с зафиксированными правилами отмены;
- API-ответы не кешируются;
- basic browser security headers;
- JSON-файл создаётся с mode `0600`.

## 9. Переход на PostgreSQL

Целевая замена хранилища:

```text
Service
  ↓
Repository / Unit of Work
  ↓
PostgreSQL
```

Для эксклюзивных ресурсов рекомендуется `tstzrange` и exclusion constraint:

```sql
EXCLUDE USING gist (
  resource_id WITH =,
  occupied_period WITH &&
)
WHERE (status = 'ACTIVE');
```

Для shared capacity нужна транзакционная блокировка строки ресурса/сеанса и повторная проверка суммы quantity. Внешние create/payment/cancel операции должны получить idempotency key.

## 10. Масштабирование

До нескольких реплик необходимо реализовать:

- PostgreSQL как единственный source of truth;
- distributed-safe hold expiration;
- transactional outbox;
- отдельный worker;
- webhook inbox с дедупликацией;
- rate limiting;
- tenant-aware authorization;
- migrations и versioned schemas.
