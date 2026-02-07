# Diploma Microservices

Микросервисная система для мобильного приложения подготовки к собеседованиям.

## 🚀 Быстрый старт

**Для быстрого запуска и тестирования системы см. [QUICKSTART.md](QUICKSTART.md)**

Краткая версия:
1. Генерация proto: `make proto`
2. Генерация JWT ключей: `bash scripts/generate-keys.sh` (или вручную через openssl)
3. Установка переменных окружения (см. QUICKSTART.md)
4. Запуск: `docker-compose up -d`
5. Тестирование: `bash scripts/test-api.sh` (Linux/macOS) или `powershell scripts/test-api.ps1` (Windows)

## Сервисы

1. **user-service** (8080) - Авторизация, профили, хранение ResumeProfile
2. **materials-service** (8081) - Файловая система пользователя (files/folders/links) + MinIO
3. **career-coach-service** (8082) - Интеграция с LLM через OpenRouter, парсинг резюме, чат
4. **job-service** (8083) - Поиск вакансий через HH API

## Мобильный флоу: Резюме и вакансии

1. **Загрузка резюме** → materials-service (получить `material_id`).
2. **Парсинг** → career-coach ParseResume(material_id): профиль сохраняется в user-service как DRAFT, создаётся сессия, возвращаются `session_id`, текущий профиль и вопросы.
3. **Уточнение** → career-coach AnswerResume(session_id, answers): обновление профиля в user-service (patch), при готовности ключевых полей — status CONFIRMED.
4. **Профиль** → user-service GetResumeProfile (для мобилки: статус, версия, confirmed_fields, confidence).
5. **Поиск вакансий** → job-service SearchJobs: читает профиль из user-service, строит запрос к HH; если профиль отсутствует или `target_roles` пустые — возвращается ошибка `InvalidArgument: resume profile incomplete`.

---

### 1. Загрузка резюме в materials-service

```bash
curl -X POST http://localhost:8081/api/v1/files/upload \
  -H "Authorization: Bearer <access_token>" \
  -H "X-User-ID: 1" \
  -F "file=@resume.pdf" \
  -F "name=resume.pdf" \
  -F "parent_id="
```

**Ответ:**
```json
{
  "material_id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "resume.pdf",
  "size": 45678,
  "mime_type": "application/pdf",
  "download_url": "http://..."
}
```

### 2. Парсинг резюме через career-coach-service

```bash
curl -X POST http://localhost:8082/api/v1/resume/parse \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 1" \
  -d '{
    "material_id": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

**Ответ:**
```json
{
  "session_id": "uuid-here",
  "draft": {
    "target_roles": ["Software Engineer"],
    "experience_level": "middle",
    "areas": [{"id": "1", "name": "Moscow"}],
    "salary_min": 150000,
    "currency": "RUR",
    "work_format": ["remote"],
    "skills_top": ["Go", "PostgreSQL"],
    "confidence": {"target_roles": 0.8}
  },
  "questions": [
    {
      "id": "target_roles",
      "text": "Уточните желаемую должность",
      "type": "single_choice",
      "options": ["Backend Developer", "Full Stack Developer"]
    }
  ],
  "status": "awaiting_user"
}
```

### 3. Ответы на вопросы

```bash
curl -X POST http://localhost:8082/api/v1/resume/answer \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 1" \
  -d '{
    "session_id": "uuid-here",
    "answers": [
      {"question_id": "target_roles", "value": "Backend Developer"},
      {"question_id": "experience_level", "value": "middle"}
    ]
  }'
```

**Ответ:**
```json
{
  "session_id": "uuid-here",
  "draft": {...},
  "questions": [],
  "status": "completed"
}
```

### 4. Получение профиля резюме из user-service

```bash
curl -X GET http://localhost:8080/api/v1/user/me/resume-profile \
  -H "Authorization: Bearer <access_token>"
```

**Ответ:**
```json
{
  "target_roles": ["Backend Developer"],
  "experience_level": "middle",
  "areas": [{"id": "1", "name": "Moscow"}],
  "salary_min": 150000,
  "currency": "RUR",
  "work_format": ["remote"],
  "skills_top": ["Go", "PostgreSQL"]
}
```

### 5. Поиск вакансий через job-service

```bash
curl -X GET "http://localhost:8083/api/v1/jobs/search" \
  -H "X-User-ID: 1"
```

**Ответ:**
```json
{
  "items": [
    {
      "id": "123456",
      "name": "Backend Developer",
      "description": "...",
      "salary": {"from": 150000, "currency": "RUR"},
      "employer": {"name": "Company Name"},
      "area": {"name": "Moscow"},
      "alternate_url": "https://hh.ru/vacancy/123456"
    }
  ],
  "found": 150,
  "pages": 8,
  "page": 0
}
```

## Запуск через Docker Compose

```bash
export INTERNAL_API_KEY=your-internal-secret-key
export OPENROUTER_API_KEY=your-openrouter-key
export HH_APP_TOKEN=your-hh-token

docker-compose up -d
```

## Переменные окружения

### Общие
- `INTERNAL_API_KEY` - секретный ключ для внутренних вызовов между сервисами

### user-service
- `DATABASE_URL` - PostgreSQL connection string
- `JWT_SECRET` - секрет для JWT токенов
- `PORT` - порт сервиса (default: 8080)

### materials-service
- `DB_DSN` - PostgreSQL connection string
- `MINIO_ENDPOINT` - адрес MinIO
- `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY` - ключи MinIO
- `HTTP_PORT` - порт сервиса (default: 8081)

### career-coach-service
- `DATABASE_URL` - PostgreSQL connection string
- `OPENROUTER_API_KEY` - ключ API OpenRouter
- `PARSE_MODEL`, `COACH_MODEL` - модели LLM
- `MATERIALS_SERVICE_URL` - URL/адрес materials-service (gRPC)
- `USER_SERVICE_GRPC` - адрес user-service по gRPC (default: user-service:9091)
- `HTTP_PORT` / `GRPC_PORT` - порты сервиса (default: 8082 / 9093)

### job-service
- `USER_SERVICE_GRPC` (или `USER_SERVICE_URL`) - адрес user-service по gRPC
- `HH_APP_TOKEN` - токен приложения HeadHunter
- `HH_USER_AGENT` - User-Agent для HH API
- `HH_HOST` - хост HH API (default: api.hh.ru)
- `HTTP_PORT` - порт сервиса (default: 8083)
