# PowerShell скрипт для тестирования основных API эндпоинтов
# Требует: grpcurl

$ErrorActionPreference = "Stop"

$GATEWAY = "localhost:9090"
$EMAIL = "test@example.com"
$PASSWORD = "password123"
$USERNAME = "testuser"

Write-Host "🚀 Тестирование API системы" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan
Write-Host ""

# Функция для выполнения gRPC запроса
function Invoke-GrpcCall {
    param(
        [string]$Method,
        [string]$Data,
        [string]$Headers = ""
    )
    
    if ($Headers -eq "") {
        $result = grpcurl -plaintext -d $Data $GATEWAY $Method 2>&1
    } else {
        $result = grpcurl -plaintext -H $Headers -d $Data $GATEWAY $Method 2>&1
    }
    
    return $result
}

# Тест 1: Регистрация
Write-Host "1. Регистрация пользователя" -ForegroundColor Yellow
$registerData = @{
    username = $USERNAME
    email = $EMAIL
    password = $PASSWORD
} | ConvertTo-Json -Compress

$REGISTER_RESPONSE = Invoke-GrpcCall -Method "gateway.BackendGateway/Register" -Data $registerData
Write-Host $REGISTER_RESPONSE
Write-Host ""

# Извлечение токена
try {
    $responseObj = $REGISTER_RESPONSE | ConvertFrom-Json
    $TOKEN = $responseObj.accessToken
} catch {
    Write-Host "❌ Не удалось получить токен. Пробуем логин..." -ForegroundColor Red
    
    $loginData = @{
        email = $EMAIL
        password = $PASSWORD
        deviceId = "test-device"
    } | ConvertTo-Json -Compress
    
    $LOGIN_RESPONSE = Invoke-GrpcCall -Method "gateway.BackendGateway/Login" -Data $loginData
    try {
        $loginObj = $LOGIN_RESPONSE | ConvertFrom-Json
        $TOKEN = $loginObj.accessToken
    } catch {
        Write-Host "❌ Не удалось получить токен. Прерываем тестирование." -ForegroundColor Red
        exit 1
    }
}

if ([string]::IsNullOrEmpty($TOKEN)) {
    Write-Host "❌ Не удалось получить токен. Прерываем тестирование." -ForegroundColor Red
    exit 1
}

Write-Host "✅ Токен получен" -ForegroundColor Green
Write-Host ""

# Тест 2: GetMe
Write-Host "2. Получение профиля (GetMe)" -ForegroundColor Yellow
$getMeData = "{}"
$getMeHeaders = "authorization: Bearer $TOKEN"
Invoke-GrpcCall -Method "gateway.BackendGateway/GetMe" -Data $getMeData -Headers $getMeHeaders
Write-Host ""

# Тест 3: RequestPasswordReset
Write-Host "3. Запрос восстановления пароля" -ForegroundColor Yellow
$resetData = @{
    email = $EMAIL
} | ConvertTo-Json -Compress
Invoke-GrpcCall -Method "gateway.BackendGateway/RequestPasswordReset" -Data $resetData
Write-Host ""

# Тест 4: ListFolder (Materials)
Write-Host "4. Список файлов (ListFolder)" -ForegroundColor Yellow
$listFolderData = @{
    parentId = ""
} | ConvertTo-Json -Compress
Invoke-GrpcCall -Method "gateway.BackendGateway/ListFolder" -Data $listFolderData -Headers $getMeHeaders
Write-Host ""

# Тест 5: ListEvents (Calendar)
Write-Host "5. Список событий календаря" -ForegroundColor Yellow
$listEventsData = @{
    startTime = "2024-01-01T00:00:00Z"
    endTime = "2024-12-31T23:59:59Z"
} | ConvertTo-Json -Compress
Invoke-GrpcCall -Method "gateway.BackendGateway/ListEvents" -Data $listEventsData -Headers $getMeHeaders
Write-Host ""

# Тест 6: ListUpcoming (Calendar)
Write-Host "6. Предстоящие события" -ForegroundColor Yellow
$upcomingData = @{
    limit = 10
} | ConvertTo-Json -Compress
Invoke-GrpcCall -Method "gateway.BackendGateway/ListUpcoming" -Data $upcomingData -Headers $getMeHeaders
Write-Host ""

# Тест 7: ListFavorites (Jobs)
Write-Host "7. Избранные вакансии" -ForegroundColor Yellow
$favoritesData = "{}"
Invoke-GrpcCall -Method "gateway.BackendGateway/ListFavorites" -Data $favoritesData -Headers $getMeHeaders
Write-Host ""

Write-Host "✅ Базовое тестирование завершено" -ForegroundColor Green
Write-Host ""
Write-Host "Для полного тестирования используйте команды из QUICKSTART.md"
