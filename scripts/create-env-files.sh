#!/bin/bash

# Скрипт для создания пустых .env файлов в каждом сервисе
# Это нужно, если вы хотите использовать env_file в docker-compose.yml

echo "Создание пустых .env файлов для всех сервисов..."

# Создаем пустые .env файлы
touch api-gateway/.env
touch user-service/.env
touch materials-service/.env
touch career-coach-service/.env
touch job-service/.env
touch calendar-service/.env

echo "✅ Созданы пустые .env файлы"
echo ""
echo "Теперь заполните их значениями или используйте общий .env в корне проекта"
