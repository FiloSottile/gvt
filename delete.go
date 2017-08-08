package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/FiloSottile/gvt/fileutils"

	"github.com/FiloSottile/gvt/gbvendor"
)

var (
	deleteAll bool // delete all dependencies
	recurse   bool // remove dependencies recursively
)

func addDeleteFlags(fs *flag.FlagSet) {
	fs.BoolVar(&deleteAll, "all", false, "delete all dependencies")
	fs.BoolVar(&recurse, "recurse", false, "remove dependencies recursively")
}

var cmdDelete = &Command{
	Name:      "delete",
	UsageLine: "delete [-all] [-recurse] importpath",
	Short:     "delete a local dependency",
	Long: `delete removes a dependency from the vendor directory and the manifest

Flags:
	-all
		remove all dependencies
	-recurse
		remove dependencies recursively

`,
	Run: func(args []string) error {
		if len(args) != 1 && !deleteAll {
			return fmt.Errorf("delete: import path or --all flag is missing")
		} else if len(args) == 1 && deleteAll {
			return fmt.Errorf("delete: you cannot specify path and --all flag at once")
		}

		m, err := vendor.ReadManifest(manifestFile)
		if err != nil {
			return fmt.Errorf("could not load manifest: %v", err)
		}

		p := args[0]
		if p[len(p)-1] == '/' {
			p = p[:len(p)-1]
		}

		var dependencies []vendor.Dependency
		if deleteAll {
			dependencies = make([]vendor.Dependency, len(m.Dependencies))
			copy(dependencies, m.Dependencies)
		} else if recurse {
			deps := m.GetSubpackages(p)
			if len(deps) == 0 {
				return fmt.Errorf("no dependencies found recursively")
			}
			dependencies = append(dependencies, deps...)
		} else {
			dependency, err := m.GetDependencyForImportpath(p)
			if err != nil {
				return fmt.Errorf("could not get dependency: %v", err)
			}
			if p != dependency.Importpath {
				return fmt.Errorf("a parent of the specified dependency is vendored, remove that instead: %v",
					dependency.Importpath)
			}
			dependencies = append(dependencies, dependency)
		}

		for _, d := range dependencies {
			path := d.Importpath

			if err := m.RemoveDependency(d); err != nil {
				return fmt.Errorf("dependency could not be deleted: %v", err)
			}

			if err := fileutils.RemoveAll(filepath.Join(vendorDir, filepath.FromSlash(path))); err != nil {
				// TODO(dfc) need to apply vendor.cleanpath here to remove indermediate directories.
				return fmt.Errorf("dependency could not be deleted: %v", err)
			}
		}
		return vendor.WriteManifest(manifestFile, m)
	},
	AddFlags: addDeleteFlags,
}
