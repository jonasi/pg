package migrate

import (
	"github.com/jonasi/pg"
)

type DB struct {
	db *pg.Txn
	l  Logger
}

func (d *DB) Exec(query string, args ...interface{}) error {
	_, err := d.db.Exec(Ctxt, query, args...)
	return err
}

type Query struct {
	query  string
	params []interface{}
}

func Q(query string, params ...interface{}) Query {
	return Query{query, params}
}

func (d *DB) ExecMany(queries ...Query) error {
	for i := range queries {
		if err := d.Exec(queries[i].query, queries[i].params...); err != nil {
			return err
		}
	}

	return nil
}
