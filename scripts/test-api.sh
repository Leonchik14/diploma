#!/bin/bash

# Скрипт для тестирования основных API эндпоинтов
# Требует: grpcurl

set -e

GATEWAY="localhost:9090"
EMAIL="test@example.com"
PASSWORD="password123"
USERNAME="testuser"

echo "🚀 Тестирование API системы"
echo "================================"
echo ""

# Цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Функция для выполнения gRPC запроса
grpc_call() {
    local method=$1
    local data=$2
    local headers=$3
    
    if [ -z "$headers" ]; then
        grpcurl -plaintext -d "$data" "$GATEWAY" "$method"
    else
        grpcurl -plaintext -H "$headers" -d "$data" "$GATEWAY" "$method"
    fi
}

# Тест 1: Регистрация
echo -e "${YELLOW}1. Регистрация пользователя${NC}"
REGISTER_RESPONSE=$(grpc_call "gateway.BackendGateway/Register" "{
  \"username\": \"$USERNAME\",
  \"email\": \"$EMAIL\",
  \"password\": \"$PASSWORD\"
}")

echo "$REGISTER_RESPONSE" | jq '.' || echo "$REGISTER_RESPONSE"
echo ""

# Извлечение токена
TOKEN=$(echo "$REGISTER_RESPONSE" | jq -r '.accessToken' 2>/dev/null || echo "")

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    echo -e "${RED}❌ Не удалось получить токен. Пробуем логин...${NC}"
    
    # Попробуем логин
    LOGIN_RESPONSE=$(grpc_call "gateway.BackendGateway/Login" "{
      \"email\": \"$EMAIL\",
      \"password\": \"$PASSWORD\",
      \"deviceId\": \"test-device\"
    }")
    
    TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.accessToken' 2>/dev/null || echo "")
fi

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    echo -e "${RED}❌ Не удалось получить токен. Прерываем тестирование.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Токен получен${NC}"
echo ""

# Тест 2: GetMe
echo -e "${YELLOW}2. Получение профиля (GetMe)${NC}"
grpc_call "gateway.BackendGateway/GetMe" "{}" "authorization: Bearer $TOKEN" | jq '.' || echo "Ошибка"
echo ""

# Тест 3: RequestPasswordReset
echo -e "${YELLOW}3. Запрос восстановления пароля${NC}"
grpc_call "gateway.BackendGateway/RequestPasswordReset" "{
  \"email\": \"$EMAIL\"
}" | jq '.' || echo "Ошибка"
echo ""

# Тест 4: ListFolder (Materials)
echo -e "${YELLOW}4. Список файлов (ListFolder)${NC}"
grpc_call "gateway.BackendGateway/ListFolder" "{
  \"parentId\": \"\"
}" "authorization: Bearer $TOKEN" | jq '.' || echo "Ошибка"
echo ""

# Тест 5: ListEvents (Calendar)
echo -e "${YELLOW}5. Список событий календаря${NC}"
grpc_call "gateway.BackendGateway/ListEvents" "{
  \"startTime\": \"2024-01-01T00:00:00Z\",
  \"endTime\": \"2024-12-31T23:59:59Z\"
}" "authorization: Bearer $TOKEN" | jq '.' || echo "Ошибка"
echo ""

# Тест 6: ListUpcoming (Calendar)
echo -e "${YELLOW}6. Предстоящие события${NC}"
grpc_call "gateway.BackendGateway/ListUpcoming" "{
  \"limit\": 10
}" "authorization: Bearer $TOKEN" | jq '.' || echo "Ошибка"
echo ""

# Тест 7: ListFavorites (Jobs)
echo -e "${YELLOW}7. Избранные вакансии${NC}"
grpc_call "gateway.BackendGateway/ListFavorites" "{}" "authorization: Bearer $TOKEN" | jq '.' || echo "Ошибка"
echo ""

echo -e "${GREEN}✅ Базовое тестирование завершено${NC}"
echo ""
echo "Для полного тестирования используйте команды из QUICKSTART.md"
