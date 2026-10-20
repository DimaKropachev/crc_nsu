# VK Mini App — карьерные консультации НГУ

React + TypeScript frontend для согласованного Go REST API. Основные рабочие сценарии: запись на карьерную консультацию, просмотр/перенос своих записей, кабинет консультанта, расписание и асинхронный анализ резюме.

## Стек

- React 18 + TypeScript strict
- Vite
- VKUI и VK Bridge
- React Router
- TanStack Query
- Vitest + React Testing Library

## Запуск

Требуются Node.js 20+ и npm.

```bash
cp .env.example .env
npm install
npm run dev
```

По умолчанию фронтенд использует реальный API с origin `http://localhost:8080`; приложение само добавляет префикс `/api/v1`. Если backend размещён на другом адресе, укажи origin в `VITE_API_BASE_URL`.

Для настоящей VK-авторизации нужно открыть приложение через VK Mini Apps, настроить приложение VK на сервере и указать тот же `VK_APP_ID` / секрет в бэкенде. Фронтенд отправляет оригинальные параметры запуска, а их подпись проверяется только сервером. Секрет VK никогда не попадает в frontend bundle.

### Локальная разработка без VK/бэкенда

В `.env` задай:

```dotenv
VITE_API_BASE_URL=http://localhost:8080
VITE_MOCK_MODE=true
```

Перезапусти `npm run dev` и на стартовом экране выбери демонстрационную роль студента или консультанта. В mock-режиме данные и анализ резюме являются демонстрационными, а не данными с сервера. **Не включай `VITE_MOCK_MODE=true` в production.**

## Команды

```bash
npm run dev
npm run build
npm run test
npm run preview
```

## API

Все запросы идут через `src/shared/api/client.ts`; типы и API-модули находятся в `src/shared/api/`.

| Экран / сценарий | API |
| --- | --- |
| Авторизация через VK | `POST /api/v1/auth/vk`, `GET /api/v1/me` |
| Согласие на напоминания (API предусмотрен backend) | `POST /api/v1/me/notification-consent` |
| Выбор консультанта и времени | `GET /api/v1/consultants`, `GET /api/v1/consultants/{id}/slots` |
| Создание записи | `POST /api/v1/files/resumes` (если приложен файл), `POST /api/v1/consultations` |
| Мои записи | `GET /api/v1/consultations/my`, `POST /api/v1/consultations/{id}/reschedule`, `POST /api/v1/consultations/{id}/cancel` |
| Кабинет консультанта | `GET /api/v1/admin/consultations`, `POST /api/v1/admin/consultations/{id}/confirm`, `POST /api/v1/admin/consultations/{id}/complete` |
| Расписание консультанта | `GET /api/v1/admin/slots`, `POST /api/v1/admin/slots`, `PATCH /api/v1/admin/slots/{id}`, `POST /api/v1/admin/slots/{id}/block` |
| Профиль консультанта | `GET /api/v1/admin/profile`, `PATCH /api/v1/admin/profile` |
| Анализ резюме | `POST /api/v1/files/resumes`, `POST /api/v1/resume-analyses` (с обязательным `external_ai_consent: true`), `GET /api/v1/resume-analyses`, `GET /api/v1/resume-analyses/{id}` |

## Структура

```text
src/
├── app/             # AuthProvider, shell и маршрутизация
├── pages/           # экраны приложения
├── shared/api/      # DTO, HTTP-клиент, API-модули и dev mock
├── shared/lib/      # форматирование дат
├── shared/ui.tsx    # переиспользуемые элементы
├── main.tsx
└── styles.css
```

## Важные детали контракта

- Бэкенд возвращает коллекции обёрнутыми в `{ "items": [...] }`; API-модули преобразуют их в массивы.
- `GET /api/v1/me` возвращает `User` напрямую, а `POST /api/v1/auth/vk` — `{ access_token, token_type, expires_in, user }`.
- Токен хранится в `sessionStorage` на время вкладки, не в `localStorage`. При `401` приложение сбрасывает сессию и показывает экран повторного входа.
- Загрузка файла идёт через `FormData`; `Content-Type` вручную не устанавливается, чтобы браузер добавил boundary.
- Отчёт строится только по результату, который вернул сервер. Демонстрационный отчёт включается лишь в mock-режиме.
- Волонтёрство, каталог возможностей и рассылки явно помечены как демонстрационные заглушки.

## Проверки

После установки зависимостей выполни `npm run build` и `npm run test`. Тестовые и настоящие данные не смешиваются: mock-режим включается исключительно явной переменной окружения.
