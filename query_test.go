package pg

import (
	"os"
	"strconv"
	"testing"
	"time"

	"golang.org/x/net/context"
)

var todo = context.Background()

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

func config() Config {
	return Config{
		Host:           env("PG_HOST", "localhost"),
		Port:           envInt("PG_PORT", 5432),
		Username:       env("PG_USERNAME", ""),
		Password:       env("PG_PASSWORD", ""),
		Database:       env("PG_DATABASE", "postgres"),
		MaxConnections: envInt("PG_MAX_CONNECTIONS", 10),
	}
}

func withDB(t *testing.T, fn func(*DB)) {
	c := config()
	db := NewDB(c)

	if err := db.Open(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	fn(db)
}

func TestConnect(t *testing.T) {
	c := config()
	db := NewDB(c)

	if err := db.Open(); err != nil {
		t.Fatal(err)
	}

	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestQuery(t *testing.T) {
	withDB(t, func(d *DB) {
		var vals []struct {
			Schemaname string
			Tablename  string
		}

		if err := d.GetMany(todo, &vals, `SELECT schemaname, tablename FROM pg_catalog.pg_tables ORDER BY schemaname, tablename`); err != nil {
			t.Fatal(err)
		}
	})
}

func TestEvent(t *testing.T) {
	withDB(t, func(d *DB) {
		var vals []struct {
			Schemaname string
			Tablename  string
		}

		qs := NewQuerySet()
		ctxt := NewContext(todo, qs)

		if err := d.GetMany(ctxt, &vals, `SELECT schemaname, tablename FROM pg_catalog.pg_tables ORDER BY schemaname, tablename`); err != nil {
			t.Fatal(err)
		}

		if len(qs.q) != 1 {
			t.Fatalf("Invalid query count %d", len(qs.q))
		}
	})
}

func TestEventError(t *testing.T) {
	withDB(t, func(d *DB) {
		qs := NewQuerySet()
		ctxt := NewContext(todo, qs)

		if err := d.Exec(ctxt, `NONSENSE`); err == nil {
			t.Fatal("Expected error")
		}

		if len(qs.q) != 1 {
			t.Fatalf("Invalid query count %d", len(qs.q))
		}

		if qs.q[0].Error == nil {
			t.Fatal("Expected error")
		}
	})
}

func TestCancel(t *testing.T) {
	withDB(t, func(d *DB) {
		qs := NewQuerySet()
		ctxt := NewContext(todo, qs)
		ctxt, _ = context.WithDeadline(ctxt, time.Now().Add(250*time.Millisecond))

		st := time.Now()

		if err := d.Exec(ctxt, `select pg_sleep_for('10 seconds')`); err != nil {
			if err != ctxt.Err() {
				t.Fatal(err)
			}
		}

		if time.Since(st) > 10*time.Second {
			t.Fatal("took longer than expected")
		}
	})
}
