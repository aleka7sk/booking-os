# ADR-001: один Go-бинарник и атомарное JSON-хранилище для MVP

**Статус:** принято для MVP

## Контекст

Пустой репозиторий нужно было превратить в демонстрируемый вертикальный продукт без зависимости от внешней инфраструктуры и package registries.

## Решение

- Go standard library backend;
- embedded vanilla JS/CSS frontend;
- JSONStore с mutex, copy-on-write и atomic rename;
- modular monolith;
- PostgreSQL оставлен целевым production-хранилищем.

## Последствия

Плюсы:

- запуск одной командой;
- отсутствие npm/runtime dependencies;
- переносимый бинарник;
- быстрый пилот и прозрачная отладка.

Минусы:

- одна реплика;
- нет горизонтального масштабирования;
- запись всего состояния при каждой транзакции;
- нет DB constraints и SQL analytics.

## Условие отмены решения

Перед первым production SaaS deployment JSONStore должен быть заменён PostgreSQL repository с транзакциями и migration tooling.
