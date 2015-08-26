package main

import (
	"github.com/jonasi/pg/migrate"
)

var _ = migrate.Add("001_create_table",
	func(db *migrate.DB) error {
		return db.Exec(`create table hello ()`)
	},
	func(db *migrate.DB) error {
		return db.Exec(`drop table hello`)
	},
)
