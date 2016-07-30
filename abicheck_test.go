package abicheck

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	// Create strvcs and fill it with test data
	vcs := strvcs{}

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
	c := New("rev1", "rev2")
	c.vcs = vcs

	changes, err := c.Check()
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
		err := ioutil.WriteFile("testdata/exp.txt", buf.Bytes(), os.FileMode(0644))
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
		t.Errorf("got:\n%v\nexp:\n%v\n", buf, string(exp))
	}
}