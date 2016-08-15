package abicheck

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"
)

// TestParse tests the results from the parser against an expected golden master
func TestParse(t *testing.T) {
	// Create strvcs and fill it with test data
	vcs := StrVCS{}

	rev1, err := ioutil.ReadFile("testdata/before.go")
	if err != nil {
		t.Fatal("cannot load test data for rev1:", err)
	}
	vcs.SetFile("rev1", "abitest.go", rev1)

	rev2, err := ioutil.ReadFile("testdata/after.go")
	if err != nil {
		t.Fatal("cannot load test data for rev2:", err)
	}
	vcs.SetFile("rev2", "abitest.go", rev2)

	// Run checks
	c := New(SetVCS(vcs))

	changes, err := c.Check("", "rev1", "rev2")
	if err != nil {
		t.Fatal(err)
	}

	lineNum := regexp.MustCompile(":[0-9]+:")
	// Save results to buffer for comparison with gold master
	var buf bytes.Buffer
	for _, change := range changes {
		fmt.Fprint(&buf, lineNum.ReplaceAllString(change.String(), ":-:"))
	}

	// Overwrite the gold master with go test -args update
	if len(os.Args) > 1 && os.Args[1] == "update" {
		err = ioutil.WriteFile("testdata/exp.txt", buf.Bytes(), os.FileMode(0644))
		if err != nil {
			t.Fatal("could not write exp data:", err)
		}
	}

	// Load gold master
	exp, err := ioutil.ReadFile("testdata/exp.txt")
	if err != nil {
		t.Fatal("cannot load expect data:", err)
	}

	// Compare results gold master
	if !reflect.DeepEqual(exp, buf.Bytes()) {
		t.Errorf("results did not match testdata/exp.txt")
		t.Errorf("run 'go test -args update && git diff testdata/exp.txt' to review")
	}
}

// TestPaths tests an example project with various paths and verifies
// it finds a certain number of changes ensuring recursive is working
// as expected
func TestPaths(t *testing.T) {

	// Make the test data dirs
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	testdataDir := filepath.Join(wd, "testdata")

	cmd := exec.Command("./make.sh")
	cmd.Dir = testdataDir
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		wd   string // working dir relative to testdata/gopath/src
		path string // import path
		exp  int    // expected number of changes
	}{
		{"", "example.com/lib", 1},
		{"", "example.com/lib/...", 2},    // recursive
		{"", "example.com/lib/b/...", 1},  // empty directory
		{"example.com/lib", "", 1},        // working directory
		{"example.com/lib", "./...", 2},   // working directory recursive
		{"example.com/lib/b", "./...", 1}, // empty working directory
	}

	oldPath := os.Getenv("GOPATH")
	defer func() {
		os.Setenv("GOPATH", oldPath)
	}()
	os.Setenv("GOPATH", filepath.Join(testdataDir, "gopath"))

	for _, test := range tests {
		t.Logf("Test: %#v", test)
		err := os.Chdir(filepath.Join(testdataDir, "gopath", "src", test.wd))
		if err != nil {
			t.Errorf("Cannot chdir: %s", err)
		}

		git, err := NewGit()
		if err != nil {
			t.Errorf("Cannot get new git: %s", err)
		}
		checker := New(SetVCS(git))

		changes, err := checker.Check(test.path, "HEAD~1", "HEAD")
		if err != nil {
			t.Errorf("Check error: %s", err)
		}

		if test.exp != len(changes) {
			t.Errorf("exp %d got %d", test.exp, len(changes))
		}
	}
}
