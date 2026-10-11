# Skillpath — Roadmap Generator API

Go backend для генерации карты обучения через OpenAI. На текущем этапе реализована инфраструктура сервиса: конфигурация, HTTP server, health check, request ID, structured logging, обработка ошибок и graceful shutdown.

Бизнес-логика roadmap и интеграция с OpenAI пока не реализованы. Endpoint `POST /roadmaps/generate` не подключён и возвращает 404.

## Запуск

Требуется Go 1.25.4 или новее.

```bash
cp .env.example .env
```

Укажите `OPENAI_API_KEY` и `OPENAI_MODEL` в `.env`, затем:

```bash
go run ./cmd/api
```

Обязательные настройки провайдера валидируются заранее согласно ТЗ, но внешний API сейчас не вызывается. Для проверки только инфраструктуры допустимы непустые тестовые значения ключа и модели.

`.env` загружается из текущей директории и не перезаписывает переменные окружения процесса. Отсутствующий файл допустим; ошибка его формата или доступа останавливает приложение. `.env` исключён из Git и Docker build context.

## Конфигурация

| Переменная | Значение по умолчанию | Назначение |
|---|---|---|
| `HTTP_PORT` | `8080` | Порт, от 1 до 65535 |
| `OPENAI_API_KEY` | — | Обязательный ключ |
| `OPENAI_MODEL` | — | Обязательная модель |
| `OPENAI_BASE_URL` | `https://api.openai.com` | Абсолютный HTTP(S) URL без credentials, query и fragment |
| `REQUEST_TIMEOUT` | `30s` | Deadline контекста запроса и настройка таймаута будущего LLM-клиента |
| `HTTP_READ_TIMEOUT` | `5s` | Чтение запроса; также используется для ReadHeaderTimeout |
| `HTTP_WRITE_TIMEOUT` | `35s` | Запись ответа; должен превышать REQUEST_TIMEOUT |
| `HTTP_IDLE_TIMEOUT` | `60s` | Ожидание следующего запроса в keep-alive соединении |
| `HTTP_SHUTDOWN_TIMEOUT` | `10s` | Время на завершение активных запросов |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

Таймауты должны быть положительными, например `500ms` или `30s`. Явно пустые значения настроек считаются ошибкой. Для совместимости `OPENAI_TIMEOUT` используется вместо `REQUEST_TIMEOUT`, только если последний отсутствует в environment.

В конфигурации HTTP deadline хранится в `HTTP.RequestTimeout`, а таймаут будущего клиента — в `OpenAI.Timeout`; оба задаются через `REQUEST_TIMEOUT`.

Request deadline отменяет контекст. Будущие handlers, services и внешние клиенты должны передавать этот контекст дальше и учитывать отмену; middleware не останавливает произвольный код принудительно.

## Health

```bash
curl -i http://localhost:8080/health
```

Статус `200 OK`, Content-Type `application/json`, тело:

```json
{"status":"ok"}
```

Это liveness check: он не вызывает OpenAI и не проверяет доступность провайдера.

Клиент может передать `X-Request-ID`. Принимаются значения длиной до 128 символов из букв ASCII, цифр, `.`, `_`, `-`. Иначе сервер генерирует новый ID. Значение возвращается в заголовке ответа и попадает в logs.

## Ошибки

```json
{"error":{"code":"INVALID_REQUEST","message":"goal is required"}}
```

Подготовлен HTTP mapper для ошибок приложения:

| Ошибка | HTTP | Code |
|---|---|---|
| Пустой goal / некорректный запрос | 400 | INVALID_REQUEST |
| Семантическая ошибка входных данных | 422 | INVALID_REQUEST |
| Ошибка внешнего LLM-запроса | 502 | LLM_REQUEST_FAILED |
| Некорректный ответ LLM | 502 | LLM_INVALID_RESPONSE |
| Неизвестная ошибка | 500 | INTERNAL_ERROR |

Неизвестные маршруты возвращают 404, неподдерживаемые методы — 405, с тем же JSON envelope. Внутренние детали ошибок не передаются клиенту. Mapper будет использоваться при реализации handler генерации.

## Lifecycle и logs

`SIGINT` и `SIGTERM` запускают graceful shutdown: сервер прекращает принимать новые соединения и ждёт завершения активных запросов. По истечении ShutdownTimeout соединения закрываются принудительно, процесс завершается с ошибкой.

Logs выводятся в stdout в JSON через `log/slog`. HTTP log содержит `request_id`, `method`, `path`, `status`, `duration`. Ключ API, заголовки Authorization и содержимое конфигурации не логируются.

## Архитектура

```text
cmd/api            startup, configuration, signals
internal/config    environment parsing and validation
internal/server    route construction and HTTP server lifecycle
internal/server/health
                   liveness endpoint
internal/middleware
                   request ID, logging, recovery, request deadline
internal/logger    JSON slog setup
internal/apperror  transport-independent application errors
internal/httperror JSON error envelope and application error mapping
internal/roadmap   reserved for handlers, service and models
internal/llm       reserved for provider integration
```

Будущий сценарий: HTTP Handler → Roadmap Service → LLM Client → OpenAI. База данных, пользователи, аутентификация и сохранение roadmap в текущий scope не входят.

## Проверки

Пока файлы `roadmap` и `llm` остаются пустыми заготовками, проверять реализованные пакеты:

```bash
go test ./cmd/api ./internal/config ./internal/logger ./internal/server/... ./internal/middleware ./internal/apperror ./internal/httperror
go build -o ./tmp/main ./cmd/api
```

`go test ./...` пока не проходит из-за пустых `.go`-файлов заготовок. Они намеренно сохранены до следующего этапа. Артефакты Air в `tmp/` исключены из Git.

## Docker

```bash
docker build -t roadmap-api .
docker run --rm --env-file .env -p 8080:8080 roadmap-api
```

Multi-stage build создаёт бинарник без CGO; runtime использует непривилегированного пользователя и CA certificates. `.env` в образ не копируется. Docker Compose не требуется.

## Следующий этап

Реализовать `POST /roadmaps/generate`, DTO из ТЗ, валидацию goal, LLM-клиент со structured output и unit tests. Roadmap должен содержать только sections и topics. После этого можно оценивать полную Definition of Done MVP.
