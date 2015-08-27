package migrate

import (
	"sort"
	"sync"
	"time"

	"github.com/jonasi/pg"
	"golang.org/x/net/context"
)

var Ctxt = context.Background()

const defaultTable = "schema_migration_log"

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
				direction text not null,
				created_at timestamptz not null default now()
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
			toRun     = migrations{}
			didRun    = migrations{}
			didRunMap = map[string]bool{}
		)

		for i := range log {
			didRunMap[log[i].Name] = log[i].Direction == "up"
		}

		for i := range ms.all {
			val, ok := didRunMap[ms.all[i].Name]

			if !ok || val == false {
				toRun = append(toRun, ms.all[i])
			} else {
				didRun = append(didRun, ms.all[i])
			}
		}

		ms.toRun = toRun
		ms.didRun = didRun
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

type MigrationStatus struct {
	Name   string
	Status string
}

func (ms *Set) Status() ([]MigrationStatus, error) {
	if err := ms.init(); err != nil {
		return nil, err
	}

	m := make([]MigrationStatus, len(ms.didRun)+len(ms.toRun))

	for i := range ms.didRun {
		m[i].Name = ms.didRun[i].Name
		m[i].Status = "run"
	}

	for i := range ms.toRun {
		m[i+len(ms.didRun)].Name = ms.toRun[i].Name
		m[i+len(ms.didRun)].Status = "not run"
	}

	return m, nil
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

	if count > len(ms.toRun) {
		count = len(ms.toRun)
	}

	var (
		db  = &DB{ms.txn}
		upQ = `INSERT INTO ` + ms.Table + ` (name, direction) VALUES ($1, $2)`
	)

	for i := 0; i < count; i++ {
		if err := ms.toRun[i].Up(db); err != nil {
			return ms.handleError(err)
		}

		if err := db.Exec(upQ, ms.toRun[i].Name, "up"); err != nil {
			return err
		}
	}

	err := ms.txn.Commit()
	return ms.handleError(err)
}

func (ms *Set) Down(count int) error {
	if err := ms.init(); err != nil {
		return err
	}

	l := len(ms.didRun)
	if count > l {
		count = l
	}

	var (
		db    = &DB{ms.txn}
		downQ = `INSERT INTO ` + ms.Table + ` (name, direction) VALUES ($1, $2)`
	)

	for i := l - 1; i >= l-count; i-- {
		if err := ms.didRun[i].Down(db); err != nil {
			return ms.handleError(err)
		}

		if err := db.Exec(downQ, ms.didRun[i].Name, "down"); err != nil {
			return err
		}
	}

	err := ms.txn.Commit()
	return ms.handleError(err)
}

func SetDB(db *pg.DB) {
	defaultSet.db = db
}

func Status() ([]MigrationStatus, error) {
	return defaultSet.Status()
}

func Add(name string, up MigrationFn, down MigrationFn) bool {
	defaultSet.Add(name, up, down)
	return true
}

func Up(count int) error {
	return defaultSet.Up(count)
}

func Down(count int) error {
	return defaultSet.Down(count)
}
