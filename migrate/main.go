package migrate

import (
	"flag"
	"github.com/jonasi/pg"
	"os"
)

func Main(db *pg.DB, args []string) {
	if args == nil {
		args = os.Args
	}

	var (
		fs   = flag.NewFlagSet(args[0], flag.ExitOnError)
		up   = fs.Int("up", 0, "the number of migrations to run")
		down = fs.Int("down", 0, "the number of migrations to run")
		list = fs.Bool("list", false, "list all migrations")
	)

	fs.Parse(args)

	defaultSet.db = db

	if *up > 0 {
		if err := defaultSet.Up(*up); err != nil {

		}

		return
	}

	if *down > 0 {
		if err := defaultSet.Down(*down); err != nil {

		}

		return

	}

	if *list {

	}
}
