package abicheck

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"sync"
	"time"
)

// Checker is used to check for changes between two versions of a package.
type Checker struct {
	vcs    vcs
	bName  string
	aName  string
	bFset  *token.FileSet
	aFset  *token.FileSet
	bDecls revDecls
	aDecls revDecls
	err    error

	parseTime time.Duration
	diffTime  time.Duration
	sortTime  time.Duration
}

// TODO New returns a Checker with
func New(before, after string) *Checker {
	return &Checker{
		vcs:   git{}, // TODO make checker auto discover
		bName: before,
		aName: after,
	}
}

func (c *Checker) Check() ([]Change, error) {
	var wg sync.WaitGroup

	// Parse revisions from VCS into go/ast

	start := time.Now()
	wg.Add(1)
	go func() {
		c.bFset, c.bDecls = c.parse(c.bName)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		c.aFset, c.aDecls = c.parse(c.aName)
		wg.Done()
	}()

	wg.Wait()
	c.parseTime = time.Since(start)

	if c.err != nil {
		// Error parsing, don't continue
		return nil, c.err
	}

	start = time.Now()
	err, changes := compareRevs(c.bDecls, c.aDecls)
	if err != nil {
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "error processing diff: %s", err)
		if derr, ok := err.(*diffError); ok {
			ast.Fprint(&buf, c.bFset, derr.bdecl, ast.NotNilFilter)
			ast.Fprint(&buf, c.aFset, derr.adecl, ast.NotNilFilter)
		}
		return nil, errors.New(buf.String())
	}
	c.diffTime += time.Since(start)

	start = time.Now()
	sort.Sort(byID(changes))
	c.sortTime += time.Since(start)

	return changes, nil
}

func (c *Checker) parse(rev string) (*token.FileSet, revDecls) {
	files, err := c.vcs.ReadDir(rev, "")
	if err != nil {
		c.err = err
		return nil, nil
	}

	fset := token.NewFileSet()
	decls := make(map[string]map[string]ast.Decl) // package to id to decls
	// TODO is there a concurrency opportunity here?
	for _, file := range files {
		contents, err := c.vcs.ReadFile(rev, file)
		if err != nil {
			c.err = fmt.Errorf("could not read file %s at revision %s: %s", file, rev, err)
			return nil, nil
		}

		filename := rev + ":" + file
		src, err := parser.ParseFile(fset, filename, contents, 0)
		if err != nil {
			c.err = fmt.Errorf("could not parse file %s at revision %s: %s", file, rev, err)
			return nil, nil
		}

		pkgName := src.Name.Name
		if decls[pkgName] == nil {
			decls[pkgName] = make(map[string]ast.Decl)
		}
		decls[pkgName] = pkgDecls(src.Decls)
	}

	return fset, decls
}

func pkgDecls(astDecls []ast.Decl) map[string]ast.Decl {
	decls := make(map[string]ast.Decl)
	for _, astDecl := range astDecls {
		switch d := astDecl.(type) {
		case *ast.GenDecl:
			for i := range d.Specs {
				var (
					id string
					// gdecl splits declaration blocks into individual declarations to view
					// only changed declarations, instead of all, I don't imagine it's needed
					// for TypeSpec (just ValueSpec
					gdecl *ast.GenDecl
				)
				switch s := d.Specs[i].(type) {
				case *ast.ValueSpec:
					// var / const
					id = s.Names[0].Name
					gdecl = &ast.GenDecl{Tok: d.Tok, Specs: []ast.Spec{s}}
				case *ast.TypeSpec:
					// type struct/interface/etc
					id = s.Name.Name
					gdecl = &ast.GenDecl{Tok: d.Tok, Specs: []ast.Spec{s}}
				default:
					// import or possibly other
					continue
				}
				if ast.IsExported(id) {
					decls[id] = gdecl
				}
			}
		case *ast.FuncDecl:
			// function or method
			var (
				id   string = d.Name.Name
				recv string
			)
			// check if we have a receiver (and not just `func () Method() {}`)
			if d.Recv != nil && len(d.Recv.List) > 0 {
				expr := d.Recv.List[0].Type
				switch e := expr.(type) {
				case *ast.Ident:
					recv = e.Name
				case *ast.StarExpr:
					recv = e.X.(*ast.Ident).Name
				}
				id = recv + "." + id
			}
			// If it's exported and it's either not a receiver OR the receiver is also exported
			if ast.IsExported(id) && recv == "" || ast.IsExported(recv) {
				// We're not interested in the body, nil it, alternatively we could set an
				// Body.List, but that included parenthesis on different lines when printed
				astDecl.(*ast.FuncDecl).Body = nil
				decls[id] = astDecl
			}
		default:
			panic(fmt.Errorf("Unknown decl type: %#v", astDecl))
		}
	}
	return decls
}

// Timing returns individual phase timing information
func (c Checker) Timing() (parseTime, diffTime, sortTime time.Duration) {
	return c.parseTime, c.diffTime, c.sortTime
}
