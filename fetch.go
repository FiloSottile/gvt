package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/FiloSottile/gvt/fileutils"
	"github.com/FiloSottile/gvt/gbvendor"
)

var (
	branch    string
	revision  string // revision (commit)
	tag       string
	noRecurse bool
	insecure  bool // Allow the use of insecure protocols
	tests     bool
	all       bool
)

func addFetchFlags(fs *flag.FlagSet) {
	fs.StringVar(&branch, "branch", "", "branch of the package")
	fs.StringVar(&revision, "revision", "", "revision of the package")
	fs.StringVar(&tag, "tag", "", "tag of the package")
	fs.BoolVar(&noRecurse, "no-recurse", false, "do not fetch recursively")
	fs.BoolVar(&insecure, "precaire", false, "allow the use of insecure protocols")
	fs.BoolVar(&tests, "t", false, "fetch _test.go files and testdata")
	fs.BoolVar(&all, "a", false, "fetch all files and subfolders")
}

var cmdFetch = &Command{
	Name:      "fetch",
	UsageLine: "fetch [-branch branch] [-revision rev | -tag tag] [-precaire] [-no-recurse] [-t|-a] importpath",
	Short:     "fetch a remote dependency",
	Long: `fetch vendors an upstream import path.

Recursive dependencies are fetched (at their master/tip/HEAD revision), unless they
or their parent package are already present.

If a subpackage of a dependency being fetched is already present, it will be deleted.

The import path may include a url scheme. This may be useful when fetching dependencies
from private repositories that cannot be probed.

Flags:
	-t
		fetch also _test.go files and testdata.
	-a
		fetch all files and subfolders, ignoring ONLY .git, .hg and .bzr.
	-branch branch
		fetch from the named branch. Will also be used by gvt update.
		If not supplied the default upstream branch will be used.
	-no-recurse
		do not fetch recursively.
	-tag tag
		fetch the specified tag.
	-revision rev
		fetch the specific revision from the branch or repository.
		If no revision supplied, the latest available will be fetched.
	-precaire
		allow the use of insecure protocols.

`,
	Run: func(args []string) error {
		switch len(args) {
		case 0:
			return fmt.Errorf("fetch: import path missing")
		case 1:
			path := args[0]
			return fetch(path)
		default:
			return fmt.Errorf("more than one import path supplied")
		}
	},
	AddFlags: addFetchFlags,
}

var (
	fetchRoot    string   // where the current session started
	rootRepoURL  string   // the url of the repo from which the root comes from
	fetchedToday []string // packages fetched during this session
)

func fetch(path string) error {
	m, err := vendor.ReadManifest(manifestFile)
	if err != nil {
		return fmt.Errorf("could not load manifest: %v", err)
	}

	fetchRoot = stripscheme(path)
	return fetchRecursive(m, path, 0)
}

func fetchRecursive(m *vendor.Manifest, fullPath string, level int) error {
	path := stripscheme(fullPath)

	// Don't even bother the user about skipping packages we just fetched
	for _, p := range fetchedToday {
		if contains(p, path) {
			return nil
		}
	}

	// First, check if this or a parent is already vendored
	if m.HasImportpath(path) {
		if level == 0 {
			return fmt.Errorf("%s or a parent of it is already vendored", path)
		} else {
			// TODO: print a different message for packages fetched during this session
			logIndent(level, "Skipping (existing):", path)
			return nil
		}
	}

	// Next, check if we are trying to vendor from the same repository we are in
	if importPath != "" && contains(importPath, path) {
		if level == 0 {
			return fmt.Errorf("refusing to vendor a subpackage of \".\"")
		} else {
			logIndent(level, "Skipping (subpackage of \".\"):", path)
			return nil
		}
	}

	if level == 0 {
		log.Println("Fetching:", path)
	} else {
		logIndent(level, "Fetching recursive dependency:", path)
	}

	// Finally, check if we already vendored a subpackage and remove it
	for _, subp := range m.GetSubpackages(path) {
		if !contains(subp.Importpath, fetchRoot) { // ignore parents of the root
			ignore := false
			for _, d := range fetchedToday {
				if contains(d, subp.Importpath) {
					ignore = true // No need to warn the user if we just downloaded it
				}
			}
			if !ignore {
				logIndent(level, "Deleting existing subpackage to prevent overlap:", subp.Importpath)
			}
		}
		if err := m.RemoveDependency(subp); err != nil {
			return fmt.Errorf("failed to remove subpackage: %v", err)
		}
	}
	if err := fileutils.RemoveAll(filepath.Join(vendorDir, path)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing folder: %v", err)
	}

	// Find and download the repository

	repo, extra, err := GlobalDownloader.DeduceRemoteRepo(fullPath, insecure)
	if err != nil {
		return err
	}

	if level == 0 {
		rootRepoURL = repo.URL()
	}

	var wc vendor.WorkingCopy
	if repo.URL() == rootRepoURL {
		wc, err = GlobalDownloader.Get(repo, branch, tag, revision)
	} else {
		wc, err = GlobalDownloader.Get(repo, "", "", "")
	}
	if err != nil {
		return err
	}

	// Add the dependency to the manifest

	rev, err := wc.Revision()
	if err != nil {
		return err
	}

	b, err := wc.Branch()
	if err != nil {
		return err
	}

	dep := vendor.Dependency{
		Importpath: path,
		Repository: repo.URL(),
		VCS:        repo.Type(),
		Revision:   rev,
		Branch:     b,
		Path:       extra,
		NoTests:    !tests,
		AllFiles:   all,
	}

	if err := m.AddDependency(dep); err != nil {
		return err
	}

	// Copy the code to the vendor folder

	dst := filepath.Join(vendorDir, dep.Importpath)
	src := filepath.Join(wc.Dir(), dep.Path)

	if err := fileutils.Copypath(dst, src, !dep.NoTests, dep.AllFiles); err != nil {
		return err
	}

	if err := fileutils.CopyLicense(dst, wc.Dir()); err != nil {
		return err
	}

	if err := vendor.WriteManifest(manifestFile, m); err != nil {
		return err
	}

	// Recurse

	fetchedToday = append(fetchedToday, path)

	if !noRecurse {
		// Look for dependencies in src, not going past wc.Dir() when looking for /vendor/,
		// knowing that wc.Dir() corresponds to rootRepoPath
		if !strings.HasSuffix(dep.Importpath, dep.Path) {
			return fmt.Errorf("unable to derive the root repo import path")
		}
		rootRepoPath := strings.TrimRight(strings.TrimSuffix(dep.Importpath, dep.Path), "/")
		deps, err := vendor.ParseImports(src, wc.Dir(), rootRepoPath, tests, all)
		if err != nil {
			return fmt.Errorf("failed to parse imports: %s", err)
		}

		for d := range deps {
			if strings.Index(d, ".") == -1 { // TODO: replace this silly heuristic
				continue
			}
			if err := fetchRecursive(m, d, level+1); err != nil {
				if strings.HasPrefix(err.Error(), "error fetching") { // I know, ok?
					return err
				} else {
					return fmt.Errorf("error fetching %s: %s", d, err)
				}
			}
		}
	}

	return nil
}

func logIndent(level int, v ...interface{}) {
	prefix := strings.Repeat("·", level)
	v = append([]interface{}{prefix}, v...)
	log.Println(v...)
}

// stripscheme removes any scheme components from url like paths.
func stripscheme(path string) string {
	u, err := url.Parse(path)
	if err != nil {
		panic(err)
	}
	return u.Host + u.Path
}

// Package a contains package b?
func contains(a, b string) bool {
	return a == b || strings.HasPrefix(b, a+"/")
}
