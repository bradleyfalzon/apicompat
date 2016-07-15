package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"sort"
	"testing"
)

func TestParse(t *testing.T) {
	vcs := strvcs{}

	rev1, err := ioutil.ReadFile("testdata/a.go")
	if err != nil {
		t.Fatal("cannot load test data for rev1:", err)
	}

	rev2, err := ioutil.ReadFile("testdata/b.go")
	if err != nil {
		t.Fatal("cannot load test data for rev2:", err)
	}

	vcs.SetFile("rev1", "abitest.go", rev1)
	vcs.SetFile("rev2", "abitest.go", rev2)

	oldDecls, err := parse(vcs, "rev1")
	newDecls, err := parse(vcs, "rev2")

	got := bytes.NewBufferString("")
	for pkgName, decls := range oldDecls {
		if _, ok := newDecls[pkgName]; ok {
			changes := diff(decls, newDecls[pkgName])
			sort.Sort(byID(changes))
			for _, change := range changes {
				fmt.Fprint(got, change)
			}
		}
	}

	if len(os.Args) > 1 && os.Args[1] == "update" {
		err := ioutil.WriteFile("testdata/exp.txt", got.Bytes(), os.FileMode(0644))
		if err != nil {
			t.Fatal("could not write exp data:", err)
		}
	}

	exp, err := ioutil.ReadFile("testdata/exp.txt")
	if err != nil {
		t.Fatal("cannot load expect data:", err)
	}

	if !reflect.DeepEqual(exp, got.Bytes()) {
		t.Errorf("got:\n%v\nexp:\n%v\n", got, string(exp))
	}
}
