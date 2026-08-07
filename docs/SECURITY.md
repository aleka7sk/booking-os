# Security and production checklist

## Уже реализовано

- PBKDF2-HMAC-SHA256 для пароля;
- криптографически случайная соль;
- constant-time сравнение;
- HMAC-подписанные сессии;
- `HttpOnly`, `SameSite=Lax`, `Secure` при HTTPS;
- request size limit;
- JSON unknown field rejection;
- security headers;
- no-store для API;
- файл данных mode `0600`;
- минимальная выдача user data в auth API;
- audit событий изменения брони.

## Обязательно перед внешним pilot

- заменить admin password;
- установить случайный session secret не менее 32 байт;
- разместить за HTTPS reverse proxy;
- закрыть прямой доступ к data volume;
- настроить ежедневный encrypted backup;
- ограничить доступ к pilot по сети или списку пользователей;
- проверить тексты согласия и cancellation policy;
- определить сроки хранения персональных данных;
- исключить secrets из логов и репозитория.

## Обязательно перед production SaaS

- PostgreSQL и tenant isolation;
- server-side RBAC на каждом маршруте;
- MFA для владельцев;
- password reset и session revocation;
- CSRF review для будущих cross-site integrations;
- rate limiting и brute-force protection;
- webhook signatures и replay protection;
- idempotency storage;
- secrets manager;
- SAST/dependency/container scanning;
- structured audit access logs;
- incident response и restore drills;
- privacy/legal review по законодательству Казахстана.

## Известные ограничения MVP

- все пользователи текущей демо-инсталляции имеют один owner account;
- JSONStore безопасен только внутри одного процесса;
- публичный token не имеет отдельного revoke action;
- нет rate limiting;
- нет автоматической ротации session secret;
- нет field-level encryption в JSON-файле.
