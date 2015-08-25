package main

import (
	"github.com/jonasi/pg"
	"github.com/jonasi/pg/migrate"
	"os"
	"strconv"
)

func main() {
	conf := pg.Config{
		Host:           env("PG_HOST", "localhost"),
		Port:           envInt("PG_PORT", 5432),
		Username:       env("PG_USERNAME", ""),
		Password:       env("PG_PASSWORD", ""),
		Database:       env("PG_DATABASE", "postgres"),
		MaxConnections: envInt("PG_MAX_CONNECTIONS", 10),
	}

	db := pg.NewDB(conf)
	migrate.Main(db, os.Args)
}

func env(name string, def string) string {
	val := os.Getenv(name)

	if val == "" {
		return def
	}

	return val
}

func envInt(name string, def int) int {
	val := env(name, "")
	ival, _ := strconv.Atoi(val)

	if ival == 0 {
		return def
	}

	return ival
}
