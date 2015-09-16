package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
)

var fs = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

func init() {
	fs.Usage = func() {
		printUsage(os.Stderr)
		os.Exit(2)
	}
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
	cmdRebuild,
	cmdUpdate,
	cmdList,
	cmdDelete,
}

func main() {
	args := os.Args[1:]

	switch {
	case len(args) < 1, args[0] == "-h", args[0] == "-help":
		fs.Usage()
		os.Exit(1)
	case args[0] == "help":
		help(args[1:])
		return
	}

	for _, command := range commands {
		if command.Name == args[0] {

			// add extra flags if necessary
			if command.AddFlags != nil {
				command.AddFlags(fs)
			}

			if err := fs.Parse(args[1:]); err != nil {
				log.Fatalf("could not parse flags: %v", err)
			}
			args = fs.Args() // reset args to the leftovers from fs.Parse

			if err := command.Run(args); err != nil {
				log.Fatalf("command %q failed: %v", command.Name, err)
			}
			return
		}
	}
	log.Fatalf("unknown command %q ", args[0])
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
