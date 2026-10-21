# VK Mini App — карьерные консультации НГУ

В репозитории два согласованных проекта:

- `backend/` — Go REST API, PostgreSQL, миграции, авторизация VK, запись на консультации, хранение резюме и worker AI-анализа.
- `frontend/` — React + TypeScript + Vite интерфейс VK Mini App.

## Запуск всего приложения

Требуются Docker Engine и Docker Compose v2. Из корня репозитория:

```bash
docker compose up --build -d
```

Приложение доступно на **http://localhost:8080**. Compose запускает PostgreSQL, применяет миграции, дожидается готовности Go API и запускает nginx с собранным frontend. API и база доступны только внутри Docker-сети.

```text
Браузер → nginx:80 → /             → React frontend
                  → /api/v1/*     → api:8080 → postgres:5432
```

Frontend использует относительные `/api/v1/*`; nginx сохраняет маршруты, Bearer-токены и multipart-загрузки. Контракт описан в `backend/api/openapi.yaml`, клиент и DTO — в `frontend/src/shared/api/`. Коллекции имеют вид `{ "items": [...] }`, ошибки — `{ "error": { "code", "message", "details" } }`, даты — RFC3339.

Без `.env` запускается development-инфраструктура, но **для реального входа нужны настройки VK**. Mock-режим в Docker отключён. Без настроек VK авторизация возвращает `503 VK_AUTH_NOT_CONFIGURED`, а не фиктивную сессию.

### Настройки окружения

```bash
cp .env.example .env
```

В Windows CMD: `copy .env.example .env`. Заполни `POSTGRES_PASSWORD`, `VK_APP_ID`, `VK_APP_SECRET`, `ADMIN_VK_IDS` и `JWT_SECRET` в корневом `.env`. Compose не читает `.env` из подкаталогов. Для пароля PostgreSQL используй URL-safe символы: он включается в строку подключения. Секреты передаются только серверу, не в frontend bundle.

Для публикации настрой HTTPS и адрес приложения в VK, установи `APP_ENV=production`, случайный `JWT_SECRET` не короче 32 символов и `CORS_ALLOWED_ORIGINS` с точным публичным origin frontend. Не используй development-пароли в production.

```bash
docker compose up --build -d
```

Проверка всей цепочки nginx → API → PostgreSQL: `GET http://localhost:8080/api/v1/health/ready`, ожидаемый ответ — `{"status":"ready"}`. Внешний порт задаётся через `HTTP_PORT`.

Управление: `docker compose ps`, `docker compose logs -f api frontend`, `docker compose down`. PostgreSQL и резюме сохраняются в именованных volumes. **`docker compose down -v` удаляет данные.** Изменение пароля в `.env` не меняет пароль уже созданной базы — его нужно менять средствами PostgreSQL.

## Запуск фронтенда

В новом терминале:

```bash
cd frontend
cp .env.example .env
npm install 
npm run dev
```

Для отдельного запуска frontend оставь `VITE_API_BASE_URL` пустым: Vite проксирует `/api/v1` на `http://localhost:8080` (локальный API или Docker-стек). Если backend расположен отдельно, укажи его origin без `/api/v1` в `VITE_API_BASE_URL`. Для локальной разработки без VK и бэкенда можно включить `VITE_MOCK_MODE=true`; данные и отчёт в таком режиме демонстрационные. Не включай mock-режим в production. Переменные Vite применяются при сборке, не при запуске nginx.

Настоящая авторизация требует открыть приложение через VK Mini Apps и настроить параметры VK на сервере. В development backend разрешает локальные origin, например `http://localhost:5173`.

## Согласие на AI-анализ

Перед стартом анализа студент подтверждает передачу текста резюме внешнему ИИ-провайдеру. Фронтенд отправляет `external_ai_consent: true` в `POST /api/v1/resume-analyses`; бэкенд сохраняет факт и время согласия и не ставит задачи без согласия в очередь. Для фактического анализа нужно настроить `AI_PROVIDER`, `AI_API_KEY` и `AI_MODEL` в корневом `.env`. Без провайдера консультации работают, анализ возвращает `503`. Worker анализа работает внутри API. Напоминания включаются через `REMINDER_ENABLED=true` и `NOTIFICATION_API_TOKEN` (токен сообщества VK).

## Проверки

Фронтенд: `npm run build`, `npm run test`. Бэкенд: `go test ./...`, `go vet ./...`. Для integration-тестов бэкенда требуется отдельная тестовая PostgreSQL. Перед production также проверь реальные параметры запуска VK и интеграцию с AI/VK API.
