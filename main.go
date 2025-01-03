package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/sinclairtarget/git-who/internal/flagutils"
	"github.com/sinclairtarget/git-who/internal/git"
	"github.com/sinclairtarget/git-who/internal/tally"
)

const version = "0.1"

var progStart time.Time

type command struct {
	flagSet  *flag.FlagSet
	run      func(args []string) error
	isHidden bool // Hide from usage
}

// Main examines the args and delegates to the specified subcommand.
//
// If no subcommand was specified, we default to the "table" subcommand.
func main() {
	subcommands := map[string]command{ // Available subcommands
		"dump":  dumpCmd(),
		"parse": parseCmd(),
		"table": tableCmd(),
		"tree":  treeCmd(),
		"hist":  histCmd(),
	}

	// --- Handle top-level flags ---
	mainFlagSet := flag.NewFlagSet("git-who", flag.ExitOnError)

	versionFlag := mainFlagSet.Bool("version", false, "Print version and exit")
	verboseFlag := mainFlagSet.Bool("v", false, "Enables debug logging")

	mainFlagSet.Usage = func() {
		fmt.Println("Usage: git-who [options...] [subcommand]")
		fmt.Println("git-who tallies authorship")
		mainFlagSet.PrintDefaults()

		fmt.Println()
		fmt.Println("Subcommands:")

		for name, cmd := range subcommands {
			if cmd.isHidden {
				continue
			}

			fmt.Println(name)
			cmd.flagSet.PrintDefaults()
		}
	}

	// Look for the index of the first arg not intended as a top-level flag.
	// We handle this manually so that specifying the default subcommand is
	// optional even when providing subcommand flags.
	subcmdIndex := 1
loop:
	for subcmdIndex < len(os.Args) {
		switch os.Args[subcmdIndex] {
		case "-version", "--version", "-v", "--v", "-h", "--help":
			subcmdIndex += 1
		default:
			break loop
		}
	}

	mainFlagSet.Parse(os.Args[1:subcmdIndex])

	if *versionFlag {
		fmt.Printf("%s\n", version)
		return
	}

	if *verboseFlag {
		configureLogging(slog.LevelDebug)
		logger().Debug("Log level set to DEBUG")
	} else {
		configureLogging(slog.LevelInfo)
	}

	args := os.Args[subcmdIndex:]

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

	progStart = time.Now()
	if err := cmd.run(subargs); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// -v- Subcommand definitions --------------------------------------------------

func tableCmd() command {
	flagSet := flag.NewFlagSet("git-who table", flag.ExitOnError)

	useCsv := flagSet.Bool("csv", false, "Output as csv")
	showEmail := flagSet.Bool("e", false, "Show email address of each author")
	linesMode := flagSet.Bool("l", false, "Sort by lines added + removed")
	filesMode := flagSet.Bool("f", false, "Sort by files changed")
	lastModifiedMode := flagSet.Bool("m", false, "Sort by last modified")
	limit := flagSet.Int("n", 10, "Limit rows in table (set to 0 for no limit)")

	filterFlags := addFilterFlags(flagSet)

	flagSet.Usage = func() {
		fmt.Println(strings.TrimSpace(`
Usage: git-who table [--csv] [-e] [-n <n>] [-l|-f|-m] [filter opts...] [revision...] [[--] path]
		`))
		fmt.Println("Print out a table summarizing authorship")
		flagSet.PrintDefaults()
	}

	return command{
		flagSet: flagSet,
		run: func(args []string) error {
			mode := tally.CommitMode

			if !isOnlyOne(*linesMode, *filesMode, *lastModifiedMode) {
				return errors.New("all sort flags are mutually exclusive")
			}

			if *linesMode {
				mode = tally.LinesMode
			} else if *filesMode {
				mode = tally.FilesMode
			} else if *lastModifiedMode {
				mode = tally.LastModifiedMode
			}

			if *limit < 0 {
				return errors.New("-n flag must be a positive integer")
			}

			revs, paths, err := git.ParseArgs(args)
			if err != nil {
				return fmt.Errorf("could not parse args: %w", err)
			}
			return table(
				revs,
				paths,
				mode,
				*useCsv,
				*showEmail,
				*limit,
				*filterFlags.since,
				filterFlags.authors,
				filterFlags.nauthors,
			)
		},
	}
}

func treeCmd() command {
	flagSet := flag.NewFlagSet("git-who tree", flag.ExitOnError)

	showEmail := flagSet.Bool("e", false, "Show email address of each author")
	useLines := flagSet.Bool("l", false, "Rank authors by lines added/changed")
	useFiles := flagSet.Bool("f", false, "Rank authors by files touched")
	useLastModified := flagSet.Bool(
		"m",
		false,
		"Rank authors by last commit time",
	)
	depth := flagSet.Int("d", 0, "Limit on tree depth")

	filterFlags := addFilterFlags(flagSet)

	flagSet.Usage = func() {
		fmt.Println(strings.TrimSpace(`
Usage: git-who tree [-e] [-l|-f|-m] [-d <depth>] [filter opts...] [revision...] [[--] path]
		`))
		fmt.Println("Print out a tree summarizing authorship")
		flagSet.PrintDefaults()
	}

	return command{
		flagSet: flagSet,
		run: func(args []string) error {
			revs, paths, err := git.ParseArgs(args)
			if err != nil {
				return fmt.Errorf("could not parse args: %w", err)
			}

			if !isOnlyOne(*useLines, *useFiles, *useLastModified) {
				return errors.New("all ranking flags are mutually exclusive")
			}

			mode := tally.CommitMode
			if *useLines {
				mode = tally.LinesMode
			} else if *useFiles {
				mode = tally.FilesMode
			} else if *useLastModified {
				mode = tally.LastModifiedMode
			}

			return tree(
				revs,
				paths,
				mode,
				*depth,
				*showEmail,
				*filterFlags.since,
				filterFlags.authors,
				filterFlags.nauthors,
			)
		},
	}
}

func histCmd() command {
	flagSet := flag.NewFlagSet("git-who hist", flag.ExitOnError)

	useLines := flagSet.Bool("l", false, "Rank authors by lines added/changed")
	useFiles := flagSet.Bool("f", false, "Rank authors by files touched")
	showEmail := flagSet.Bool("e", false, "Show email address of each author")

	filterFlags := addFilterFlags(flagSet)

	flagSet.Usage = func() {
		fmt.Println(strings.TrimSpace(`
Usage: git-who hist [-e] [-l|-f] [filter opts...] [revision...] [[--] path]
		`))
		fmt.Println("Print out a timeline summarizing authorship")
		flagSet.PrintDefaults()
	}

	return command{
		flagSet: flagSet,
		run: func(args []string) error {
			revs, paths, err := git.ParseArgs(args)
			if err != nil {
				return fmt.Errorf("could not parse args: %w", err)
			}

			if !isOnlyOne(*useLines, *useFiles) {
				return errors.New("all ranking flags are mutually exclusive")
			}

			mode := tally.CommitMode
			if *useLines {
				mode = tally.LinesMode
			} else if *useFiles {
				mode = tally.FilesMode
			}

			return hist(
				revs,
				paths,
				mode,
				*showEmail,
				*filterFlags.since,
				filterFlags.authors,
				filterFlags.nauthors,
			)
		},
	}
}

func dumpCmd() command {
	flagSet := flag.NewFlagSet("git-who dump", flag.ExitOnError)

	short := flagSet.Bool("s", false, "Use short log")

	filterFlags := addFilterFlags(flagSet)

	return command{
		flagSet: flagSet,
		run: func(args []string) error {
			revs, paths, err := git.ParseArgs(args)
			if err != nil {
				return fmt.Errorf("could not parse args: %w", err)
			}
			return dump(
				revs,
				paths,
				*short,
				*filterFlags.since,
				filterFlags.authors,
				filterFlags.nauthors,
			)
		},
		isHidden: true,
	}
}

func parseCmd() command {
	flagSet := flag.NewFlagSet("git-who parse", flag.ExitOnError)

	filterFlags := addFilterFlags(flagSet)

	return command{
		flagSet: flagSet,
		run: func(args []string) error {
			revs, paths, err := git.ParseArgs(args)
			if err != nil {
				return fmt.Errorf("could not parse args: %w", err)
			}
			return parse(
				revs,
				paths,
				*filterFlags.since,
				filterFlags.authors,
				filterFlags.nauthors,
			)
		},
		isHidden: true,
	}
}

// -^---------------------------------------------------------------------------

func configureLogging(level slog.Level) {
	handler := slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{
			Level: level,
		},
	)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// Used to check mutual exclusion.
func isOnlyOne(flags ...bool) bool {
	var foundOne bool
	for _, f := range flags {
		if f {
			if foundOne {
				return false
			}

			foundOne = true
		}
	}

	return true
}

type filterFlags struct {
	since    *string
	authors  flagutils.SliceFlag
	nauthors flagutils.SliceFlag
}

func addFilterFlags(set *flag.FlagSet) *filterFlags {
	flags := filterFlags{
		since: set.String("since", "", strings.TrimSpace(`
Only count commits after the given date. See git-commit(1) for valid formats
		`)),
	}

	set.Var(&flags.authors, "author", strings.TrimSpace(`
Only count commits by these authors. Can be specified multiple times
	`))

	set.Var(&flags.nauthors, "nauthor", strings.TrimSpace(`
Exclude commits by these authors. Can be specified multiple times
	`))

	return &flags
}
