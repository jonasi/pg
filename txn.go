package pg

import (
	"github.com/jmoiron/sqlx"
	"log"
)

func (d *DB) Begin() (*Txn, error) {
	tx, err := d.queryer.impl.(*sqlx.DB).Beginx()

	if err != nil {
		return nil, err
	}

	return &Txn{queryer{tx}}, nil
}

func (d *DB) InTxn(fn func(*Txn) error) (err error) {
	tx, err := d.Begin()

	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			if err2 := tx.Rollback(); err2 != nil {
				log.Printf("Rollback error: %v\n", err2)
			}
		} else {
			err = tx.Commit()
		}
	}()

	return fn(tx)
}

type Txn struct {
	queryer
}

func (t *Txn) Commit() error {
	return t.queryer.impl.(*sqlx.Tx).Commit()
}

func (t *Txn) Rollback() error {
	return t.queryer.impl.(*sqlx.Tx).Rollback()
}
