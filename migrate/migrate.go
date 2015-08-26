package migrate

import (
	"sort"
	"sync"
	"time"

	"github.com/jonasi/pg"
	"golang.org/x/net/context"
)

var Ctxt = context.Background()

const defaultTable = "migration_log"

type Event struct {
	Name      string    `db:"name"`
	Direction string    `db:"direction"`
	CreatedAt time.Time `db:"created_at"`
}

type MigrationFn func(*DB) error

type Migration struct {
	Name string
	Up   MigrationFn
	Down MigrationFn
}

type migrations []Migration

func (m migrations) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m migrations) Less(i, j int) bool {
	return m[i].Name < m[j].Name
}

func (m migrations) Len() int {
	return len(m)
}

var defaultSet = &Set{
	all:   migrations{},
	Table: defaultTable,
}

type Set struct {
	Table    string     // Table name for the schema log
	db       *pg.DB     // db instance
	txn      *pg.Txn    // txn instance
	err      error      // current error
	all      migrations // all migrations in order
	toRun    migrations // migrations that need to be run in order - subset of `all`
	didRun   migrations // migrations that have been run in order - subset of `all`
	log      []Event    // list of all migration events (in order)
	initOnce sync.Once  // protect ms.init()
}

func (ms *Set) init() error {
	var err error

	ms.initOnce.Do(func() {
		if err = ms.db.Open(); err != nil {
			return
		}

		ms.txn, err = ms.db.Begin()

		if err != nil {
			return
		}

		_, err = ms.txn.Exec(Ctxt, `
			create table if not exists `+ms.Table+` (
				name text not null,
				created_at timestamptz not null,
				direction text not null
			)
		`)

		if err != nil {
			return
		}

		var log []Event
		err = ms.txn.GetMany(Ctxt, &log, `select * from `+ms.Table+` order by created_at`)

		if err != nil {
			return
		}

		ms.log = log

		var (
			toRun  = migrations{}
			didRun = map[string]bool{}
		)

		for i := range log {
			didRun[log[i].Name] = log[i].Direction == "up"
		}

		for i := range ms.all {
			val, ok := didRun[ms.all[i].Name]

			if !ok || val == false {
				toRun = append(toRun, ms.all[i])
			}
		}

		ms.toRun = toRun
	})

	return ms.handleError(err)
}

func (ms *Set) handleError(err error) error {
	if err != nil {
		ms.err = err

		if ms.txn != nil {
			ms.txn.Rollback()
			ms.txn = nil
		}
	}

	return err
}

func (ms *Set) Add(name string, up MigrationFn, down MigrationFn) {
	ms.all = append(ms.all, Migration{
		Name: name,
		Up:   up,
		Down: down,
	})

	sort.Sort(ms.all)
}

func (ms *Set) Up(count int) error {
	if err := ms.init(); err != nil {
		return err
	}

	err := ms.txn.Commit()
	return ms.handleError(err)
}

func (ms *Set) Down(count int) error {
	if err := ms.init(); err != nil {
		return err
	}

	err := ms.txn.Commit()
	return ms.handleError(err)
}

func Add(name string, up MigrationFn, down MigrationFn) bool {
	defaultSet.Add(name, up, down)
	return true
}
