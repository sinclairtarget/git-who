package main

import (
    "flag"
    "fmt"
    "log"
    "os"
)

const version = "0.1"

type command struct {
    flagSet *flag.FlagSet
    run func(args []string) error
}

// Main examines the args and delegates to the specified subcommand.
//
// If no subcommand was specified, we default to the "table" subcommand.
func main() {
    subcommands := map[string]command {
        "table": tableCmd(),
        "tree": treeCmd(),
    }

    mainFlagSet := flag.NewFlagSet("git-who", flag.ExitOnError)
    versionFlag := mainFlagSet.Bool("version", false, "Print version and exit")

    mainFlagSet.Parse(os.Args[1:])

    if *versionFlag {
        fmt.Printf("%s\n", version)
        return
    }

    args := mainFlagSet.Args()

    cmd := subcommands["table"] // Default to "table"
    if len(args) > 0 {
        firstArg := args[0]
        if subcommand, ok := subcommands[firstArg]; ok {
            cmd = subcommand
        }
    }

    if err := cmd.run(args[len(args):]); err != nil {
        log.Fatal(err)
    }
}

// -------------------- Subcommand Definitions  --------------------------------

func tableCmd() command {
    flagSet := flag.NewFlagSet("git-who table", flag.ExitOnError)

    return command{
        flagSet: flagSet,
        run: func(args []string) error {
            fmt.Println("Run tableCmd()")
            return nil
        },
    }
}

func treeCmd() command {
    flagSet := flag.NewFlagSet("git-who tree", flag.ExitOnError)

    return command{
        flagSet: flagSet,
        run: func(args []string) error {
            fmt.Println("Run treeCmd()")
            return nil
        },
    }
}
