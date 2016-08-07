package abicheck

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"testing"
)

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
	c := New()
	c.vcs = vcs

	changes, err := c.Check("rev1", "rev2")
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
		t.Errorf("got:\n%v\nexp:\n%v\n", buf.String(), string(exp))
	}
}
