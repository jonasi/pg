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
		fs     = flag.NewFlagSet(args[0], flag.ExitOnError)
		up     = fs.Int("up", 0, "the number of migrations to run")
		down   = fs.Int("down", 0, "the number of migrations to rollback")
		dryrun = fs.Bool("dryrun", false, "perform a dry run operation")
	)

	fs.Parse(args[1:])

	defaultSet.db = db

	if *up > 0 {
		if *dryrun {

		}

		if err := defaultSet.Up(*up); err != nil {
			panic(err)
		}

		return
	}

	if *down > 0 {
		if *dryrun {

		}

		if err := defaultSet.Down(*down); err != nil {
			panic(err)
		}

		return
	}

	fs.PrintDefaults()
}
