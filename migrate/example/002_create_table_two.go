package main

import (
	"github.com/jonasi/pg/migrate"
)

var _ = migrate.Add("002_create_table_two",
	func(db *migrate.DB) error {
		return db.Exec(`create table goodbye ( id int primary key )`)
	},
	func(db *migrate.DB) error {
		return db.Exec(`drop table goodbye`)
	},
)
