package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/FiloSottile/gvt/gbvendor"
)

var (
	rbInsecure bool // Allow the use of insecure protocols
)

func addRebuildFlags(fs *flag.FlagSet) {
	fs.BoolVar(&rbInsecure, "precaire", false, "allow the use of insecure protocols")
}

var cmdRebuild = &Command{
	Name:      "rebuild",
	UsageLine: "rebuild",
	Short:     "rebuild dependencies from manifest",
	Long: `rebuild fetches the dependencies listed in the manifest.

It's meant for workflows that don't include checking in to VCS the vendored
source, for example if .gitignore includes lines like

    vendor/**
    !vendor/manifest

Note that such a setup requires "gvt rebuild" to build the source, relies on
the availability of the dependencies repositories and breaks "go get".

Flags:
	-precaire
		allow the use of insecure protocols.
`,
	Run: func(args []string) error {
		switch len(args) {
		case 0:
			return rebuild()
		default:
			return fmt.Errorf("rebuild takes no arguments")
		}
	},
	AddFlags: addRebuildFlags,
}

func rebuild() error {
	m, err := vendor.ReadManifest(manifestFile())
	if err != nil {
		return fmt.Errorf("could not load manifest: %v", err)
	}

	for _, dep := range m.Dependencies {
		dst := filepath.Join(vendorDir(), dep.Importpath)
		if _, err := os.Stat(dst); err == nil {
			if err := vendor.RemoveAll(dst); err != nil {
				// TODO need to apply vendor.cleanpath here too
				return fmt.Errorf("dependency could not be deleted: %v", err)
			}
		}

		log.Printf("fetching %s", dep.Importpath)

		repo, _, err := vendor.DeduceRemoteRepo(dep.Importpath, rbInsecure)
		if err != nil {
			return err
		}

		wc, err := repo.Checkout("", "", dep.Revision)
		if err != nil {
			return err
		}

		src := filepath.Join(wc.Dir(), dep.Path)
		if err := vendor.Copypath(dst, src); err != nil {
			return err
		}

		if err := wc.Destroy(); err != nil {
			return err
		}
	}

	return nil
}
