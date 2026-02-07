# Исправление: relation "nodes" does not exist

**Причина:** Все сервисы (user-service, materials-service, career-coach) используют одну БД `diploma` и по умолчанию одну таблицу goose `goose_db_version`. Версии миграций разных сервисов смешивались: user-service записывал версии 1–4, и materials-service считал свои миграции 1–3 уже применёнными и не создавал таблицу `nodes`.

**Что сделано:** У каждого сервиса теперь своя таблица версий:
- user-service → `goose_user_version`
- materials-service → `goose_materials_version`
- career-coach-service → `goose_coach_version`

**Что сделать:** Пересобрать образы и перезапустить контейнеры. При первом старте materials-service создаст таблицу `goose_materials_version` и применит все свои миграции, включая создание `nodes`.

```bash
docker-compose build materials-service user-service career-coach-service
docker-compose up -d
```

Если после этого ошибка сохранится (например, осталась старая таблица `goose_materials_version` без реальных миграций), сбросьте версии materials:

```bash
docker exec -it diploma-postgres psql -U postgres -d diploma -c "DROP TABLE IF EXISTS goose_materials_version;"
docker-compose restart materials-service
```
