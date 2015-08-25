package main

import (
	"github.com/jonasi/pg"
	"github.com/jonasi/pg/migrate"
)

var _ = migrate.Add("001_create_table",
	func(db *pg.DB) error {
		return db.Exec(migrate.Ctxt, `create table hello ()`)
	},
	func(db *pg.DB) error {
		return db.Exec(migrate.Ctxt, `drop table hello`)
	},
)
