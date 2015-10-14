package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"runtime"
	"sort"

	"go/build"

	"github.com/FiloSottile/gvt/gbvendor"
)

var (
	branch    string
	revision  string // revision (commit)
	tag       string
	noRecurse bool
	insecure  bool // Allow the use of insecure protocols

	recurse bool // should we fetch recursively
)

func addFetchFlags(fs *flag.FlagSet) {
	fs.StringVar(&branch, "branch", "", "branch of the package")
	fs.StringVar(&revision, "revision", "", "revision of the package")
	fs.StringVar(&tag, "tag", "", "tag of the package")
	fs.BoolVar(&noRecurse, "no-recurse", false, "do not fetch recursively")
	fs.BoolVar(&insecure, "precaire", false, "allow the use of insecure protocols")
}

var cmdFetch = &Command{
	Name:      "fetch",
	UsageLine: "fetch [-branch branch | -revision rev | -tag tag] [-precaire] [-no-recurse] importpath",
	Short:     "fetch a remote dependency",
	Long: `fetch vendors an upstream import path.

The import path may include a url scheme. This may be useful when fetching dependencies
from private repositories that cannot be probed.

Flags:
	-branch branch
		fetch from the name branch. If not supplied the default upstream
		branch will be used.
	-no-recurse
		do not fetch recursively.
	-tag tag
		fetch the specified tag. If not supplied the default upstream
		branch will be used.
	-revision rev
		fetch the specific revision from the branch (if supplied). If no
		revision supplied, the latest available will be supplied.
	-precaire
		allow the use of insecure protocols.

`,
	Run: func(args []string) error {
		switch len(args) {
		case 0:
			return fmt.Errorf("fetch: import path missing")
		case 1:
			path := args[0]
			recurse = !noRecurse
			return fetch(path, recurse)
		default:
			return fmt.Errorf("more than one import path supplied")
		}
	},
	AddFlags: addFetchFlags,
}

func fetch(path string, recurse bool) error {
	m, err := vendor.ReadManifest(manifestFile())
	if err != nil {
		return fmt.Errorf("could not load manifest: %v", err)
	}

	repo, extra, err := vendor.DeduceRemoteRepo(path, insecure)
	if err != nil {
		return err
	}

	// strip of any scheme portion from the path, it is already
	// encoded in the repo.
	path = stripscheme(path)

	if m.HasImportpath(path) {
		return fmt.Errorf("%s is already vendored", path)
	}

	wc, err := repo.Checkout(branch, tag, revision)

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
		Importpath: path,
		Repository: repo.URL(),
		Revision:   rev,
		Branch:     branch,
		Path:       extra,
	}

	if err := m.AddDependency(dep); err != nil {
		return err
	}

	dst := filepath.Join(vendorDir(), dep.Importpath)
	src := filepath.Join(wc.Dir(), dep.Path)

	if err := vendor.Copypath(dst, src); err != nil {
		return err
	}

	if err := vendor.WriteManifest(manifestFile(), m); err != nil {
		return err
	}

	if err := wc.Destroy(); err != nil {
		return err
	}

	if !recurse {
		return nil
	}

	// if we are recursing, overwrite branch, tag and revision
	// values so recursive fetching checks out from HEAD.
	branch = ""
	tag = ""
	revision = ""

	for done := false; !done; {

		paths := []struct {
			Root, Prefix string
		}{
			{filepath.Join(runtime.GOROOT(), "src"), ""},
		}
		m, err := vendor.ReadManifest(manifestFile())
		if err != nil {
			return err
		}
		for _, d := range m.Dependencies {
			paths = append(paths, struct{ Root, Prefix string }{filepath.Join(vendorDir(), filepath.FromSlash(d.Importpath)), filepath.FromSlash(d.Importpath)})
		}

		dsm, err := vendor.LoadPaths(paths...)
		if err != nil {
			return err
		}

		is, ok := dsm[filepath.Join(vendorDir(), path)]
		if !ok {
			return fmt.Errorf("unable to locate depset for %q", path)
		}

		missing := findMissing(pkgs(is.Pkgs), dsm)
		switch len(missing) {
		case 0:
			done = true
		default:

			// sort keys in ascending order, so the shortest missing import path
			// with be fetched first.
			keys := keys(missing)
			sort.Strings(keys)
			pkg := keys[0]
			log.Printf("fetching recursive dependency %s", pkg)
			if err := fetch(pkg, false); err != nil {
				return err
			}
		}
	}

	return nil
}

func keys(m map[string]bool) []string {
	var s []string
	for k := range m {
		s = append(s, k)
	}
	return s
}

func pkgs(m map[string]*vendor.Pkg) []*vendor.Pkg {
	var p []*vendor.Pkg
	for _, v := range m {
		p = append(p, v)
	}
	return p
}

func findMissing(pkgs []*vendor.Pkg, dsm map[string]*vendor.Depset) map[string]bool {
	missing := make(map[string]bool)
	imports := make(map[string]*vendor.Pkg)
	for _, s := range dsm {
		for _, p := range s.Pkgs {
			imports[p.ImportPath] = p
		}
	}

	// make fake C package for cgo
	imports["C"] = &vendor.Pkg{
		Depset: nil, // probably a bad idea
		Package: &build.Package{
			Name: "C",
		},
	}
	stk := make(map[string]bool)
	push := func(v string) {
		if stk[v] {
			panic(fmt.Sprintln("import loop:", v, stk))
		}
		stk[v] = true
	}
	pop := func(v string) {
		if !stk[v] {
			panic(fmt.Sprintln("impossible pop:", v, stk))
		}
		delete(stk, v)
	}

	// checked records import paths who's dependencies are all present
	checked := make(map[string]bool)

	var fn func(string)
	fn = func(importpath string) {
		p, ok := imports[importpath]
		if !ok {
			missing[importpath] = true
			return
		}

		// have we already walked this arm, if so, skip it
		if checked[importpath] {
			return
		}

		sz := len(missing)
		push(importpath)
		for _, i := range p.Imports {
			if i == importpath {
				continue
			}
			fn(i)
		}

		// if the size of the missing map has not changed
		// this entire subtree is complete, mark it as such
		if len(missing) == sz {
			checked[importpath] = true
		}
		pop(importpath)
	}
	for _, pkg := range pkgs {
		fn(pkg.ImportPath)
	}
	return missing
}

// stripscheme removes any scheme components from url like paths.
func stripscheme(path string) string {
	u, err := url.Parse(path)
	if err != nil {
		panic(err)
	}
	return u.Host + u.Path
}
