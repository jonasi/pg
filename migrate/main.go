package migrate

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/jonasi/pg"
)

type countVar struct {
	value int
	set   bool
}

func (u *countVar) String() string {
	return fmt.Sprintf("%d", u.value)
}

func (u *countVar) Set(str string) error {
	u.set = true

	// use the default value
	if str == "true" {
		return nil
	}

	n, err := strconv.Atoi(str)
	u.value = n

	return err
}

func (u *countVar) IsBoolFlag() bool { return true }

func Main(db *pg.DB, args []string) {
	if args == nil {
		args = os.Args
	}

	var (
		fs     = flag.NewFlagSet(args[0], flag.ExitOnError)
		up     = countVar{math.MaxInt32, false}
		down   = countVar{1, false}
		dryrun = fs.Bool("dryrun", false, "perform a dry run operation")
		status = fs.Bool("status", false, "print the current status of each migration")
	)

	fs.Var(&up, "up", "the number of migrations to run")
	fs.Var(&down, "down", "the number of migrations to rollback")

	fs.Parse(args[1:])

	SetDB(db)

	if *status {
		st, err := Status()
		if err != nil {
			panic(err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, '\t', 0)
		fmt.Fprintln(w, "name\tstatus")

		for i := range st {
			fmt.Fprintln(w, st[i].Name+"\t"+st[i].Status)
		}

		w.Flush()

		return
	}

	if up.set {
		if *dryrun {

		}

		if err := Up(int(up.value)); err != nil {
			panic(err)
		}

		return
	}

	if down.set {
		if *dryrun {

		}

		if err := Down(down.value); err != nil {
			panic(err)
		}

		return
	}

	fs.PrintDefaults()
}
