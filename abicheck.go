package abicheck

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// When the path is not set, it means the current working directory
// go/build understands this as "."
const cwd = "."

// Checker is used to check for changes between two versions of a package.
type Checker struct {
	vcs     VCS
	vlog    io.Writer
	path    string // import path
	recurse bool   // scan paths recursively

	b map[string]pkg
	a map[string]pkg
}

// TODO New returns a Checker with
func New(options ...func(*Checker)) *Checker {
	c := &Checker{}
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

// Blank revision means use VCSs default
func (c *Checker) Check(path, beforeRev, afterRev string) ([]Change, error) {
	// If revision is unset use VCS's default revision
	dBefore, dAfter := c.vcs.DefaultRevision()
	if beforeRev == "" {
		beforeRev = dBefore
	}
	if afterRev == "" {
		afterRev = dAfter
	}

	// If path is unset, use local directory
	c.path = path
	if path == "" {
		c.path = cwd
	}

	// Detect recursion
	if strings.HasSuffix(c.path, string(os.PathSeparator)+"...") {
		c.recurse = true
		c.path = c.path[:len(c.path)-len(string(os.PathSeparator)+"...")]
	}
	c.logf("import path: %q before: %q after: %q recursive: %v\n", c.path, beforeRev, afterRev, c.recurse)

	// Parse revisions from VCS into go/ast
	start := time.Now()
	var err error
	if c.b, err = c.parse(beforeRev); err != nil {
		return nil, err
	}
	if c.a, err = c.parse(afterRev); err != nil {
		return nil, err
	}
	parse := time.Since(start)

	start = time.Now()
	changes, err := c.compareDecls()
	if err != nil {
		var buf bytes.Buffer
		fmt.Fprintf(&buf, "error comparing declarations: %s\n", err)
		if derr, ok := err.(*diffError); ok {
			_ = ast.Fprint(&buf, c.b[derr.pkg].fset, derr.bdecl, ast.NotNilFilter)
			_ = ast.Fprint(&buf, c.a[derr.pkg].fset, derr.adecl, ast.NotNilFilter)
		}
		return nil, errors.New(buf.String())
	}
	diff := time.Since(start)

	start = time.Now()
	sort.Sort(byID(changes))
	sort := time.Since(start)

	c.logf("Timing: parse: %v, diff: %v, sort: %v, total: %v\n", parse, diff, sort, parse+diff+sort)
	c.logf("Changes detected: %v\n", len(changes))

	return changes, nil
}

func (c Checker) logf(format string, a ...interface{}) {
	if c.vlog != nil {
		fmt.Fprintf(c.vlog, format, a...)
	}
}

type pkg struct {
	importPath string // import path
	fset       *token.FileSet
	decls      map[string]ast.Decl
	info       *types.Info
}

func (c Checker) parse(rev string) (pkgs map[string]pkg, err error) {
	c.logf("Parsing revision: %s path: %s recurse: %v\n", rev, c.path, c.recurse)

	// c.path is either dot or import path
	paths := []string{c.path}
	if c.recurse {
		// Technically this isn't correct, GOPATH could be a list
		var (
			dir    = filepath.Join(os.Getenv("GOPATH"), "src")
			prefix = ""
		)
		if c.path == cwd {
			// could c.path = getwd instead ?
			if dir, err = os.Getwd(); err != nil {
				return nil, err
			}
			prefix = "." + string(os.PathSeparator)
		}
		paths = append(paths, c.getDirsRecursive(dir, rev, c.path, prefix)...)
	}

	c.logf("building paths: %s\n", paths)

	pkgs = make(map[string]pkg)
	for _, path := range paths {
		p, err := c.parseDir(rev, path)
		if err != nil {
			// skip errors if we're recursing and the error is no buildable sources
			if !c.recurse || !strings.Contains(err.Error(), "no buildable") {
				return pkgs, err
			}
		}
		pkgs[p.importPath] = p
	}
	return pkgs, nil
}

// getDirsRecursive returns relative paths to all subdirectories within base
// at revision rev. Paths can be prefixed with prefix
func (c Checker) getDirsRecursive(base, rev, rel, prefix string) (dirs []string) {
	paths, err := c.vcs.ReadDir(rev, filepath.Join(base, rel))
	if err != nil {
		c.logf("could not read path: %s revision: %s, error: %s\n", filepath.Join(base, rel), rev, err)
		return dirs
	}

	for _, path := range paths {
		if !path.IsDir() || path.Name() == "testdata" {
			continue
		}

		dirs = append(dirs, prefix+filepath.Join(rel, path.Name()))
		dirs = append(dirs, c.getDirsRecursive(base, rev, filepath.Join(rel, path.Name()), prefix)...)
	}
	return dirs
}

func (c Checker) parseDir(rev, dir string) (pkg, error) {

	// Use go/build to get the list of files relevant for a specfic OS and ARCH

	var ctx = build.Default
	ctx.ReadDir = func(dir string) ([]os.FileInfo, error) {
		return c.vcs.ReadDir(rev, dir)
	}
	ctx.OpenFile = func(path string) (io.ReadCloser, error) {
		return c.vcs.OpenFile(rev, path)
	}
	ctx.GOPATH = os.Getenv("GOPATH")

	// wd is for relative imports, such as "."
	wd, err := os.Getwd()
	if err != nil {
		return pkg{}, err
	}
	ipkg, err := ctx.Import(dir, wd, 0)
	if err != nil {
		return pkg{}, fmt.Errorf("go/build error: %v", err)
	}

	var (
		fset     = token.NewFileSet()
		pkgFiles []*ast.File
	)
	for _, file := range ipkg.GoFiles {
		contents, err := c.vcs.OpenFile(rev, filepath.Join(ipkg.Dir, file))
		if err != nil {
			return pkg{}, fmt.Errorf("could not read file %q at revision %q: %s", file, rev, err)
		}

		filename := file
		if rev != revisionFS {
			filename = rev + ":" + file
		}
		src, err := parser.ParseFile(fset, filename, contents, 0)
		if err != nil {
			return pkg{}, fmt.Errorf("could not parse file %q at revision %q: %s", file, rev, err)
		}

		pkgFiles = append(pkgFiles, src)
	}

	// Loop through all the parsed files and type check them

	p := pkg{
		importPath: ipkg.ImportPath,
		fset:       fset,
		info: &types.Info{
			Types: make(map[ast.Expr]types.TypeAndValue),
			Defs:  make(map[*ast.Ident]types.Object),
			Uses:  make(map[*ast.Ident]types.Object),
		},
	}

	conf := &types.Config{
		IgnoreFuncBodies:         true,
		DisableUnusedImportCheck: true,
		Importer:                 importer.Default(),
	}
	_, err = conf.Check(ipkg.ImportPath, fset, pkgFiles, p.info)
	if err != nil {
		return pkg{}, fmt.Errorf("go/types error: %v", err)
	}

	// Get declarations and nil their bodies, so do it last
	p.decls = pkgDecls(pkgFiles)

	return p, nil
}

// pkgDecls returns all declarations that need to be checked, this includes
// all exported declarations as well as unexported types that are returned by
// exported functions. Structs have both exported and unexported fields.
func pkgDecls(files []*ast.File) map[string]ast.Decl {
	var (
		// exported values and functions
		decls = make(map[string]ast.Decl)

		// unexported values and functions
		priv = make(map[string]ast.Decl)

		// IDs of ValSpecs that are returned by a function
		returned []string
	)
	for _, file := range files {
		for _, astDecl := range file.Decls {
			switch d := astDecl.(type) {
			case *ast.GenDecl:
				// split declaration blocks into individual declarations to view
				// only changed declarations, instead of all, I don't imagine it's needed
				// for TypeSpec (just ValueSpec), it does this by creating a new GenDecl
				// with just that loops spec
				for i := range d.Specs {
					var (
						id   string
						decl *ast.GenDecl
					)
					switch s := d.Specs[i].(type) {
					case *ast.ValueSpec:
						// var / const
						// split multi assignments into individial declarations to simplify matching
						for j := range s.Names {
							id = s.Names[j].Name
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
							decl = &ast.GenDecl{Tok: d.Tok, Specs: []ast.Spec{spec}}
						}
					case *ast.TypeSpec:
						// type struct/interface/etc
						id = s.Name.Name
						decl = &ast.GenDecl{Tok: d.Tok, Specs: []ast.Spec{s}}
					case *ast.ImportSpec:
						// ignore
						continue
					default:
						panic(fmt.Errorf("Unknown declaration: %#v", s))
					}
					if ast.IsExported(id) {
						decls[id] = decl
						continue
					}
					priv[id] = decl
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
				astDecl.(*ast.FuncDecl).Body = nil
				// If it's exported and it's either not a receiver OR the receiver is also exported
				if ast.IsExported(d.Name.Name) && (recv == "" || ast.IsExported(recv)) {
					// We're not interested in the body, nil it, alternatively we could set an
					// Body.List, but that included parenthesis on different lines when printed
					decls[id] = astDecl

					// note which ident types are returned, to find those that were not
					// exported but are returned and therefor need to be checked
					if d.Type.Results != nil {
						for _, field := range d.Type.Results.List {
							switch ftype := field.Type.(type) {
							case *ast.Ident:
								returned = append(returned, ftype.String())
							case *ast.StarExpr:
								if ident, ok := ftype.X.(*ast.Ident); ok {
									returned = append(returned, ident.String())
								}
							}
						}
					}
				} else {
					priv[id] = astDecl
				}
			default:
				panic(fmt.Errorf("Unknown decl type: %#v", astDecl))
			}
		}
	}

	// Add any value specs returned by a function, but wasn't exported
	for _, id := range returned {
		// Find unexported types that need to be checked
		if _, ok := priv[id]; ok {
			decls[id] = priv[id]
		}

		// Find exported functions with unexported receivers that also need to be checked
		for rid, decl := range priv {
			// len(type)+1 to account for dot separator
			if len(rid) <= len(id)+1 {
				continue
			}
			pid, pfunc := rid[:len(id)], rid[len(id)+1:]
			if id == pid && ast.IsExported(pfunc) {
				decls[rid] = decl
			}
		}
	}
	return decls
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
			c := Change{Pkg: pkgName, Change: Breaking, Msg: "package removed"}
			changes = append(changes, c)
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
