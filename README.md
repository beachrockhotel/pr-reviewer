# PR Reviewer Assignment Service

Сервис для автоматического назначения ревьюверов на Pull Request'ы внутри команды.

## Команды

```bash
task tools:install

task gen:oapi

docker compose up --build
```

## Дополнительно реализованы
- Линтер
- Эндпоинт статистики
    - `GET /stats` — количество PR по статусам (`OPEN`, `MERGED`).
- Нагрузочное тестирование

## Качество кода

Для проверки стиля и статического анализа используется golangci-lint:

```bash
task fmt
task lint
```

## Нагрузочное тестирование
Тестировал ручку `POST /pullRequest/create`
```bash
Итоги:
Requests/sec: ~16 392

Средняя задержка: ~0.6 ms

p95: ~0.8 ms

p99: ~1.0 ms
```
Тестировал ручку `GET /users/getReview`
```bash
Итоги:

Requests/sec: ~24 860

Средняя задержка: ~0.4 ms

p95: ~0.6 ms

p99: ~0.8 ms

Статусы: только 200.
"
```