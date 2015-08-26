package migrate

import (
	"github.com/jonasi/pg"
)

type DB struct {
	db *pg.DB
}

func (d *DB) Exec(query string, args ...interface{}) error {
	_, err := d.db.Exec(Ctxt, query, args...)
	return err
}
