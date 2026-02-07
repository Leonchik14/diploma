package database

import (
	"context"
	"io/fs"
	"log"

	"user-service/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
)

var DB *pgxpool.Pool

func Connect(databaseURL string) error {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return err
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return err
	}

	if err := pool.Ping(context.Background()); err != nil {
		return err
	}

	DB = pool
	log.Println("Connected to PostgreSQL database")

	return RunMigrations(databaseURL, migrations.FS)
}

func RunMigrations(databaseURL string, migrationsFS fs.FS) error {
	db, err := goose.OpenDBWithDriver("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	goose.SetTableName("goose_user_version")
	goose.SetBaseFS(migrationsFS)
	if err := goose.Up(db, "."); err != nil {
		return err
	}

	log.Println("Database migrations completed")
	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
