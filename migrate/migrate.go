package migrate

import (
	"github.com/jonasi/pg"
	"golang.org/x/net/context"
	"sort"
	"sync"
	"time"
)

var Ctxt = context.Background()

const defaultTable = "migration_log"

type migrationEvent struct {
	Name      string    `db:"name"`
	Direction string    `db:"direction"`
	CreatedAt time.Time `db:"created_at"`
}

type MigrationFn func(*pg.DB) error

type Migration struct {
	Name string
	Up   func(*pg.DB) error
	Down func(*pg.DB) error
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

var defaultSet = &MigrationSet{
	allMigrations: migrations{},
	Table:         defaultTable,
}

type MigrationSet struct {
	Table               string
	db                  *pg.DB
	txn                 *pg.Txn
	err                 error
	allMigrations       migrations
	completedMigrations []migrationEvent
	initOnce            sync.Once
}

func (ms *MigrationSet) init() error {
	var err error

	ms.initOnce.Do(func() {
		ms.txn, err = ms.db.Begin()

		if err != nil {
			return
		}

		err = ms.txn.Exec(Ctxt, `
			create table if not exists `+ms.Table+` (
				name text not null,
				created_at timestamptz not null,
				direction text not null
			)
		`)

		if err != nil {
			return
		}

		var migrations []migrationEvent

		err = ms.txn.GetMany(Ctxt, &migrations, `select * from `+ms.Table+` order by created_at`)

		if err != nil {
			return
		}
	})

	return ms.handleError(err)
}

func (ms *MigrationSet) handleError(err error) error {
	if err != nil {
		ms.err = err

		if ms.txn != nil {
			ms.txn.Rollback()
			ms.txn = nil
		}
	}

	return err
}

func (ms *MigrationSet) Add(name string, up MigrationFn, down MigrationFn) {
	ms.allMigrations = append(ms.allMigrations, Migration{
		Name: name,
		Up:   up,
		Down: down,
	})

	sort.Sort(ms.allMigrations)
}

func (ms *MigrationSet) Up(count int) error {
	if err := ms.init(); err != nil {
		return err
	}

	return nil
}

func (ms *MigrationSet) Down(count int) error {
	if err := ms.init(); err != nil {
		return err
	}

	return nil
}

func Add(name string, up MigrationFn, down MigrationFn) bool {
	defaultSet.Add(name, up, down)
	return true
}
