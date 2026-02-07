package database

import (
	"context"
	"io/fs"
	"log"

	"materials-service/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
)

var DB *pgxpool.Pool

func Connect(dsn string) error {
	config, err := pgxpool.ParseConfig(dsn)
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

	return RunMigrations(dsn, migrations.FS)
}

func RunMigrations(dsn string, migrationsFS fs.FS) error {
	db, err := goose.OpenDBWithDriver("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	goose.SetTableName("goose_materials_version")
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
