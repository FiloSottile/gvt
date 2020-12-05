package fileutils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCopypathSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no symlinks on windows y'all")
	}
	dst := mktemp(t)
	defer RemoveAll(dst)
	src := filepath.Join("_testdata", "copyfile")
	if err := Copypath(dst, src, true, false, false); err != nil {
		t.Fatalf("copypath(%s, %s): %v", dst, src, err)
	}
	res, err := os.Readlink(filepath.Join(dst, "a", "rick"))
	if err != nil {
		t.Fatal(err)
	}
	if res != "/never/going/to/give/you/up" {
		t.Fatalf("target == %s, expected /never/going/to/give/you/up", res)
	}
}

func mktemp(t *testing.T) string {
	s, err := ioutil.TempDir("", "fileutils_test")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestShouldSkip(t *testing.T) {
	_, filename, _, _ := runtime.Caller(1)
	stat, _ := os.Stat(filename)

	expectations := [][]interface{}{
		[]interface{}{"a.go", stat, false, false, false, false},     // default: go files are ok
		[]interface{}{"a_test.go", stat, false, false, false, true}, // default: test files are not ok
		[]interface{}{"a.mak", stat, false, false, false, true},     // default: makefiles are  not ok
		[]interface{}{"a.rand", stat, false, false, false, true},    // default: all files are  not ok

		[]interface{}{"a_test.go", stat, true, false, false, false}, // Allow test files
		[]interface{}{"a.mak", stat, false, false, true, false},     // Allow makefiles
		[]interface{}{"a.rand", stat, false, true, false, false},    // Allow all files
	}

	for _, e := range expectations {
		result := ShouldSkip(e[0].(string), e[1].(os.FileInfo), e[2].(bool), e[3].(bool), e[4].(bool))

		if result != e[5].(bool) {
			t.Fatalf("wrong result expected(%v) got(%v)", e[5].(bool), result)
		}
	}
}
