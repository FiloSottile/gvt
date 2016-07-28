// package fileutils provides utililty methods to copy and move files and directories.
package fileutils

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// https://golang.org/cmd/go/#hdr-File_types
var goFileTypes = []string{
	".go",
	".c", ".h",
	".cc", ".cpp", ".cxx", ".hh", ".hpp", ".hxx",
	".m",
	".s", ".S",
	".swig", ".swigcxx",
	".syso",
}

var licenseFiles = []string{
	"LICENSE", "LICENCE", "UNLICENSE", "COPYING", "COPYRIGHT",
}

func ShouldSkip(path string, info os.FileInfo, tests, all bool) bool {
	name := filepath.Base(path)

	relevantFile := false
	for _, ext := range goFileTypes {
		if strings.HasSuffix(name, ext) {
			relevantFile = true
			break
		}
	}

	testdata := false
	for _, n := range strings.Split(filepath.Dir(path), string(filepath.Separator)) {
		if n == "testdata" || n == "_testdata" {
			testdata = true
		}
	}

	skip := false
	switch {
	case all && !(name == ".git" && info.IsDir()) && name != ".bzr" && name != ".hg":
		skip = false

	// Include all files in a testdata folder
	case tests && testdata:
		skip = false

	// https://golang.org/cmd/go/#hdr-Description_of_package_lists
	case strings.HasPrefix(name, "."):
		skip = true
	case strings.HasPrefix(name, "_") && name != "_testdata":
		skip = true

	case !tests && name == "_testdata" && info.IsDir():
		skip = true
	case !tests && name == "testdata" && info.IsDir():
		skip = true
	case !tests && strings.HasSuffix(name, "_test.go") && !info.IsDir():
		skip = true

	case !relevantFile && !info.IsDir():
		skip = true
	}

	return skip
}

// Copypath copies the contents of src to dst, excluding any file that is not
// relevant to the Go compiler.
func Copypath(dst string, src string, tests, all bool) error {
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		skip := ShouldSkip(path, info, tests, all)

		if skip {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return nil
		}

		dst := filepath.Join(dst, path[len(src):])

		if info.Mode()&os.ModeSymlink != 0 {
			return Copylink(dst, path)
		}

		return Copyfile(dst, path)
	})
	if err != nil {
		// if there was an error during copying, remove the partial copy.
		RemoveAll(dst)
	}
	return err
}

func Copyfile(dst, src string) error {
	err := mkdir(filepath.Dir(dst))
	if err != nil {
		return fmt.Errorf("copyfile: mkdirall: %v", err)
	}
	r, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("copyfile: open(%q): %v", src, err)
	}
	defer r.Close()
	w, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("copyfile: create(%q): %v", dst, err)
	}
	defer w.Close()
	_, err = io.Copy(w, r)
	return err
}

func Copylink(dst, src string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return fmt.Errorf("copylink: readlink: %v", err)
	}
	if err := mkdir(filepath.Dir(dst)); err != nil {
		return fmt.Errorf("copylink: mkdirall: %v", err)
	}
	if err := os.Symlink(target, dst); err != nil {
		return fmt.Errorf("copylink: symlink: %v", err)
	}
	return nil
}

// RemoveAll removes path and any children it contains. Unlike os.RemoveAll it
// deletes read only files on Windows.
func RemoveAll(path string) error {
	if runtime.GOOS == "windows" {
		// Simple case: if Remove works, we're done.
		err := os.Remove(path)
		if err == nil || os.IsNotExist(err) {
			return nil
		}
		// make sure all files are writable so we can delete them
		filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				// walk gave us some error, give it back.
				return err
			}
			mode := info.Mode()
			if mode|0200 == mode {
				return nil
			}
			return os.Chmod(path, mode|0200)
		})
	}
	return os.RemoveAll(path)
}

// CopyLicense copies the license file from folder src to folder dst.
func CopyLicense(dst, src string) error {
	files, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		for _, candidate := range licenseFiles {
			if strings.ToLower(candidate) == strings.TrimSuffix(
				strings.TrimSuffix(strings.ToLower(f.Name()), ".md"), ".txt") {
				if err := Copyfile(filepath.Join(dst, f.Name()),
					filepath.Join(src, f.Name())); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func mkdir(path string) error {
	return os.MkdirAll(path, 0755)
}
