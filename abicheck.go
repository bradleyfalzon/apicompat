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
	"io"
	"sort"
	"time"
)

// Checker is used to check for changes between two versions of a package.
type Checker struct {
	vcs  VCS
	vlog io.Writer
	b    map[string]pkg
	a    map[string]pkg

	err error

	parseTime time.Duration
	diffTime  time.Duration
	sortTime  time.Duration
}

// TODO New returns a Checker with
func New(options ...func(*Checker)) *Checker {
	c := &Checker{
		vcs: Git{}, // TODO make checker auto discover
	}

	for _, option := range options {
		option(c)
	}
	return c
}

func SetVCS(vcs VCS) func(*Checker) {
	return func(c *Checker) {
		c.vcs = vcs
	}
}

func SetVLog(w io.Writer) func(*Checker) {
	return func(c *Checker) {
		c.vlog = w
	}
}

func (c *Checker) Check(beforeRev, afterRev string) ([]Change, error) {
	// Parse revisions from VCS into go/ast
	start := time.Now()
	c.b = c.parse(beforeRev)
	c.a = c.parse(afterRev)
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
			_ = ast.Fprint(&buf, c.b[derr.pkg].fset, derr.bdecl, ast.NotNilFilter)
			_ = ast.Fprint(&buf, c.a[derr.pkg].fset, derr.adecl, ast.NotNilFilter)
		}
		return nil, errors.New(buf.String())
	}
	c.diffTime += time.Since(start)

	start = time.Now()
	sort.Sort(byID(changes))
	c.sortTime += time.Since(start)

	c.logf("Parse time: %v, Diff time: %v, Sort time: %v, Total time: %v\n",
		c.parseTime, c.diffTime, c.sortTime, c.parseTime+c.diffTime+c.sortTime)

	c.logf("%v changes detected\n", len(changes))

	return changes, nil
}

func (c Checker) logf(format string, a ...interface{}) {
	if c.vlog != nil {
		fmt.Fprintf(c.vlog, format, a...)
	}
}

type pkg struct {
	fset  *token.FileSet
	decls map[string]ast.Decl
	info  *types.Info
}

func (c *Checker) parse(rev string) map[string]pkg {
	files, err := c.vcs.ReadDir(rev, "")
	if err != nil {
		c.err = err
		return nil
	}

	fset := token.NewFileSet()
	pkgFiles := make(map[string][]*ast.File)
	for _, file := range files {
		contents, err := c.vcs.ReadFile(rev, file)
		if err != nil {
			c.err = fmt.Errorf("could not read file %s at revision %s: %s", file, rev, err)
			return nil
		}

		filename := rev + ":" + file
		src, err := parser.ParseFile(fset, filename, contents, 0)
		if err != nil {
			c.err = fmt.Errorf("could not parse file %s at revision %s: %s", file, rev, err)
			return nil
		}

		pkgName := src.Name.Name
		pkgFiles[pkgName] = append(pkgFiles[pkgName], src)
	}

	// Loop through all the parsed files and type check them

	pkgs := make(map[string]pkg)
	for pkgName, files := range pkgFiles {
		p := pkg{
			fset:  fset,
			decls: make(map[string]ast.Decl),
			info: &types.Info{
				Types: make(map[ast.Expr]types.TypeAndValue),
				Defs:  make(map[*ast.Ident]types.Object),
				Uses:  make(map[*ast.Ident]types.Object),
			},
		}

		for _, file := range files {
			pkgDecls(p.decls, file.Decls)
		}

		conf := &types.Config{
			IgnoreFuncBodies:         true,
			DisableUnusedImportCheck: true,
			Importer:                 importer.Default(),
		}
		_, err := conf.Check("", fset, files, p.info)
		if err != nil {
			c.err = err
			return nil
		}

		pkgs[pkgName] = p
	}
	return pkgs
}

func pkgDecls(decls map[string]ast.Decl, astDecls []ast.Decl) {
	for _, astDecl := range astDecls {
		switch d := astDecl.(type) {
		case *ast.GenDecl:
			// split declaration blocks into individual declarations to view
			// only changed declarations, instead of all, I don't imagine it's needed
			// for TypeSpec (just ValueSpec), it does this by creating a new GenDecl
			// with just that loops spec
			for i := range d.Specs {
				switch s := d.Specs[i].(type) {
				case *ast.ValueSpec:
					// var / const
					// split multi assignments into individial declarations to simplify matching
					for j := range s.Names {
						id := s.Names[j].Name
						spec := &ast.ValueSpec{
							Doc:     s.Doc,
							Names:   []*ast.Ident{s.Names[j]},
							Type:    s.Type,
							Comment: s.Comment,
						}
						if len(s.Values)-1 >= j {
							// Check j is not nil
							spec.Values = []ast.Expr{s.Values[j]}
						}
						if ast.IsExported(id) {
							decls[id] = &ast.GenDecl{Tok: d.Tok, Specs: []ast.Spec{spec}}
						}
					}
				case *ast.TypeSpec:
					// type struct/interface/etc
					id := s.Name.Name
					if ast.IsExported(id) {
						decls[id] = &ast.GenDecl{Tok: d.Tok, Specs: []ast.Spec{s}}
					}
				default:
					// import or possibly other
					continue
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
}

// change is the ast declaration containing the before and after
type Change struct {
	Pkg    string   // Pkg is the name of the package the change occurred in
	ID     string   // ID is an identifier to match a declaration between versions
	Msg    string   // Msg describes the change
	Change string   // Change describes whether it was unknown, no change, non-breaking or breaking change
	Pos    string   // Pos is the ASTs position prefixed with a version
	Before ast.Decl // Before is the previous declaration
	After  ast.Decl // After is the new declaration
}

func (c Change) String() string {
	fset := token.FileSet{} // only require non-nil fset
	pcfg := printer.Config{Mode: printer.RawFormat, Indent: 1}
	buf := bytes.Buffer{}

	fmt.Fprintf(&buf, "%s: %s %s\n", c.Pos, c.Change, c.Msg)

	if c.Before != nil {
		_ = pcfg.Fprint(&buf, &fset, c.Before)
		fmt.Fprintln(&buf)
	}
	if c.After != nil {
		_ = pcfg.Fprint(&buf, &fset, c.After)
		fmt.Fprintln(&buf)
	}
	return buf.String()
}

// byID implements sort.Interface for []change based on the id field
type byID []Change

func (a byID) Len() int           { return len(a) }
func (a byID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byID) Less(i, j int) bool { return a[i].ID < a[j].ID }

type diffError struct {
	err error
	pkg string
	bdecl,
	adecl ast.Decl
}

func (e diffError) Error() string {
	return e.err.Error()
}

// compareDecls compares a Checker's before and after declarations and returns
// all changes or nil and an error
func (c Checker) compareDecls() ([]Change, error) {
	var changes []Change
	for pkgName, bpkg := range c.b {
		apkg, ok := c.a[pkgName]
		if !ok {
			continue
		}

		d := NewDeclChecker(bpkg.info, apkg.info)
		for id, bDecl := range bpkg.decls {
			aDecl, ok := apkg.decls[id]
			if !ok {
				// in before, not in after, therefore it was removed
				c := Change{Pkg: pkgName, ID: id, Change: Breaking, Msg: "declaration removed", Pos: pos(bpkg.fset, bDecl), Before: bDecl}
				changes = append(changes, c)
				continue
			}

			// in before and in after, check if there's a difference
			change, err := d.Check(bDecl, aDecl)
			if err != nil {
				return nil, &diffError{pkg: pkgName, err: err, bdecl: bDecl, adecl: aDecl}
			}

			if change.Change == None {
				continue
			}

			changes = append(changes, Change{
				Pkg:    pkgName,
				ID:     id,
				Change: change.Change,
				Msg:    change.Msg,
				Pos:    pos(apkg.fset, aDecl),
				Before: bDecl,
				After:  aDecl,
			})
		}

		for id, aDecl := range apkg.decls {
			if _, ok := bpkg.decls[id]; !ok {
				// in after, not in before, therefore it was added
				c := Change{Pkg: pkgName, ID: id, Change: NonBreaking, Msg: "declaration added", Pos: pos(apkg.fset, aDecl), After: aDecl}
				changes = append(changes, c)
			}
		}
	}
	return changes, nil
}

// pos returns the declaration's position within a file.
//
// For some reason Pos does not work on a ast.GenDec, it's only working on a
// ast.FuncDec but I'm not certain why. Fortunately, when Pos is invalid, End()
// has always been valid, so just use that.
//
// TODO fixme, this function shouldn't be required for the above reason.
// TODO actually we should just return the pos, leave it up to the app to figure it out
func pos(fset *token.FileSet, decl ast.Decl) string {
	p := decl.Pos()
	if !p.IsValid() {
		p = decl.End()
	}

	pos := fset.Position(p)
	return fmt.Sprintf("%s:%d", pos.Filename, pos.Line)
}
