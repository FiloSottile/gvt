package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/FiloSottile/gvt/fileutils"
	"github.com/FiloSottile/gvt/gbvendor"
)

var (
	updateAll bool // update all dependencies
)

func addUpdateFlags(fs *flag.FlagSet) {
	fs.BoolVar(&updateAll, "all", false, "update all dependencies")
	fs.BoolVar(&insecure, "precaire", false, "allow the use of insecure protocols")
	fs.BoolVar(&noRecurse, "no-recurse", false, "do not fetch recursively")
}

var cmdUpdate = &Command{
	Name:      "update",
	UsageLine: "update [ -all | importpath ]",
	Short:     "update a local dependency",
	Long: `update replaces the source with the latest available from the head of the fetched branch.

Updating from one copy of a dependency to another is ONLY possible when the
dependency was fetched by branch, without using -tag or -revision. It will be
updated to the HEAD of that branch, switching branches is not supported.

To update across branches, or from one tag/revision to another, you must first
use delete to remove the dependency, then fetch [ -tag | -revision | -branch ]
to replace it.

Flags:
	-all
		update all dependencies in the manifest.
  -no-recurse
    skip checking for and fetching new transitive dependencies
	-precaire
		allow the use of insecure protocols.

`,
	Run: func(args []string) error {
		if len(args) != 1 && !updateAll {
			return fmt.Errorf("update: import path or -all flag is missing")
		} else if len(args) == 1 && updateAll {
			return fmt.Errorf("update: you cannot specify path and -all flag at once")
		}

		m, err := vendor.ReadManifest(manifestFile)
		if err != nil {
			return fmt.Errorf("could not load manifest: %v", err)
		}

		var dependencies []vendor.Dependency
		if updateAll {
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

			repo, err := vendor.NewRemoteRepo(d.Repository, d.VCS, insecure)
			if err != nil {
				return fmt.Errorf("could not determine repository for import %q", d.Importpath)
			}

			wc, err := GlobalDownloader.Get(repo, d.Branch, "", "")
			if err != nil {
				return err
			}

			rev, err := wc.Revision()
			if err != nil {
				return err
			}

			branch, err := wc.Branch()
			if err != nil {
				return err
			}

			dep := vendor.Dependency{
				Importpath: d.Importpath,
				Repository: repo.URL(),
				VCS:        repo.Type(),
				Revision:   rev,
				Branch:     branch,
				Path:       d.Path,
				NoTests:    d.NoTests,
				AllFiles:   d.AllFiles,
			}

			if err := fileutils.RemoveAll(filepath.Join(vendorDir, filepath.FromSlash(d.Importpath))); err != nil {
				// TODO(dfc) need to apply vendor.cleanpath here to remove intermediate directories.
				return fmt.Errorf("dependency could not be deleted: %v", err)
			}

			dst := filepath.Join(vendorDir, filepath.FromSlash(dep.Importpath))
			src := filepath.Join(wc.Dir(), dep.Path)

			if err := fileutils.Copypath(dst, src, !d.NoTests, d.AllFiles); err != nil {
				return err
			}

			if err := fileutils.CopyLicense(dst, wc.Dir()); err != nil {
				return err
			}

			if err := m.AddDependency(dep); err != nil {
				return err
			}

			if err := vendor.WriteManifest(manifestFile, m); err != nil {
				return err
			}

			if !noRecurse {
				// Look for dependencies in src, not going past wc.Dir() when looking for /vendor/,
				// knowing that wc.Dir() corresponds to rootRepoPath
				if !strings.HasSuffix(dep.Importpath, dep.Path) {
					return fmt.Errorf("unable to derive the root repo import path")
				}
				rootRepoPath := strings.TrimRight(strings.TrimSuffix(dep.Importpath, dep.Path), "/")
				deps, err := vendor.ParseImports(src, wc.Dir(), rootRepoPath, !d.NoTests, d.AllFiles)
				if err != nil {
					return fmt.Errorf("failed to parse imports: %s", err)
				}

				for d := range deps {
					if strings.Index(d, ".") == -1 { // TODO: replace this silly heuristic
						continue
					}
					if !m.HasImportpath(d) {
						if err := fetch(d); err != nil {
							return err
						}
					}
				}
			}
		}

		return nil
	},
	AddFlags: addUpdateFlags,
}
