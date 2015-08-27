package pg

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/net/context"
)

type queryInterface interface {
	Get(interface{}, string, ...interface{}) error
	Select(interface{}, string, ...interface{}) error
	Exec(string, ...interface{}) (sql.Result, error)
	Queryx(string, ...interface{}) (*sqlx.Rows, error)
}

type queryer struct {
	impl queryInterface
}

func (q *queryer) Get(ctxt context.Context, dest interface{}, query string, args ...interface{}) error {
	return do(ctxt, query, args, func(query string, args ...interface{}) error {
		return q.impl.Get(dest, query, args...)
	})
}

func (q *queryer) GetMany(ctxt context.Context, dest interface{}, query string, args ...interface{}) error {
	return do(ctxt, query, args, func(query string, args ...interface{}) error {
		return q.impl.Select(dest, query, args...)
	})
}

func (q *queryer) Exec(ctxt context.Context, query string, args ...interface{}) (res sql.Result, err error) {
	err = do(ctxt, query, args, func(query string, args ...interface{}) error {
		var err2 error
		res, err2 = q.impl.Exec(query, args...)

		return err2
	})

	return
}

func (q *queryer) Query(ctxt context.Context, query string, args ...interface{}) (rows *sqlx.Rows, err error) {
	err = do(ctxt, query, args, func(query string, args ...interface{}) error {
		var err2 error
		rows, err2 = q.impl.Queryx(query, args)

		return err2
	})

	return
}

func do(ctxt context.Context, query string, args []interface{}, fn func(query string, args ...interface{}) error) (err error) {
	var (
		st    = time.Now()
		ch    = make(chan error)
		q, ok = FromContext(ctxt)
	)

	defer func() {
		if !ok {
			return
		}

		q.Add(query, args, st, err)
	}()

	go func() {
		ch <- fn(query, args...)
	}()

	select {
	case <-ctxt.Done():
		return ctxt.Err()
	case err := <-ch:
		return err
	}
}
