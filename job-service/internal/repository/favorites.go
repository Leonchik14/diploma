package repository

import (
	"context"
	"job-service/internal/database"
)

type FavoritesRepo struct{}

func NewFavoritesRepo() *FavoritesRepo {
	return &FavoritesRepo{}
}

func (r *FavoritesRepo) Add(ctx context.Context, userID int64, vacancyID string) error {
	_, err := database.DB.Exec(ctx,
		`INSERT INTO favorite_vacancies (user_id, vacancy_id)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id, vacancy_id) DO NOTHING`,
		userID, vacancyID,
	)
	return err
}

func (r *FavoritesRepo) Remove(ctx context.Context, userID int64, vacancyID string) error {
	_, err := database.DB.Exec(ctx,
		`DELETE FROM favorite_vacancies WHERE user_id = $1 AND vacancy_id = $2`,
		userID, vacancyID,
	)
	return err
}

func (r *FavoritesRepo) ListIDs(ctx context.Context, userID int64) ([]string, error) {
	rows, err := database.DB.Query(ctx,
		`SELECT vacancy_id FROM favorite_vacancies WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *FavoritesRepo) DeleteByUser(ctx context.Context, userID int64) error {
	_, err := database.DB.Exec(ctx,
		`DELETE FROM favorite_vacancies WHERE user_id = $1`,
		userID,
	)
	return err
}
