package main

import (
	"flag"
	"fmt"
	"github.com/FiloSottile/gvt/gbvendor"
)

var (
	freezeAll bool // freeze all dependencies
)

func addFreezeFlags(fs *flag.FlagSet) {
	fs.BoolVar(&freezeAll, "all", false, "freeze all dependencies")
}

var cmdFreeze = &Command{
	Name:      "freeze",
	UsageLine: "freeze [-all] import",
	Short:     "freeze a local dependency",
	Long: `Freezes the dependency (or all if specified) to allow locking in a specific version.
Useful for 'enterprise' environments where you want everyone to have the same version of a
dependency. This allows users to quickly 'gvt fetch importpath' and later lock in the version
they are using.

Flags:
	-all
		will freeze all dependencies in the manifest, otherwise only the dependency supplied.

`,
	Run: func(args []string) error {
		if len(args) != 1 && !freezeAll {
			return fmt.Errorf("freeze: import path or --all flag is missing")
		} else if len(args) == 1 && freezeAll {
			return fmt.Errorf("freeze: you cannot specify path and --all flag at once")
		}

		m, err := vendor.ReadManifest(manifestFile())
		if err != nil {
			return fmt.Errorf("could not load manifest: %v", err)
		}

		var dependencies []vendor.Dependency
		if freezeAll {
			dependencies = make([]vendor.Dependency, len(m.Dependencies))
			copy(dependencies, m.Dependencies)
		} else {
			p := args[0]
			dependency, err := m.GetDependencyForImportpath(p)
			if err != nil {
				return fmt.Errorf("could not get dependency: %v", err)
			}
			dependencies = append(dependencies, dependency)
		}

		for _, d := range dependencies {
			err = m.RemoveDependency(d)
			if err != nil {
				return fmt.Errorf("dependency could not be deleted from manifest: %v", err)
			}

			dep := vendor.Dependency{
				Importpath: d.Importpath,
				Repository: d.Repository,
				Revision:   d.Revision,
				Branch:     "HEAD",
				Path:       d.Path,
			}

			if err := m.AddDependency(dep); err != nil {
				return err
			}

			if err := vendor.WriteManifest(manifestFile(), m); err != nil {
				return err
			}
		}

		return nil
	},
	AddFlags: addFreezeFlags,
}
