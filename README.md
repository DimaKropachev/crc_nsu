# VK Mini App — карьерные консультации НГУ

В репозитории два согласованных проекта:

- `backend/` — Go REST API, PostgreSQL, миграции, авторизация VK, запись на консультации, хранение резюме и worker AI-анализа.
- `frontend/` — React + TypeScript + Vite интерфейс VK Mini App.

## Запуск бэкенда

```bash
cd backend
cp .env.example .env
```

Заполни `POSTGRES_PASSWORD`, `VK_APP_ID`, `VK_APP_SECRET`, `ADMIN_VK_IDS` и `JWT_SECRET` в `.env`. Для локальной разработки можно использовать development-конфигурацию, но не публикуй секреты.

```bash
docker compose up --build
```

API по умолчанию будет доступен на `http://localhost:8080`. Проверить запуск можно запросом `GET http://localhost:8080/api/v1/health/ready`.

## Запуск фронтенда

В новом терминале:

```bash
cd frontend
cp .env.example .env
npm install 
npm run dev
```

В `.env` фронтенда укажи origin бэкенда в `VITE_API_BASE_URL`. Для локальной разработки без VK и бэкенда можно включить `VITE_MOCK_MODE=true`; данные и отчёт в таком режиме демонстрационные. Не включай mock-режим в production.

Настоящая авторизация требует открыть приложение через VK Mini Apps и настроить параметры VK на сервере. В development backend разрешает локальные origin, например `http://localhost:5173`.

## Согласие на AI-анализ

Перед стартом анализа студент подтверждает передачу текста резюме внешнему ИИ-провайдеру. Фронтенд отправляет `external_ai_consent: true` в `POST /api/v1/resume-analyses`; бэкенд сохраняет факт и время согласия и не ставит задачи без согласия в очередь. Для фактического анализа нужно настроить `AI_PROVIDER`, `AI_API_KEY` и `AI_MODEL` в backend `.env`.

## Проверки

Фронтенд: `npm run build`, `npm run test`. Бэкенд: `go test ./...`, `go vet ./...`. Для integration-тестов бэкенда требуется отдельная тестовая PostgreSQL. Перед production также проверь реальные параметры запуска VK и интеграцию с AI/VK API.
