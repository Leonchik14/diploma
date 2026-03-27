# API Gateway — документация API

Единая точка входа в бэкенд — **api-gateway**. Все вызовы идут по **gRPC** на порт **9090** (по умолчанию). Включён gRPC reflection для интроспекции.

---

## Аутентификация

- **Методы без токена** (публичные): `Register`, `Login`, `Refresh`, `CheckPasswordResetEmail`, `SendPasswordResetCode`, `VerifyPasswordReset`.
- **Остальные методы** требуют JWT в metadata:
  - Ключ: `Authorization`, значение: `Bearer <access_token>` (access_token выдаётся в ответах `Login` / `Register` / `Refresh`).
- Шлюз извлекает из JWT `user_id` и передаёт его в бэкенд через metadata `x-user-id`.

---

## 1. Auth (аутентификация)

Сервис: `gateway.BackendGateway`. Все RPC без авторизации.

| RPC | Описание |
|-----|----------|
| `Register` | Регистрация |
| `Login` | Вход |
| `Refresh` | Обновление access по refresh-токену |
| `CheckPasswordResetEmail` | Проверка, зарегистрирован ли email |
| `SendPasswordResetCode` | Отправка кода сброса пароля на email |
| `VerifyPasswordReset` | Подтверждение кода и установка нового пароля |

### Register

- **Request:** `RegisterRequest`  
  `first_name`, `last_name`, `email`, `password`, `device_id` (опционально).
- **Response:** `RegisterResponse`  
  `access_token`, `refresh_token`, `user` (UserProfile: id, first_name, last_name, email, username).

### Login

- **Request:** `LoginRequest`  
  `email`, `password`, `device_id` (опционально).
- **Response:** `LoginResponse`  
  `access_token`, `refresh_token`, `user` (UserProfile).

### Refresh

- **Request:** `RefreshRequest`  
  `refresh_token`, `device_id` (опционально).
- **Response:** `RefreshResponse`  
  `access_token`, `user` (UserProfile).

### CheckPasswordResetEmail

- **Request:** `PasswordResetCheckEmailRequest` — `email`.
- **Response:** `PasswordResetCheckEmailResponse` — `exists` (bool).

### SendPasswordResetCode

- **Request:** `PasswordResetSendCodeRequest` — `email`.
- **Response:** `PasswordResetSendCodeResponse` — `sent` (bool).

### VerifyPasswordReset

- **Request:** `PasswordResetVerifyRequest`  
  `email`, `code`, `password` (новый пароль).
- **Response:** `PasswordResetVerifyResponse` — `success` (bool).

---

## 2. User (пользователь и резюме)

Требуется JWT.

| RPC | Описание |
|-----|----------|
| `GetMe` | Текущий пользователь |
| `GetResumeProfile` | Профиль резюме |
| `UpdateResumeProfile` | Обновление профиля резюме |
| `UpdateUserProfile` | Обновление имени, фамилии, email, уведомлений |
| `DeleteAccount` | Удаление аккаунта (с подтверждением паролем) |

### GetMe

- **Request:** `GetMeRequest` — пустой.
- **Response:** `GetMeResponse` — `user` (UserProfile: id, first_name, last_name, email, username, resume_uploaded, total_interviews, completed_interviews, upcoming_interviews, notifications_enabled).

### GetResumeProfile

- **Request:** `GetResumeProfileRequest` — пустой.
- **Response:** `GetResumeProfileResponse`  
  `profile` (ResumeProfile), `status` (DRAFT/CONFIRMED), `version`, `source_material_id`, `confirmed_fields`, `confidence`.

**ResumeProfile:** target_roles[], experience_level, areas[], salary_min, currency, work_format[], skills_top[], education_level, notes.

### UpdateResumeProfile

- **Request:** `UpdateResumeProfileRequest` — `user_id`, `profile` (ResumeProfile).
- **Response:** `UpdateResumeProfileResponse` — `success`.

### UpdateUserProfile

- **Request:** `UpdateUserProfileRequest` — опционально: `first_name`, `last_name`, `email`, `notifications_enabled`.
- **Response:** `UpdateUserProfileResponse` — `success`.

### DeleteAccount

- **Request:** `DeleteAccountRequest` — `password`.
- **Response:** `DeleteAccountResponse` — `deleted`.

---

## 3. Materials (файлы и папки)

Требуется JWT.

| RPC | Описание |
|-----|----------|
| `UploadFile` | Загрузка файла |
| `DownloadFile` | Скачивание по material_id |
| `ListFolder` | Список узлов в папке |
| `CreateFolder` | Создание папки |
| `CreateLink` | Создание ссылки |
| `RenameNode` | Переименование узла |
| `DeleteNode` | Удаление узла |

### UploadFile

- **Request:** `UploadFileRequest`  
  `file_content` (bytes), `filename`, опционально `parent_id`, `name`.
- **Response:** `UploadFileResponse`  
  `material_id`, `name`, `size`, `mime_type`, опционально `download_url`.

### DownloadFile

- **Request:** `DownloadFileRequest` — `material_id`.
- **Response:** `DownloadFileResponse` — `content` (bytes), `filename`, `mime_type`, `size`, опционально `content_base64`.

### ListFolder

- **Request:** `ListFolderRequest` — опционально `parent_id` (корень, если не задан).
- **Response:** `ListFolderResponse` — `nodes[]` (Node: id, user_id, parent_id, type, name, file/link, created_at, updated_at, material_id).

### CreateFolder

- **Request:** `CreateFolderRequest` — `name`, опционально `parent_id`.
- **Response:** `CreateFolderResponse` — `node` (Node).

### CreateLink

- **Request:** `CreateLinkRequest` — `name`, `url`, опционально `title`, `description`, `parent_id`.
- **Response:** `CreateLinkResponse` — `node` (Node).

### RenameNode

- **Request:** `RenameNodeRequest` — `node_id`, `new_name`.
- **Response:** `RenameNodeResponse` — `success`.

### DeleteNode

- **Request:** `DeleteNodeRequest` — `node_id`.
- **Response:** `DeleteNodeResponse` — `success`.

---

## 4. Coach (карьерный коуч и резюме)

Требуется JWT.

| RPC | Описание |
|-----|----------|
| `Ask` | Вопрос к коучу (чат) |
| `ParseResume` | Парсинг резюме по material_id, создание сессии |
| `AnswerResume` | Ответы на уточняющие вопросы по резюме |
| `GetResumeSession` | Получение сессии парсинга резюме |
| `PrepareForVacancy` | Рекомендации по подготовке к вакансии (по ID с HH) |
| `UploadAndParseResume` | Загрузка файла резюме и создание сессии парсинга |
| `ReviewResume` | Оценка резюме + рекомендации по улучшению |
| `GetCoachChatHistory` | Единая лента: чат Ask + вызовы ReviewResume и PrepareForVacancy |
| `ClearChatHistory` | Удаление истории чата с коучом |

### Ask

- **Request:** `AskRequest`  
  опционально `conversation_id`, обязательно `question`, опционально `resume_profile`, `context_chunks[]` (source, title, content).
- **Response:** `AskResponse` — `conversation_id`, `answer`.

### ParseResume

- **Request:** `ParseResumeRequest` — `material_id` (файл резюме из Materials).
- **Response:** `ParseResumeResponse`  
  `session_id`, `draft` (ResumeProfileDraft), `questions[]` (id, text, type, options), `status`.

### UploadAndParseResume

- **Request:** `UploadAndParseResumeRequest` — `file_content` (bytes), `filename`.
- **Response:** `UploadAndParseResumeResponse`  
  `session_id`, `draft` (ResumeProfileDraft), `questions[]`, `status`.

### AnswerResume

- **Request:** `AnswerResumeRequest` — `session_id`, `answers[]` (question_id, value).
- **Response:** `AnswerResumeResponse` — `session_id`, `draft`, `questions[]`, `status`.

### GetResumeSession

- **Request:** `GetResumeSessionRequest` — `session_id`.
- **Response:** `GetResumeSessionResponse` — `session_id`, `draft`, `questions[]`, `status`.

### PrepareForVacancy

- **Request:** `PrepareForVacancyRequest` — `vacancy_id` (ID вакансии с hh.ru).
- **Response:** `PrepareForVacancyResponse` — `recommendations` (текст рекомендаций по подготовке к собеседованию).
- Сервис загружает вакансию с HH API, отправляет в LLM и возвращает рекомендации.

### ReviewResume

- **Request:** `ReviewResumeRequest` — пустой (используется профиль текущего пользователя).
- **Response:** `ReviewResumeResponse` — `score` (0–10), `recommendations` (текст с оценкой и рекомендациями по улучшению).
- Анализирует текущее резюме пользователя (ResumeProfile) и даёт оценку с советами.

### GetCoachChatHistory

- **Request:** `GetCoachChatHistoryRequest` — `page_size` (по умолчанию 50, макс. 200), `page_offset` (смещение для пагинации).
- **Response:** `GetCoachChatHistoryResponse` — `entries[]`, `total_count` (всего записей до пагинации).
- Записи отсортированы **от новых к старым**. Тип записи — `kind`:
  - `COACH_HISTORY_ENTRY_KIND_ASK_USER` / `ASK_ASSISTANT` — сообщения из чата (`conversation_id`, `content`, `created_at`);
  - `COACH_HISTORY_ENTRY_KIND_REVIEW_RESUME` — анализ резюме (`content` = текст ответа, опционально `resume_score`);
  - `COACH_HISTORY_ENTRY_KIND_PREPARE_VACANCY` — подготовка к вакансии (`content`, опционально `vacancy_id`).
- **gRPC (gateway):** `GetCoachChatHistory` — тот же JWT, что и у остальных методов Coach.

### ClearChatHistory

- **Request:** `ClearChatHistoryRequest` — опционально `conversation_id`. Если **не передавать** (или пустая строка) — удаляются **все** диалоги пользователя **и** записи ReviewResume/PrepareForVacancy из ленты; если указан UUID диалога — удаляется только этот диалог (события review/prepare не трогаются).
- **Response:** `ClearChatHistoryResponse` — `ok`, `deleted_conversations` (сколько строк удалено в БД).

---

## 5. Jobs (вакансии и избранное)

Требуется JWT.

| RPC | Описание |
|-----|----------|
| `SearchJobs` | Поиск вакансий (HH) |
| `GetVacancy` | Получить вакансию по ID (для внутренних вызовов) |
| `AddFavorite` | Добавить в избранное |
| `RemoveFavorite` | Убрать из избранного |
| `ListFavorites` | Список избранных вакансий |

### SearchJobs

- **Request:** `SearchJobsRequest` — `page` (по умолчанию 0), `per_page` (по умолчанию 10, макс 100). Параметры поиска задаются на стороне сервиса/конфига.
- **Response:** `SearchJobsResponse` — `items[]` (Vacancy), `found`, `page`, `pages`, `per_page`.

**Vacancy:** id, name, description, salary (from, to, currency), employer (name, logo_url), area (name), alternate_url, experience, is_favorite, archived.

### AddFavorite

- **Request:** `AddFavoriteRequest` — `vacancy_id`.
- **Response:** `AddFavoriteResponse` — `success`.

### RemoveFavorite

- **Request:** `RemoveFavoriteRequest` — `vacancy_id`.
- **Response:** `RemoveFavoriteResponse` — `success`.

### ListFavorites

- **Request:** `ListFavoritesRequest` — пустой.
- **Response:** `ListFavoritesResponse` — `vacancies[]` (Vacancy).

---

## 6. Calendar (события)

Требуется JWT.

| RPC | Описание |
|-----|----------|
| `CreateEvent` | Создать событие |
| `GetEvent` | Получить событие по id |
| `UpdateEvent` | Обновить событие (patch) |
| `DeleteEvent` | Удалить событие |
| `ListEvents` | Список событий в диапазоне с пагинацией |
| `ListUpcoming` | Ближайшие события |

### Типы событий (EventType)

INTERVIEW, CALL, MEETING, TEST_TASK, PREP, DEADLINE, OTHER.

### CreateEvent

- **Request:** `CreateEventRequest` — `event` (Event).
- **Response:** `CreateEventResponse` — `event` (Event).

**Event:** id, title, description, event_type, start_time, end_time (Timestamp), timezone, location, related_vacancy_id, reminder_enabled, reminder_minutes, created_at, updated_at.

### GetEvent

- **Request:** `GetEventRequest` — `id`.
- **Response:** `GetEventResponse` — `event`.

### UpdateEvent

- **Request:** `UpdateEventRequest` — `id`, `patch` (EventPatch: опциональные поля для обновления).
- **Response:** `UpdateEventResponse` — `event`.

### DeleteEvent

- **Request:** `DeleteEventRequest` — `id`.
- **Response:** `DeleteEventResponse` — `success`.

### ListEvents

- **Request:** `ListEventsRequest`  
  `from_time`, `to_time` (Timestamp), `page_size`, `page_token`, `sort` (SORT_START_ASC / SORT_START_DESC).
- **Response:** `ListEventsResponse` — `events[]`, `next_page_token`.

### ListUpcoming

- **Request:** `ListUpcomingRequest` — `limit`, опционально `from_time`.
- **Response:** `ListUpcomingResponse` — `events[]`.

---

## Подключение

- **Порт:** 9090 (gRPC).
- **Сервис:** `gateway.BackendGateway`.
- **Reflection:** включён — можно использовать инструменты вроде grpcurl, BloomRPC, Postman для вызова методов.
- **Пример (grpcurl):**
  - Список методов:  
    `grpcurl -plaintext localhost:9090 list gateway.BackendGateway`
  - Вызов с JWT:  
    `grpcurl -plaintext -H "Authorization: Bearer <token>" localhost:9090 gateway.BackendGateway/GetMe`

Полные определения сообщений см. в `proto/*.proto`.
