package migrate

import (
	"sort"
	"sync"
	"time"

	"github.com/jonasi/pg"
	"golang.org/x/net/context"
)

type Logger interface {
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Errorf(string, ...interface{})
}

type nullLogger struct{}

func (n *nullLogger) Debugf(string, ...interface{}) {}
func (n *nullLogger) Infof(string, ...interface{})  {}
func (n *nullLogger) Errorf(string, ...interface{}) {}

var defaultLogger = &nullLogger{}

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

var defaultSet = NewSet(defaultTable, defaultLogger)

func NewSet(table string, logger Logger) *Set {
	if table == "" {
		table = defaultTable
	}

	if logger == nil {
		logger = defaultLogger
	}

	return &Set{
		all:    migrations{},
		table:  table,
		logger: logger,
	}
}

type Set struct {
	table    string     // Table name for the schema log
	db       *pg.DB     // db instance
	txn      *DB        // txn instance
	all      migrations // all migrations in order
	toRun    migrations // migrations that need to be run in order - subset of `all`
	didRun   migrations // migrations that have been run in order - subset of `all`
	log      []Event    // list of all migration events (in order)
	initErr  error      // the result of the init process
	initOnce sync.Once  // protect ms.init()
	logger   Logger
}

func (ms *Set) init() error {
	ms.initOnce.Do(func() {
		if ms.initErr = ms.db.Open(); ms.initErr != nil {
			return
		}

		var txn *pg.Txn
		txn, ms.initErr = ms.db.Begin()

		if ms.initErr != nil {
			return
		}

		ms.txn = &DB{txn, ms.logger}

		ms.initErr = ms.txn.Exec(`
			create table if not exists ` + ms.table + ` (
				name text not null,
				direction text not null,
				created_at timestamptz not null default now()
			)
		`)

		if ms.initErr != nil {
			return
		}

		ms.initErr = ms.txn.Exec(`
			create or replace view ` + ms.table + `_current as (
				select name, created_at from (
					select distinct on (name) * from ` + ms.table + ` order by name, created_at desc
				) x where direction = 'up' order by created_at
			)
		`)

		if ms.initErr != nil {
			return
		}

		var log []Event
		ms.initErr = ms.txn.db.GetMany(Ctxt, &log, `select * from `+ms.table+` order by created_at`)

		if ms.initErr != nil {
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

	return ms.initErr
}

func (ms *Set) run(commit bool, fn func() error) (err error) {
	defer func() {
		if ms.txn.db == nil {
			return
		}

		if !commit || err != nil {
			ms.txn.db.Rollback()
		} else {
			err = ms.txn.db.Commit()
		}
	}()

	if err := ms.init(); err != nil {
		return err
	}

	return fn()
}

type MigrationStatus struct {
	Name   string
	Status string
}

func (ms *Set) Status() (m []MigrationStatus, err error) {
	err = ms.run(true, func() error {
		m = make([]MigrationStatus, len(ms.didRun)+len(ms.toRun))

		for i := range ms.didRun {
			m[i].Name = ms.didRun[i].Name
			m[i].Status = "run"
		}

		for i := range ms.toRun {
			m[i+len(ms.didRun)].Name = ms.toRun[i].Name
			m[i+len(ms.didRun)].Status = "not run"
		}

		return nil
	})

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

func (ms *Set) Up(count int, commit bool) error {
	return ms.run(commit, func() error {
		if count > len(ms.toRun) {
			count = len(ms.toRun)
		}

		upQ := `INSERT INTO ` + ms.table + ` (name, direction) VALUES ($1, $2)`

		for i := 0; i < count; i++ {
			ms.logger.Infof("Running %s.Up", ms.toRun[i].Name)

			if err := ms.toRun[i].Up(ms.txn); err != nil {
				return err
			}

			if err := ms.txn.Exec(upQ, ms.toRun[i].Name, "up"); err != nil {
				return err
			}
		}

		return nil
	})
}

func (ms *Set) Down(count int, commit bool) error {
	return ms.run(commit, func() error {
		l := len(ms.didRun)
		if count > l {
			count = l
		}

		downQ := `INSERT INTO ` + ms.table + ` (name, direction) VALUES ($1, $2)`

		for i := l - 1; i >= l-count; i-- {
			ms.logger.Infof("Running %s.Down", ms.didRun[i].Name)

			if err := ms.didRun[i].Down(ms.txn); err != nil {
				return err
			}

			if err := ms.txn.Exec(downQ, ms.didRun[i].Name, "down"); err != nil {
				return err
			}
		}

		return nil
	})
}

func SetDB(db *pg.DB) {
	defaultSet.db = db
}

func SetLogger(l Logger) {
	defaultSet.logger = l
}

func SetTable(table string) {
	defaultSet.table = table
}

func Status() ([]MigrationStatus, error) {
	return defaultSet.Status()
}

func Add(name string, up MigrationFn, down MigrationFn) bool {
	defaultSet.Add(name, up, down)
	return true
}

func Up(count int, commit bool) error {
	return defaultSet.Up(count, commit)
}

func Down(count int, commit bool) error {
	return defaultSet.Down(count, commit)
}
