package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
)

const version = "0.1"

type command struct {
	flagSet *flag.FlagSet
	run     func(args []string) error
}

// Main examines the args and delegates to the specified subcommand.
//
// If no subcommand was specified, we default to the "table" subcommand.
func main() {
	subcommands := map[string]command{ // Available subcommands
		"table": tableCmd(),
		"tree":  treeCmd(),
	}

	// --- Handle top-level flags ---
	mainFlagSet := flag.NewFlagSet("git-who", flag.ExitOnError)
	versionFlag := mainFlagSet.Bool("version", false, "Print version and exit")

	mainFlagSet.Usage = func() {
		fmt.Println("Usage: git-who [options...] [subcommand]")
		fmt.Println("git-who tallies authorship")
		mainFlagSet.PrintDefaults()
	}

	mainFlagSet.Parse(os.Args[1:])

	if *versionFlag {
		fmt.Printf("%s\n", version)
		return
	}

	args := mainFlagSet.Args()

	// --- Handle subcommands ---
	cmd := subcommands["table"] // Default to "table"
	if len(args) > 0 {
		first := args[0]
		if subcommand, ok := subcommands[first]; ok {
			cmd = subcommand
			args = args[1:]
		}
	}

	cmd.flagSet.Parse(args)
	subargs := cmd.flagSet.Args()

	if err := cmd.run(subargs); err != nil {
		log.Fatal(err)
	}
}

// -v- Subcommand definitions --------------------------------------------------

func tableCmd() command {
	flagSet := flag.NewFlagSet("git-who table", flag.ExitOnError)

	useCsv := flagSet.Bool("csv", false, "Output as csv")

	flagSet.Usage = func() {
		fmt.Println("Usage: git-who table [--csv] [revision...] [[--] path]")
		fmt.Println("Print out a table summarizing authorship")
		flagSet.PrintDefaults()
	}

	return command{
		flagSet: flagSet,
		run: func(args []string) error {
			revs, path := git.ParseArgs(args)
			return table(revs, path, *useCsv)
		},
	}
}

func treeCmd() command {
	flagSet := flag.NewFlagSet("git-who tree", flag.ExitOnError)

	useLines := flagSet.Bool("l", false, "Rank authors by lines added/changed")
	useFiles := flagSet.Bool("f", false, "Rank authors by files touched")
	depth := flagSet.Int("d", 0, "Limit on tree depth")

	flagSet.Usage = func() {
		fmt.Println("Usage: git-who tree [-l|-f] [-d <depth>] [revision...] [[--] path]")
		fmt.Println("Print out a table summarizing authorship")
		flagSet.PrintDefaults()
	}

	return command{
		flagSet: flagSet,
		run: func(args []string) error {
			revs, path := git.ParseArgs(args)

			var mode tally.TallyMode
			if *useLines {
				mode = tally.LinesMode
			} else if *useFiles {
				mode = tally.FilesMode
			}

			return tree(revs, path, mode, *depth)
		},
	}
}

// -^---------------------------------------------------------------------------
