package postgres

import (
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
)

type Config struct {
	Host     string
	Login    string
	Password string
	DB       string
	Port     uint16
}

type Repo struct {
	db *sqlx.DB
}

func (r *Repo) Close() error {
	return r.db.Close()
}

func NewPostgresRepo(cfg Config) (*Repo, error) {
	db, err := sqlx.Connect("postgres",
		fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			cfg.Host,
			cfg.Port,
			cfg.Login,
			cfg.Password,
			cfg.DB))
	if err != nil {
		return nil, err
	}

	log.Println("Ping DB...")
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	log.Println("DB OK!")

	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	m, err := migrate.NewWithDatabaseInstance(
		"file://./migrations",
		"postgres", driver)
	if err != nil {
		db.Close()
		return nil, err
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		db.Close()
		return nil, err
	}

	return &Repo{db: db}, nil
}
