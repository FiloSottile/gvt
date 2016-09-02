package vendor

import (
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/FiloSottile/gvt/fileutils"
)

// ParseImports parses Go packages from a specific root returning a set of import paths.
// vendorRoot is how deep to go looking for vendor folders, usually the repo root.
// vendorPrefix is the vendorRoot import path.
func ParseImports(root, vendorRoot, vendorPrefix string, tests, all bool) (map[string]bool, error) {
	pkgs := make(map[string]bool)

	var walkFn = func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fileutils.ShouldSkip(p, info, tests, all) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() || filepath.Ext(p) != ".go" {
			return nil
		}

		fs := token.NewFileSet()
		f, err := parser.ParseFile(fs, p, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}

		for _, s := range f.Imports {
			pkg := strings.Replace(s.Path.Value, "\"", "", -1)
			if strings.HasPrefix(pkg, "./") {
				middle, err := filepath.Rel(vendorRoot, filepath.Dir(p))
				if err != nil {
					panic(err)
				}
				pkg = path.Join(vendorPrefix, middle, pkg)
			}
			if vp := findVendor(vendorRoot, filepath.Dir(p), pkg); vp != "" {
				pkg = path.Join(vendorPrefix, vp)
			}
			pkgs[pkg] = true
		}
		return nil
	}

	err := filepath.Walk(root, walkFn)
	return pkgs, err
}

// findVendor looks for pkgName in a vendor folder at start/vendor or deeper, stopping
// at root. start is expected to match or be a subfolder of root.
//
// It returns the path to pkgName inside the vendor folder, relative to root.
func findVendor(root, start, pkgName string) string {
	if !strings.HasPrefix(start, root) {
		log.Fatalln("Assertion failed:", root, "prefix of", start)
	}

	levels := strings.Split(strings.TrimPrefix(start, root), string(filepath.Separator))
	for {
		candidate := filepath.Join(append(append([]string{root}, levels...), "vendor", pkgName)...)

		files, err := ioutil.ReadDir(candidate)
		if err != nil {
			files = nil
		}
		isPackage := false
		for _, f := range files {
			if !f.IsDir() && filepath.Ext(f.Name()) == ".go" {
				isPackage = true
				break
			}
		}

		if isPackage {
			return strings.TrimPrefix(candidate, root)
		}

		if len(levels) == 0 {
			return ""
		}
		levels = levels[:len(levels)-1]
	}
}

// FetchMetadata fetchs the remote metadata for path.
func FetchMetadata(path string, insecure bool) (rc io.ReadCloser, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("unable to determine remote metadata protocol: %s", err)
		}
	}()
	// try https first
	rc, err = fetchMetadata("https", path)
	if err == nil {
		return
	}
	// try http if supported
	if insecure {
		rc, err = fetchMetadata("http", path)
	}
	return
}

func fetchMetadata(scheme, path string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s://%s?go-get=1", scheme, path)
	switch scheme {
	case "https", "http":
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("failed to access url %q", url)
		}
		return resp.Body, nil
	default:
		return nil, fmt.Errorf("unknown remote protocol scheme: %q", scheme)
	}
}

// ParseMetadata fetchs and decodes remote metadata for path.
func ParseMetadata(path string, insecure bool) (string, string, string, error) {
	rc, err := FetchMetadata(path, insecure)
	if err != nil {
		return "", "", "", err
	}
	defer rc.Close()

	imports, err := parseMetaGoImports(rc)
	if err != nil {
		return "", "", "", err
	}
	match := -1
	for i, im := range imports {
		if !strings.HasPrefix(path, im.Prefix) {
			continue
		}
		if match != -1 {
			return "", "", "", fmt.Errorf("multiple meta tags match import path %q", path)
		}
		match = i
	}
	if match == -1 {
		return "", "", "", fmt.Errorf("go-import metadata not found")
	}
	return imports[match].Prefix, imports[match].VCS, imports[match].RepoRoot, nil
}
