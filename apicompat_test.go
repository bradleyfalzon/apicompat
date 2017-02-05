package apicompat

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

// TestParse tests the results from the parser against an expected golden master
func TestParse(t *testing.T) {
	// Create strvcs and fill it with test data
	var vcs StrVCS

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

	changes, err := c.Check("", false, "rev1", "rev2")
	if err != nil {
		t.Fatal(err)
	}

	// Save results to buffer for comparison with gold master
	var buf bytes.Buffer
	for _, change := range changes {
		fmt.Fprint(&buf, change)
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
		t.Fatalf("error executing make.sh: %s", err)
	}

	tests := []struct {
		wd   string // working dir relative to testdata/gopath/src
		path string // import path
		exp  int    // expected number of changes
	}{
		{"", "example.com/lib", 1},
		{"", "example.com/lib/...", 2},    // recursive and ignore internal/vendor/main
		{"", "example.com/lib/b/...", 1},  // empty directory
		{"", "example.com/lib/main", 0},   // main package
		{"example.com/lib", "", 1},        // working directory
		{"example.com/lib", "./...", 2},   // working directory recursive and ignore internal/vendor/main
		{"example.com/lib/b", "./...", 1}, // empty working directory
		{"example.com/lib/main", "", 0},   // main package
	}

	oldPath := os.Getenv("GOPATH")
	defer func() {
		if err := os.Setenv("GOPATH", oldPath); err != nil {
			t.Fatalf("cannot setenv in defer: %s", err)
		}
	}()
	if err := os.Setenv("GOPATH", filepath.Join(testdataDir, "gopath")); err != nil {
		t.Fatalf("cannot setenv: %s", err)
	}

	for _, test := range tests {
		t.Logf("Test: %#v", test)
		err := os.Chdir(filepath.Join(testdataDir, "gopath", "src", test.wd))
		if err != nil {
			t.Errorf("Cannot chdir: %s", err)
		}

		rel, rec, err := RelativePathToTarget(test.path)
		if err != nil {
			t.Fatalf("unexpected error from RelativePathToTarget: %v", err)
		}

		git, err := NewGit(rel)
		if err != nil {
			t.Errorf("Cannot get new git: %s", err)
		}
		checker := New(SetVCS(git))

		changes, err := checker.Check(rel, rec, "HEAD~1", "HEAD")
		if err != nil {
			t.Errorf("Check error: %s", err)
		}

		if test.exp != len(changes) {
			t.Errorf("exp %d got %d", test.exp, len(changes))
		}
	}
}
