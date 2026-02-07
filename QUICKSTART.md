# 🚀 Быстрый старт - Запуск и тестирование системы

## Предварительные требования

- Docker и Docker Compose
- Go 1.22+ (для локальной разработки)
- `grpcurl` (для тестирования gRPC API)
- `protoc` и `protoc-gen-go` (для генерации proto файлов)

### Установка grpcurl

**Windows:**
```powershell
choco install grpcurl
# или
scoop install grpcurl
```

**Linux/macOS:**
```bash
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

## Шаг 1: Подготовка окружения

### 1.0 Настройка переменных окружения

**Важно:** Каждый сервис автоматически загружает переменные из своего `.env` файла при запуске через Docker Compose.

#### Создайте .env файлы для каждого сервиса:

```bash
# Создайте .env файлы из примеров (вручную или скопируйте)
# API Gateway
cp api-gateway/.env.example api-gateway/.env  # если есть .env.example

# User Service
cp user-service/.env.example user-service/.env

# Materials Service  
cp materials-service/.env.example materials-service/.env

# Career Coach Service
cp career-coach-service/.env.example career-coach-service/.env

# Job Service
cp job-service/.env.example job-service/.env

# Calendar Service
cp calendar-service/.env.example calendar-service/.env
```

#### Или создайте .env файлы вручную в каждой папке сервиса:

**`user-service/.env`** (обязательно для SMTP):
```env
DATABASE_URL=postgres://postgres:password@postgres:5432/diploma?sslmode=disable
JWT_PRIVATE_KEY=your-private-key-here
GRPC_PORT=9091
INTERNAL_API_KEY=your-internal-secret-key
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM_EMAIL=your-email@gmail.com
SMTP_FROM_NAME=Interview Prep App
SMTP_TLS=true
```

**`api-gateway/.env`**:
```env
PORT=9090
JWT_PUBLIC_KEY=your-public-key-here
INTERNAL_API_KEY=your-internal-secret-key
```

**`career-coach-service/.env`**:
```env
OPENROUTER_API_KEY=your-openrouter-api-key
DATABASE_URL=postgres://postgres:password@postgres:5432/diploma?sslmode=disable
GRPC_PORT=9093
INTERNAL_API_KEY=your-internal-secret-key
```

**`job-service/.env`**:
```env
GRPC_PORT=9094
HH_APP_TOKEN=your-headhunter-app-token
INTERNAL_API_KEY=your-internal-secret-key
```

**`materials-service/.env`** и **`calendar-service/.env`**:
```env
DATABASE_URL=postgres://postgres:password@postgres:5432/diploma?sslmode=disable
GRPC_PORT=9092  # или 9095 для calendar
INTERNAL_API_KEY=your-internal-secret-key
```

**Примечание:** Docker Compose автоматически загрузит эти `.env` файлы благодаря директиве `env_file:` в `docker-compose.yml`.

### 1.1 Генерация proto файлов

```bash
make proto
```

Или вручную:
```bash
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/*.proto
```

### 1.2 Генерация JWT ключей

**Linux/macOS:**
```bash
bash scripts/generate-keys.sh
```

**Windows (PowerShell):**
```powershell
openssl genrsa -out jwt_private.pem 2048
openssl rsa -in jwt_private.pem -pubout -out jwt_public.pem
```

Затем установите переменные окружения:
```bash
# Linux/macOS
export JWT_PRIVATE_KEY="$(cat jwt_private.pem)"
export JWT_PUBLIC_KEY="$(cat jwt_public.pem)"

# Windows PowerShell
$env:JWT_PRIVATE_KEY = Get-Content jwt_private.pem -Raw
$env:JWT_PUBLIC_KEY = Get-Content jwt_public.pem -Raw
```

### 1.3 Настройка переменных окружения

Создайте файл `.env` в корне проекта или экспортируйте переменные:

```bash
# Обязательные переменные
export INTERNAL_API_KEY="your-internal-secret-key-change-me"
export JWT_PRIVATE_KEY="$(cat jwt_private.pem)"
export JWT_PUBLIC_KEY="$(cat jwt_public.pem)"

# Для password reset (опционально, можно использовать тестовый SMTP)
export SMTP_HOST="smtp.gmail.com"
export SMTP_PORT="587"
export SMTP_USER="your-email@gmail.com"
export SMTP_PASSWORD="your-app-password"
export SMTP_FROM_EMAIL="your-email@gmail.com"
export SMTP_FROM_NAME="Interview Prep App"
export SMTP_TLS="true"

# Для career-coach-service (обязательно)
export OPENROUTER_API_KEY="your-openrouter-api-key"

# Для job-service (обязательно)
export HH_APP_TOKEN="your-headhunter-app-token"
export HH_USER_AGENT="InterviewPrepApp/1.0"
```

**Примечание:** Для тестирования password reset можно использовать сервис типа [Mailtrap](https://mailtrap.io/) или [MailHog](https://github.com/mailhog/MailHog).

## Шаг 2: Запуск всех сервисов

### Вариант A: Docker Compose (рекомендуется)

```bash
docker-compose up -d
```

Проверка статуса:
```bash
docker-compose ps
```

Просмотр логов:
```bash
# Все сервисы
docker-compose logs -f

# Конкретный сервис
docker-compose logs -f user-service
docker-compose logs -f api-gateway
```

### Вариант B: Локальный запуск (для разработки)

Запустите каждый сервис в отдельном терминале:

```bash
# Terminal 1: PostgreSQL (если не используете Docker)
docker run -d --name diploma-postgres \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=password \
  -e POSTGRES_DB=diploma \
  -p 5432:5432 \
  postgres:16-alpine

# Terminal 2: MinIO (если не используете Docker)
docker run -d --name diploma-minio \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  -p 9000:9000 -p 9001:9001 \
  minio/minio server /data --console-address ":9001"

# Terminal 3: user-service
cd user-service
go run cmd/server/main.go

# Terminal 4: materials-service
cd materials-service
go run cmd/materials-service/main.go

# Terminal 5: career-coach-service
cd career-coach-service
go run cmd/server/main.go

# Terminal 6: job-service
cd job-service
go run cmd/server/main.go

# Terminal 7: calendar-service
cd calendar-service
go run cmd/server/main.go

# Terminal 8: api-gateway
cd api-gateway
go run cmd/server/main.go
```

## Шаг 3: Проверка работоспособности

### 3.1 Проверка health checks

```bash
# Проверка через Docker
docker-compose ps

# Проверка gRPC health (если установлен grpc_health_probe)
grpc_health_probe -addr=localhost:9090
```

### 3.2 Проверка доступности портов

```bash
# Проверка портов
netstat -an | grep -E "9090|9091|9092|9093|9094|9095"

# Или через curl (для HTTP сервисов)
curl http://localhost:8081/health  # materials-service
curl http://localhost:8082/health  # career-coach-service
curl http://localhost:8083/health  # job-service
```

## Шаг 4: Тестирование API

### 4.1 Регистрация пользователя

```bash
grpcurl -plaintext \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123"
  }' \
  localhost:9090 \
  gateway.BackendGateway/Register
```

**Ответ:**
```json
{
  "accessToken": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refreshToken": "uuid-refresh-token-here",
  "user": {
    "id": 1,
    "username": "testuser",
    "email": "test@example.com"
  }
}
```

Сохраните `accessToken` для дальнейших запросов:
```bash
export TOKEN="your-access-token-here"
```

### 4.2 Авторизация (Login)

```bash
grpcurl -plaintext \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "deviceId": "device-123"
  }' \
  localhost:9090 \
  gateway.BackendGateway/Login
```

### 4.3 Получение профиля (GetMe)

```bash
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -d '{}' \
  localhost:9090 \
  gateway.BackendGateway/GetMe
```

### 4.4 Восстановление пароля

**Шаг 1: Запрос кода**
```bash
grpcurl -plaintext \
  -d '{"email": "test@example.com"}' \
  localhost:9090 \
  gateway.BackendGateway/RequestPasswordReset
```

**Шаг 2: Проверка кода** (код придет на email)
```bash
grpcurl -plaintext \
  -d '{
    "email": "test@example.com",
    "code": "123456"
  }' \
  localhost:9090 \
  gateway.BackendGateway/VerifyPasswordResetCode
```

**Шаг 3: Установка нового пароля**
```bash
grpcurl -plaintext \
  -d '{
    "email": "test@example.com",
    "code": "123456",
    "newPassword": "newpassword123"
  }' \
  localhost:9090 \
  gateway.BackendGateway/ResetPassword
```

### 4.5 Загрузка файла (Materials)

```bash
# Сначала закодируйте файл в base64
FILE_B64=$(base64 -i resume.pdf)

grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -d "{
    \"fileContent\": \"$FILE_B64\",
    \"filename\": \"resume.pdf\",
    \"name\": \"My Resume\",
    \"parentId\": \"\"
  }" \
  localhost:9090 \
  gateway.BackendGateway/UploadFile
```

**Ответ:**
```json
{
  "node": {
    "id": "uuid-here",
    "name": "My Resume",
    "type": "FILE",
    "size": 45678,
    "mimeType": "application/pdf"
  }
}
```

### 4.6 Парсинг резюме (Career Coach)

```bash
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -d '{
    "materialId": "uuid-from-upload"
  }' \
  localhost:9090 \
  gateway.BackendGateway/ParseResume
```

**Ответ:**
```json
{
  "sessionId": "uuid-session",
  "draft": {
    "targetRoles": ["Software Engineer"],
    "experienceLevel": "middle",
    "salaryMin": 150000
  },
  "questions": [
    {
      "id": "target_roles",
      "text": "Уточните желаемую должность",
      "type": "SINGLE_CHOICE"
    }
  ],
  "status": "AWAITING_USER"
}
```

### 4.7 Ответы на вопросы резюме

```bash
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -d '{
    "sessionId": "uuid-session",
    "answers": [
      {
        "questionId": "target_roles",
        "value": "Backend Developer"
      }
    ]
  }' \
  localhost:9090 \
  gateway.BackendGateway/AnswerResume
```

### 4.8 Поиск вакансий (Jobs)

```bash
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -d '{
    "text": "Go developer",
    "area": "1",
    "salary": 300000,
    "page": 0,
    "perPage": 20
  }' \
  localhost:9090 \
  gateway.BackendGateway/SearchJobs
```

### 4.9 Календарь событий

**Создание события:**
```bash
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -d '{
    "title": "Собеседование в Google",
    "description": "Техническое собеседование",
    "startTime": "2024-12-20T10:00:00Z",
    "endTime": "2024-12-20T11:00:00Z",
    "eventType": "INTERVIEW"
  }' \
  localhost:9090 \
  gateway.BackendGateway/CreateEvent
```

**Список событий:**
```bash
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -d '{
    "startTime": "2024-12-01T00:00:00Z",
    "endTime": "2024-12-31T23:59:59Z"
  }' \
  localhost:9090 \
  gateway.BackendGateway/ListEvents
```

### 4.10 Удаление аккаунта

```bash
grpcurl -plaintext \
  -H "authorization: Bearer $TOKEN" \
  -d '{"password": "password123"}' \
  localhost:9090 \
  gateway.BackendGateway/DeleteAccount
```

## Шаг 5: Полезные команды

### Просмотр логов

```bash
# Все сервисы
docker-compose logs -f

# Конкретный сервис
docker-compose logs -f user-service
docker-compose logs -f api-gateway
docker-compose logs -f materials-service
```

### Перезапуск сервиса

```bash
docker-compose restart user-service
```

### Остановка всех сервисов

```bash
docker-compose down
```

### Остановка с удалением volumes (⚠️ удалит данные БД)

```bash
docker-compose down -v
```

### Пересборка образов

```bash
docker-compose build --no-cache
docker-compose up -d
```

## Шаг 6: Проверка базы данных

### Подключение к PostgreSQL

```bash
# Через Docker
docker exec -it diploma-postgres psql -U postgres -d diploma

# Или напрямую
psql -h localhost -U postgres -d diploma
```

### Полезные SQL запросы

```sql
-- Список пользователей
SELECT id, username, email, created_at, deleted_at FROM users;

-- Список refresh токенов
SELECT user_id, device_id, expires_at, created_at FROM refresh_tokens;

-- Список файлов пользователя
SELECT id, name, type, user_id, created_at FROM nodes WHERE user_id = 1;

-- Список событий календаря
SELECT id, title, event_type, start_time, end_time FROM calendar_events WHERE user_id = 1;
```

## Шаг 7: Проверка MinIO

MinIO Console доступна по адресу: http://localhost:9001

- Username: `minioadmin`
- Password: `minioadmin`

## Порты сервисов

| Сервис | Порт (Docker) | Порт (локально) | Описание |
|--------|---------------|-----------------|----------|
| api-gateway | 9090 | 9090 | Внешний gRPC gateway |
| user-service | 9091 | 9091 | gRPC сервис авторизации |
| materials-service | 9092 | 8081 (HTTP), 9092 (gRPC) | Файловый сервис |
| career-coach-service | 9093 | 8082 (HTTP), 9093 (gRPC) | LLM интеграция |
| job-service | 9094 | 8083 (HTTP), 9094 (gRPC) | Поиск вакансий |
| calendar-service | 9095 | 9095 | Календарь событий |
| PostgreSQL | 5432 | 5432 | База данных |
| MinIO | 9000 | 9000 | Object storage |
| MinIO Console | 9001 | 9001 | Web UI |

## Troubleshooting

### Проблема: Сервисы не запускаются

1. Проверьте логи:
```bash
docker-compose logs user-service
```

2. Проверьте переменные окружения:
```bash
docker-compose config
```

3. Проверьте доступность портов:
```bash
netstat -an | grep 9090
```

### Проблема: Ошибка подключения к БД

1. Убедитесь, что PostgreSQL запущен:
```bash
docker-compose ps postgres
```

2. Проверьте строку подключения в `docker-compose.yml`

### Проблема: Ошибка "invalid internal API key"

Убедитесь, что переменная `INTERNAL_API_KEY` одинакова во всех сервисах.

### Проблема: Ошибка JWT

1. Проверьте, что JWT ключи сгенерированы и установлены
2. Убедитесь, что `JWT_PRIVATE_KEY` и `JWT_PUBLIC_KEY` правильно экспортированы

## Дополнительные ресурсы

- [gRPC API Documentation](README_GRPC.md)
- [User Service README](user-service/README.md)
- [Materials Service README](materials-service/README.md)
- [Career Coach Service README](career-coach-service/README.md)
- [Job Service README](job-service/README.md)
- [Calendar Service README](calendar-service/README.md)
