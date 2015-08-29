package pg

import (
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
)

type Config struct {
	Host           string
	Port           int
	Username       string
	Password       string
	Database       string
	MaxConnections int
	Logger         pgx.Logger
}

func NewDB(config Config) *DB {
	return &DB{config: config}
}

type DB struct {
	config Config
	queryer
}

func (d *DB) Open() error {
	conf := pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig{
			Host:     d.config.Host,
			Port:     uint16(d.config.Port),
			User:     d.config.Username,
			Password: d.config.Password,
			Database: d.config.Database,
			Logger:   d.config.Logger,
		},
		MaxConnections: d.config.MaxConnections,
	}

	p, err := pgx.NewConnPool(conf)

	if err != nil {
		return err
	}

	db, err := stdlib.OpenFromConnPool(p)

	if err != nil {
		return err
	}

	dbx := sqlx.NewDb(db, "pgx")
	d.queryer = queryer{impl: dbx}

	return nil
}

func (d *DB) Close() error {
	return d.queryer.impl.(*sqlx.DB).Close()
}
