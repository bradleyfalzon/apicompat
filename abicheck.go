package abicheck

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
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
	bTypes map[string]*types.Checker
	aTypes map[string]*types.Checker
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
		c.bFset, c.bDecls, c.bTypes = c.parse(c.bName)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		c.aFset, c.aDecls, c.aTypes = c.parse(c.aName)
		wg.Done()
	}()

	wg.Wait()
	c.parseTime = time.Since(start)

	if c.err != nil {
		// Error parsing, don't continue
		return nil, c.err
	}

	start = time.Now()
	changes, err := c.compareDecls()
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

func (c *Checker) parse(rev string) (*token.FileSet, revDecls, map[string]*types.Checker) {
	files, err := c.vcs.ReadDir(rev, "")
	if err != nil {
		c.err = err
		return nil, nil, nil
	}

	typConfig := &types.Config{
		IgnoreFuncBodies:         true,
		DisableUnusedImportCheck: true,
		Importer:                 importer.Default(),
	}

	fset := token.NewFileSet()
	typs := make(map[string]*types.Checker)
	decls := make(map[string]map[string]ast.Decl) // package to id to decls
	// TODO is there a concurrency opportunity here?
	for _, file := range files {
		contents, err := c.vcs.ReadFile(rev, file)
		if err != nil {
			c.err = fmt.Errorf("could not read file %s at revision %s: %s", file, rev, err)
			return nil, nil, nil
		}

		filename := rev + ":" + file
		src, err := parser.ParseFile(fset, filename, contents, 0)
		if err != nil {
			c.err = fmt.Errorf("could not parse file %s at revision %s: %s", file, rev, err)
			return nil, nil, nil
		}

		pkgName := src.Name.Name
		if decls[pkgName] == nil {
			decls[pkgName] = make(map[string]ast.Decl)
		}
		decls[pkgName] = pkgDecls(src.Decls)

		if typs[pkgName] == nil {
			pkg := types.NewPackage("", pkgName)
			info := &types.Info{Types: make(map[ast.Expr]types.TypeAndValue)}
			typs[pkgName] = types.NewChecker(typConfig, fset, pkg, info)
		}
		err = typs[pkgName].Files([]*ast.File{src})
		if err != nil {
			c.err = fmt.Errorf("could not get types for file %s at revision %s: %s", file, rev, err)
			return nil, nil, nil
		}
	}

	return fset, decls, typs
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

type ChangeType uint8

const (
	ChangeError ChangeType = iota
	ChangeUnknown
	ChangeNone
	ChangeNonBreaking
	ChangeBreaking
)

func (c ChangeType) String() string {
	switch c {
	case ChangeError:
		return "parse error"
	case ChangeUnknown:
		return "unknowable"
	case ChangeNone:
		return "no change"
	case ChangeNonBreaking:
		return "non-breaking change"
	case ChangeBreaking:
		return "breaking change"
	}
	panic(fmt.Sprintf("unknown ChangeType: %d", c))
}

type OpType uint8

const (
	OpAdd OpType = iota
	OpRemove
	OpChange
)

func (op OpType) String() string {
	switch op {
	case OpAdd:
		return "added"
	case OpRemove:
		return "removed"
	case OpChange:
		return "changed"
	}
	panic(fmt.Sprintf("unknown operation type: %d", op))
}

// change is the ast declaration containing the before and after
type Change struct {
	Pkg     string
	ID      string
	Summary string
	Op      OpType
	Change  ChangeType
	Before  ast.Decl
	After   ast.Decl
}

func (c Change) String() string {
	fset := token.FileSet{} // only require non-nil fset
	pcfg := printer.Config{Mode: printer.RawFormat, Indent: 1}
	buf := bytes.Buffer{}

	if c.Op == OpChange {
		fmt.Fprintf(&buf, "%s (%s - %s)\n", c.Op, c.Change, c.Summary)
	} else {
		fmt.Fprintln(&buf, c.Op)
	}

	if c.Before != nil {
		pcfg.Fprint(&buf, &fset, c.Before)
		fmt.Fprintln(&buf)
	}
	if c.After != nil {
		pcfg.Fprint(&buf, &fset, c.After)
		fmt.Fprintln(&buf)
	}
	return buf.String()
}

// byID implements sort.Interface for []change based on the id field
type byID []Change

func (a byID) Len() int           { return len(a) }
func (a byID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byID) Less(i, j int) bool { return a[i].ID < a[j].ID }

// revDecls is a map between a package to an id to ast.Decl, where the id is
// a unique name to match declarations for before and after
type revDecls map[string]map[string]ast.Decl

type diffError struct {
	summary string
	bdecl,
	adecl ast.Decl
	bpos,
	apos token.Pos
}

func (e diffError) Error() string {
	return e.summary
}

// compareDecls compares a Checker's before and after declarations and returns
// all changes or nil and an error
func (c Checker) compareDecls() ([]Change, error) {
	var changes []Change

	for pkg, bDecls := range c.bDecls {
		aDecls, ok := c.aDecls[pkg]
		if !ok {
			continue
		}

		for id, bDecl := range bDecls {
			aDecl, ok := aDecls[id]
			if !ok {
				// in before, not in after, therefore it was removed
				changes = append(changes, Change{Pkg: pkg, ID: id, Op: OpRemove, Before: bDecl})
				continue
			}

			// in before and in after, check if there's a difference
			changeType, summary := compareDecl(bDecl, aDecl)

			switch changeType {
			case ChangeNone, ChangeUnknown:
				continue
			case ChangeError:
				return nil, &diffError{summary: summary, bdecl: bDecl, adecl: aDecl}
			}

			changes = append(changes, Change{
				Pkg:     pkg,
				ID:      id,
				Op:      OpChange,
				Change:  changeType,
				Summary: summary,
				Before:  bDecl,
				After:   aDecl,
			})
		}

		for id, aDecl := range aDecls {
			if _, ok := bDecls[id]; !ok {
				// in after, not in before, therefore it was added
				changes = append(changes, Change{Pkg: pkg, ID: id, Op: OpAdd, After: aDecl})
			}
		}
	}

	return changes, nil
}
