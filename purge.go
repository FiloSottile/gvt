package main

import (
	"path/filepath"
	errors "github.com/pkg/errors"
	"strings"

	fileutils "github.com/FiloSottile/gvt/fileutils"
	gbvendor "github.com/FiloSottile/gvt/gbvendor"
	"path"
)

var cmdPurge = &Command{
	Name:      "purge",
	UsageLine: "purge",
	Short:     "purges all unreferenced dependencies",
	Long: `gb vendor purge will remove all unreferenced dependencies

`,
	Run: func(args []string) error {
		m, err := gbvendor.ReadManifest(manifestFile)
		if err != nil {
			return errors.Wrap(err, "could not load manifest")
		}
		root := path.Dir(vendorDir)
		imports, err := gbvendor.ParseImports(root, root, "", true, true)
		if err != nil {
			return errors.Wrap(err, "import could not be parsed")
		}

		var hasImportWithPrefix = func(d string) bool {
			for i := range imports {
				if strings.HasPrefix(i, d) {
					return true
				}
			}
			return false
		}

		dependencies := make([]gbvendor.Dependency, len(m.Dependencies))
		copy(dependencies, m.Dependencies)

		for _, d := range dependencies {
			if !hasImportWithPrefix(d.Importpath) {
				dep, err := m.GetDependencyForImportpath(d.Importpath)
				if err != nil {
					return errors.Wrap(err, "could not get get dependency")
				}

				if err := m.RemoveDependency(dep); err != nil {
					return errors.Wrap(err, "dependency could not be removed")
				}
				if err := fileutils.RemoveAll(filepath.Join(root, "vendor", "src", filepath.FromSlash(d.Importpath))); err != nil {
					// TODO(dfc) need to apply vendor.cleanpath here to remove intermediate directories.
					return errors.Wrap(err, "dependency could not be deleted")
				}
			}
		}

		return gbvendor.WriteManifest(manifestFile, m)
	},
}
