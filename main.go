package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var fs = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

func init() {
	fs.Usage = func() {}
}

type Command struct {
	Name      string
	UsageLine string
	Short     string
	Long      string
	Run       func(args []string) error
	AddFlags  func(fs *flag.FlagSet)
}

var commands = []*Command{
	cmdFetch,
	cmdRestore,
	cmdUpdate,
	cmdList,
	cmdDelete,
}

func main() {
	args := os.Args[1:]

	switch {
	case len(args) < 1, args[0] == "-h", args[0] == "-help":
		printUsage(os.Stdout)
		os.Exit(0)
	case args[0] == "help":
		help(args[1:])
		return
	case args[0] == "rebuild":
		// rebuild was renamed restore, alias for backwards compatibility
		args[0] = "restore"
	}

	for _, command := range commands {
		if command.Name == args[0] {

			// add extra flags if necessary
			if command.AddFlags != nil {
				command.AddFlags(fs)
			}

			if err := fs.Parse(args[1:]); err != nil {
				if err == flag.ErrHelp {
					help(args[:1])
					os.Exit(0)
				}
				fmt.Fprint(os.Stderr, "\n")
				help(args[:1])
				os.Exit(3)
			}

			if err := command.Run(fs.Args()); err != nil {
				log.Fatalf("command %q failed: %v", command.Name, err)
			}
			return
		}
	}
	fmt.Fprintf(os.Stderr, "unknown command: %q\n\n", args[0])
	printUsage(os.Stderr)
	os.Exit(3)
}

const manifestfile = "manifest"

func vendorDir() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(wd, "vendor")
}

func manifestFile() string {
	return filepath.Join(vendorDir(), manifestfile)
}
