# VK Mini App — карьерные консультации НГУ

Приложение для записи на карьерные консультации и анализа резюме.

## Технологии

- **Frontend (`frontend/`):** React 18, TypeScript, Vite, VKUI и VK Bridge, React Router, TanStack Query.
- **Backend (`backend/`):** Go 1.23, REST API на chi, JWT-авторизация, pgx для работы с PostgreSQL.
- **База данных и запуск:** PostgreSQL 16, SQL-миграции, Docker Compose, nginx для раздачи frontend и проксирования API.
- **Тестирование:** Vitest и Testing Library на frontend, стандартные тесты Go на backend.

## Запуск всего приложения

1. Установи Git и Docker Desktop (Windows/macOS) либо Docker Engine с Docker Compose v2 (Linux). Запусти Docker. Node.js и Go для этого способа не нужны.
2. Клонируй репозиторий и открой терминал в его корневой папке `CRC_NSU` — там, где находится `docker-compose.yaml`.
3. Создай корневой `.env` из примера. Если файл уже есть, сохрани свои настройки.

   Windows (PowerShell или CMD):

   ```powershell
   copy .env.example .env
   ```

   Linux/macOS:

   ```bash
   cp .env.example .env
   ```

4. Для локального запуска оставь `APP_ENV=development`. Значений из примера достаточно для запуска инфраструктуры. Для входа через VK заполни `VK_APP_ID` и `VK_APP_SECRET` параметрами Mini App; подробности ниже.
5. Собери и запусти приложение из корня репозитория:

   ```bash
   docker compose up --build -d
   ```

6. Дождись готовности контейнеров (`docker compose ps`) и открой **http://localhost:8080**. При первой сборке Docker скачивает образы и зависимости, поэтому требуется интернет и запуск может занять несколько минут.

Compose запускает PostgreSQL, применяет миграции, дожидается готовности Go API и запускает nginx с собранным frontend. API и база доступны только внутри Docker-сети.

Для проверки открой **http://localhost:8080/api/v1/health/ready**: ожидаемый ответ — `{"status":"ready"}`. Это подтверждает работу nginx, API и базы. Для просмотра интерфейса без настройки VK используй демонстрационный режим из раздела «Запуск фронтенда».

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

В `.env.example` значение `JWT_SECRET` пустое: в development сервер использует локальный секрет по умолчанию. Перед публикацией задай собственный случайный секрет. Сервер завершает работу в production, если `CORS_ALLOWED_ORIGINS` пуст, `JWT_SECRET` короче 32 символов или начинается с `replace-with-`, либо отсутствуют `VK_APP_ID` и `VK_APP_SECRET`. Причину завершения смотри через `docker compose logs --tail=50 api`; после исправления `.env` пересоздай контейнер командой `docker compose up -d --no-deps --force-recreate api`.

```bash
docker compose up --build -d
```

Проверка всей цепочки nginx → API → PostgreSQL: `GET http://localhost:8080/api/v1/health/ready`, ожидаемый ответ — `{"status":"ready"}`. Внешний порт задаётся через `HTTP_PORT`.

### Cloudflare Tunnel и ошибка «Origin не разрешён»

На Windows и Linux можно запустить Quick Tunnel в сети Compose (без `--network host`):

```bash
docker compose -f docker-compose.yaml -f compose.tunnel.yaml up -d --build
docker compose -f docker-compose.yaml -f compose.tunnel.yaml logs --tail=100 tunnel
```

Origin туннеля внутри Docker — `http://frontend:80`, а не `localhost` контейнера cloudflared. Конфигурация использует HTTP/2. Это временный Quick Tunnel, не постоянный production-домен. Для Aeza используй постоянный домен и named tunnel с отдельными credentials; секреты не помещай в Git. Не публикуй development-конфигурацию с реальными данными. До публикации задай production-настройки ниже и проверь HTTPS, вход через VK и ограничения доступа. Успешный `docker compose config` проверяет конфигурацию, но не соединение с Cloudflare.

Для туннеля на Linux-сервере сначала проверь локальную готовность приложения, затем запусти:

```bash
curl -i http://127.0.0.1:8080/api/v1/health/ready
docker run --rm --network host cloudflare/cloudflared:latest tunnel --protocol http2 --url http://127.0.0.1:8080
```

Полученный публичный origin нужно добавить в `CORS_ALLOWED_ORIGINS` в корневом `.env`. Например, если туннель выдал `https://example.trycloudflare.com`, настройка выглядит так:

```dotenv
CORS_ALLOWED_ORIGINS=https://example.trycloudflare.com
```

Используй фактический адрес туннеля без завершающего `/`, пути и параметров запуска VK. Несколько origin перечисляются через запятую. Указывай origin страницы приложения, а не родительского окна `vk.com`. Затем примени окружение к API:

```bash
docker compose up -d --no-deps --force-recreate api
```

Проверь разрешение origin (подставь тот же фактический адрес):

```bash
curl -i -H 'Origin: https://example.trycloudflare.com' http://127.0.0.1:8080/api/v1/health/ready
```

Ожидается `200 OK` и `Access-Control-Allow-Origin` с этим адресом. Проверка без заголовка `Origin` подтверждает готовность приложения, но не проверяет CORS. После этого повтори вход из VK; публичный URL также должен быть указан в настройках Mini App. Ошибка origin возникает до проверки роли пользователя и не связана с правами администратора.

Quick Tunnel выдаёт новый адрес при новом запуске: обновляй `.env`, пересоздавай API и меняй URL в VK. Для постоянного размещения используй туннель с постоянным доменом. Не отключай проверку origin и проверку подписи VK для обхода этой ошибки.

Управление: `docker compose ps`, `docker compose logs -f api frontend`, `docker compose down`. PostgreSQL и резюме сохраняются в именованных volumes. **`docker compose down -v` удаляет данные.** Изменение пароля в `.env` не меняет пароль уже созданной базы — его нужно менять средствами PostgreSQL.

## Запуск фронтенда

Этот способ подходит для разработки интерфейса с автоматическим обновлением. Установи **Node.js 22 LTS** (npm входит в комплект). В терминале из корня репозитория:

```bash
cd frontend
npm ci
```

Создай `frontend/.env`: в Windows выполни `copy .env.example .env`, в Linux/macOS — `cp .env.example .env`. Если файл уже есть, отредактируй его.

Для запуска **без backend и ключей VK** укажи в `frontend/.env`:

```dotenv
VITE_API_BASE_URL=
VITE_MOCK_MODE=true
```

Затем в папке `frontend` выполни:

```bash
npm run dev
```

Открой адрес из вывода Vite (обычно **http://localhost:5173**). Данные и отчёт в этом режиме демонстрационные. Остановка — `Ctrl+C`.

Для работы с реальным API сначала запусти Docker-стек по инструкции выше, установи `VITE_MOCK_MODE=false` и оставь `VITE_API_BASE_URL` пустым: Vite проксирует `/api/v1` на `http://localhost:8080`. Если backend расположен отдельно, укажи его origin без `/api/v1` в `VITE_API_BASE_URL`. После изменения `.env` перезапусти Vite. В Docker mock-режим отключён; переменные Vite применяются при сборке, не при запуске nginx.

Настоящая авторизация требует открыть приложение через VK Mini Apps и настроить параметры VK на сервере. В development backend разрешает локальные origin, например `http://localhost:5173`.

## Проверки

В папке `frontend`: `npm run build`, `npm run test`. В папке `backend` (требуется Go 1.23 или новее): `go test ./...`, `go vet ./...`. Для integration-тестов бэкенда требуется отдельная тестовая PostgreSQL. Перед production также проверь реальные параметры запуска VK и интеграцию с AI/VK API.
