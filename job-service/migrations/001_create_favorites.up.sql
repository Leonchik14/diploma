-- +goose Up
CREATE TABLE IF NOT EXISTS favorite_vacancies (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    vacancy_id VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, vacancy_id)
);

CREATE INDEX IF NOT EXISTS idx_favorite_vacancies_user_id ON favorite_vacancies(user_id);
