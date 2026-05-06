# Валидации клиентского ввода по сервисам

Документ фиксирует, какие проверки и ограничения применяются к входящим данным клиента (request fields, metadata, pagination, форматы и размеры).

## API Gateway

### Auth interceptor
- `authorization` header обязателен для защищенных методов.
- Формат токена: `Bearer <jwt>` (иначе `invalid authorization format`).
- JWT должен быть валидным (иначе `invalid token` / `invalid token claims` / `invalid user id in token`).

Примечание: сам gateway валидации body-запросов почти не делает, в основном проксирует в backend-сервисы.

---

## User Service

### AuthService/Register
- `email` обязателен.
- `email` нормализуется: `TrimSpace` + `lowercase`.
- Проверка формата email (regex).
- `password` обязателен, минимум `8` символов.

### AuthService/Login
- `email` обязателен.
- `email` нормализуется: `TrimSpace` + `lowercase`.
- Проверка формата email (regex).

### AuthService/Refresh
- `refresh_token` должен существовать, быть активным и не истекшим.

### AuthService/VerifyPasswordReset (legacy flow)
- `code` должен существовать, не быть использованным и не быть просроченным.
- `password` минимум `8` символов.

### UserService/UpdateUserProfile
- Если передан `email`, то проверяется уникальность (`email already taken`).

### UserService/UploadProfilePhoto
- `file_content` обязателен.
- Максимальный размер файла: `5 MB` (`file too large`).
- `filename` обязателен.
- Допустимые расширения: `.jpg`, `.jpeg`, `.png`, `.webp`.
- Допустимые MIME: `image/jpeg`, `image/png`, `image/webp`.
- Проверяется консистентность `mime_type` и расширения файла.

### UserService/GetResumeProfileInternal
- `user_id` обязателен (`> 0`).

### UserService/UpsertResumeProfileInternal
- `user_id` обязателен (`> 0`).
- `profile` обязателен.

### UserService/PatchResumeProfileInternal
- `user_id` обязателен (`> 0`).

### UserService/UpdateResumeProfile
- `profile` обязателен.

### UserService/DeleteAccount
- `password` обязателен.

### UserService/RequestPasswordReset
- `email` обязателен.
- Проверка формата email (regex).
- Проверка cooldown между отправками кода.

### UserService/VerifyPasswordResetCode
- `email` обязателен.
- `code` обязателен.
- Проверка формата email (regex).
- Проверка формата кода: ровно 6 цифр.
- Ограничение по числу попыток (`too many attempts`).

### UserService/ResetPassword
- `email` обязателен.
- `code` обязателен.
- `new_password` обязателен.
- Проверка формата email (regex).
- Проверка формата кода: ровно 6 цифр.
- Минимальная длина нового пароля: `8` символов.
- Ограничение по числу попыток (`too many attempts`).

---

## Job Service

### Metadata/internal auth
- Обязателен `x-user-id` в metadata.
- `x-user-id` должен быть uint-числом.
- Для внутренних вызовов обязателен корректный `x-internal-api-key`.

### JobsService/SearchJobs
- `page` должен быть `>= 0` (иначе `InvalidArgument`).
- `per_page` должен быть в диапазоне `1..100` (иначе `InvalidArgument`).

### JobsService/GetVacancy
- `vacancy_id` обязателен.
- `vacancy_id` должен быть числовым (`regex: ^\d+$`).

### JobsService/AddFavorite
- `vacancy_id` обязателен.
- `vacancy_id` должен быть числовым (`regex: ^\d+$`).

### JobsService/RemoveFavorite
- `vacancy_id` обязателен.
- `vacancy_id` должен быть числовым (`regex: ^\d+$`).

### JobsService/DeleteUserData
- `user_id` обязателен (`> 0`).

---

## Career Coach Service

### Metadata/internal auth
- Обязателен `x-internal-api-key`.
- Обязателен `x-user-id`.
- `x-user-id` должен быть uint-числом.

### CoachService/Ask
- `question` обязателен.
- `question` после `TrimSpace`.
- Максимальная длина `question`: `2000` символов.
- Запрещены управляющие символы (кроме `\n`, `\r`, `\t`).
- Максимум `20` элементов в `context_chunks`.
- Для каждого `context_chunk.content`: обязателен, максимум `4000` символов, без запрещенных управляющих символов.

### CoachService/ParseResume
- `material_id` обязателен.

### CoachService/UploadAndParseResume
- `file_content` обязателен.
- `filename` обязателен.

### CoachService/AnswerResume
- `session_id` обязателен.
- `session_id` должен быть валидным UUID.

### CoachService/GetResumeSession
- `session_id` обязателен.
- `session_id` должен быть валидным UUID.

### CoachService/PrepareForVacancy
- `vacancy_id` обязателен.

### CoachService/AddChatMessage
- `content` после `TrimSpace` должен быть непустым.
- Максимальная длина `content`: `4000` символов.
- Запрещены управляющие символы (кроме `\n`, `\r`, `\t`).
- `owner` должен быть только `USER` или `ASSISTANT`.
- `conversation_id` (если передан) обрезается через `TrimSpace`.

### CoachService/GetCoachChatHistory
- `page_size`:
  - если `<= 0`, ставится `50`,
  - если `> 200`, ставится `200`.
- `page_offset`:
  - если `< 0`, ставится `0`.

### Resume parser (внутренняя обработка Upload/Parse)
- Текст резюме очищается и ограничивается `max_resume_chars` (тримминг по длине, не ошибка).

---

## Materials Service

### Metadata/internal auth
- Обязателен `x-internal-api-key`.
- Обязателен `x-user-id`.
- `x-user-id` должен быть uint-числом.

### MaterialsService/UploadFile
- `file_content` обязателен.
- Максимальный размер файла: `25 MB`.
- `filename` обязателен.
- Максимальная длина `filename`: `120` символов.
- Поддерживаются только расширения: `.pdf`, `.doc`, `.docx`, `.txt`, `.rtf`, `.jpg`, `.jpeg`, `.png`, `.webp`.
- Проверяется MIME-consistency по содержимому файла (`DetectContentType`) и расширению.
- Квота на пользователя: суммарный размер активных файлов не более `100 MB`.
- `name` (или fallback из `filename`) проходит нормализацию/валидацию:
  - `TrimSpace`,
  - не пустой,
  - максимум `120` символов,
  - запрет символов: `< > : " / \ | ? *` и control chars.

### MaterialsService/DownloadFile
- `material_id` обязателен.
- Материал должен быть типа `file` (`material is not a file`).
- Доступ только владельцу (`access denied`).

### MaterialsService/CreateFolder
- `name` проходит нормализацию/валидацию:
  - `TrimSpace`,
  - не пустой,
  - максимум `120` символов,
  - blacklist недопустимых символов.

### MaterialsService/CreateLink
- `name` проходит ту же валидацию, что и в `CreateFolder`.
- `url` обязателен и триммится.

### MaterialsService/RenameNode
- `node_id` обязателен (`!= 0`).
- `new_name` проходит ту же валидацию имени (trim + длина + blacklist).

### MaterialsService/DeleteNode
- `node_id` обязателен (`!= 0`).

### MaterialsService/DeleteByMaterialID
- `material_id` обязателен.

### MaterialsService/RecentFiles
- Лимит результата: максимум `5` файлов (внутренний константный лимит).

---

## Calendar Service

### Metadata/internal auth
- Обязателен `x-internal-api-key`.
- Обязателен `x-user-id`.
- `x-user-id` должен быть uint-числом.

### CalendarService/CreateEvent
- `event` обязателен.
- В `event` обязателен `start_time`.
- В `event` обязателен `end_time`.
- `title` обязателен.
- `title` максимум `200` символов.
- `description` (если передан) максимум `5000` символов.
- `location` (если передан) максимум `255` символов.
- `timezone` (если передан) валидируется как IANA timezone (`time.LoadLocation`).
- `start_time < end_time` (равенство тоже запрещено).

### CalendarService/UpdateEvent
- `patch` обязателен.
- Если передан `title`, он не может быть пустым.
- Если передан `title`, максимум `200` символов.
- Если передан `description`, максимум `5000` символов.
- Если передан `location`, максимум `255` символов.
- Если передан `timezone`, валидируется как IANA timezone.
- После применения patch должно выполняться `start_time < end_time`.

### CalendarService/ListEvents
- Обязательны `from_time` и `to_time`.
- `from_time < to_time`.
- Максимальный диапазон: не более `1 года`.
- `page_size`:
  - если `<= 0`, ставится `50`,
  - если `> 200`, ставится `200`.

### CalendarService/ListUpcoming
- `limit`:
  - если `<= 0`, ставится `10`,
  - если `> 50`, ставится `50`.
