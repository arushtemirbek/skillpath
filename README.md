# Skillpath — Roadmap Generator API

Go backend для генерации карты обучения через OpenAI. Реализован MVP: генерация двухуровневого roadmap через OpenAI, валидация входных данных и ответа, конфигурация, request ID, structured logging, обработка ошибок и graceful shutdown.

Данные не сохраняются; пользователей, базы данных и аутентификации нет.

## Запуск

Требуется Go 1.25.4 или новее.

```bash
cp .env.example .env
```

Укажите `OPENAI_API_KEY` и `OPENAI_MODEL` в `.env`, затем:

```bash
go run ./cmd/api
```

Для генерации нужен рабочий ключ OpenAI и модель, поддерживающая Chat Completions со Structured Outputs (`json_schema`). Модель выбирается через `OPENAI_MODEL`, без подмены или fallback. `/health` также проверяет провайдера реальным коротким запросом.

`.env` загружается из текущей директории и не перезаписывает переменные окружения процесса. Отсутствующий файл допустим; ошибка его формата или доступа останавливает приложение. `.env` исключён из Git и Docker build context.

## Конфигурация

| Переменная | Значение по умолчанию | Назначение |
|---|---|---|
| `HTTP_PORT` | `8080` | Порт, от 1 до 65535 |
| `OPENAI_API_KEY` | — | Обязательный ключ |
| `OPENAI_MODEL` | — | Обязательная модель |
| `OPENAI_BASE_URL` | `https://api.openai.com` | Абсолютный HTTP(S) URL без credentials, query и fragment |
| `REQUEST_TIMEOUT` | `30s` | Deadline контекста запроса и таймаут LLM-клиента |
| `HTTP_READ_TIMEOUT` | `5s` | Чтение запроса; также используется для ReadHeaderTimeout |
| `HTTP_WRITE_TIMEOUT` | `35s` | Запись ответа; должен превышать REQUEST_TIMEOUT |
| `HTTP_IDLE_TIMEOUT` | `60s` | Ожидание следующего запроса в keep-alive соединении |
| `HTTP_SHUTDOWN_TIMEOUT` | `10s` | Время на завершение активных запросов |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

Таймауты должны быть положительными, например `500ms` или `30s`. Явно пустые значения настроек считаются ошибкой. Для совместимости `OPENAI_TIMEOUT` используется вместо `REQUEST_TIMEOUT`, только если последний отсутствует в environment.

В конфигурации HTTP deadline хранится в `HTTP.RequestTimeout`, а таймаут клиента — в `OpenAI.Timeout`; оба задаются через `REQUEST_TIMEOUT`.

Request deadline отменяет контекст. Контекст передаётся из handler в service и HTTP-запрос OpenAI; отмена входящего запроса отменяет внешний вызов; middleware не останавливает произвольный код принудительно.

## Health

```bash
curl -i http://localhost:8080/health
```

Это проверка готовности сервиса и его внешних зависимостей. Каждый вызов `/health` отправляет небольшой запрос к настроенной модели OpenAI с просьбой ответить `OK` (до 64 completion tokens). Это настоящий API-вызов с расходом токенов.

Успешный ответ: `200 OK`:

```json
{"status":"ok","checks":{"openai":"ok"}}
```

При недоступности OpenAI, ошибке ключа/модели, неожиданном ответе или timeout: `503 Service Unavailable`:

```json
{"status":"error","checks":{"openai":"error"}}
```

Проверки зависимостей выполняются параллельно с общим REQUEST_TIMEOUT. Чтобы добавить базу или кэш, реализуйте `health.Checker` (`Check(context.Context) error`) и передайте именованную зависимость в `health.NewHandler` при сборке приложения. Проверка должна учитывать отмену context. Сырые ошибки зависимостей не возвращаются клиенту.

Клиент может передать `X-Request-ID`. Принимаются значения длиной до 128 символов из букв ASCII, цифр, `.`, `_`, `-`. Иначе сервер генерирует новый ID. Значение возвращается в заголовке ответа и попадает в logs.

## Генерация roadmap

```bash
curl -i http://localhost:8080/roadmaps/generate \
  -H 'Content-Type: application/json' \
  -d '{"goal":"Стать Go backend developer","context":"Python developer с опытом 2 года"}'
```

`goal` обязателен; `context` необязателен. Пробелы по краям удаляются. Неизвестные поля, malformed JSON, несколько JSON-объектов и тело больше 1 MiB отклоняются с `400 INVALID_REQUEST`. Пустой или состоящий из пробелов goal возвращает `400` с сообщением `goal is required` без вызова LLM.

Пример формы успешного ответа (`200 OK`; конкретное содержимое зависит от модели):

```json
{
  "title": "Go Backend Developer",
  "description": "Карта подготовки к backend-разработке на Go.",
  "sections": [
    {
      "title": "Go Fundamentals",
      "description": "Основы языка.",
      "topics": [
        {"title": "Interfaces"},
        {"title": "Error handling"}
      ]
    }
  ]
}
```

Адаптер вызывает `/v1/chat/completions` с `response_format.type=json_schema`, `strict=true` и `additionalProperties=false` на всех уровнях. При `OPENAI_BASE_URL=https://api.openai.com/v1` версия не дублируется. Формат реализован по [официальной документации Structured Outputs](https://developers.openai.com/api/docs/guides/structured-outputs?api-mode=chat).

После ответа проверяются завершение генерации и отсутствие refusal. JSON декодируется строго: обязательные поля должны присутствовать, лишние поля (включая `children`) запрещены. Service проверяет непустой title roadmap, наличие sections, title каждой section, наличие topics и title каждого topic. Отказ, обрезанный ответ, некорректная структура или нарушение этих правил дают `502 LLM_INVALID_RESPONSE`.

Ответ провайдера ограничен 2 MiB. Redirects не выполняются, чтобы credentials не передавались на другой endpoint. HTTP-ошибки API, network errors и timeouts дают `502 LLM_REQUEST_FAILED`. Logs LLM содержат `model`, `duration`, upstream `status` (0, если ответ не получен) и безопасное описание `error`; prompts и сырые ответы не логируются.

## Ошибки

```json
{"error":{"code":"INVALID_REQUEST","message":"goal is required"}}
```

HTTP mapper преобразует ошибки приложения, включая wrapped errors:

| Ошибка | HTTP | Code |
|---|---|---|
| Пустой goal / некорректный запрос | 400 | INVALID_REQUEST |
| Семантическая ошибка входных данных | 422 | INVALID_REQUEST |
| Ошибка внешнего LLM-запроса | 502 | LLM_REQUEST_FAILED |
| Некорректный ответ LLM | 502 | LLM_INVALID_RESPONSE |
| Неизвестная ошибка | 500 | INTERNAL_ERROR |

Неизвестные маршруты возвращают 404, неподдерживаемые методы — 405, с тем же JSON envelope. Внутренние детали ошибок не передаются клиенту. Handler генерации использует этот mapper.

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
internal/roadmap   HTTP handler, service, domain models, prompts, schema and decoding
internal/llm       generic OpenAI completions and dependency check
```

Сценарий: HTTP Handler → Roadmap Service → Roadmap Generator → общий LLM Client → OpenAI.

`llm.Complete` принимает сообщения и необязательную JSON Schema. Общий клиент не импортирует `roadmap`; планы и курсы смогут использовать тот же клиент со своими промптами, схемами и валидацией. Ошибки централизованы в `apperror`, их преобразование в HTTP — в `httperror`. База данных, пользователи, аутентификация и сохранение roadmap в текущий scope не входят.

## Проверки

```bash
go test -race ./...
go vet ./...
go build -o ./tmp/main ./cmd/api
```

Тесты service и handler используют fake LLM. Тесты адаптера и полного сценария HTTP → Service → Client используют локальный mock-провайдер. Реальные платные запросы OpenAI не выполняются. Проверяются успешная генерация, пустой goal, malformed JSON, ошибки API, неполные ответы, лишние уровни, бизнес-инварианты и отмена контекста.

Артефакты Air в `tmp/` исключены из Git.

## Docker

```bash
docker build -t roadmap-api .
docker run --rm --env-file .env -p 8080:8080 roadmap-api
```

Multi-stage build создаёт бинарник без CGO; runtime использует непривилегированного пользователя и CA certificates. `.env` в образ не копируется. Docker Compose не требуется.

## Ограничения и дальнейшее развитие

Карта содержит ровно два уровня: sections → topics. Это обзор знаний, а не уроки или расписание. Бизнес-валидация проверяет структуру и непустые заголовки; она не может гарантировать педагогическую полноту контента LLM.

Для текущего MVP нет persistence, аутентификации, streaming и автоматических retries. Модель должна поддерживать используемый формат Structured Outputs; неподдерживаемая модель или ошибка её доступа возвращается как `502 LLM_REQUEST_FAILED`.

Дальнейшие задачи: persistence, история roadmap, аутентификация и observability. Учебные планы и уроки должны развиваться отдельно от двухуровневого roadmap.
