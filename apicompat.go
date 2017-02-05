package apicompat

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
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"
)

// When the path is not set, it means the current working directory
// go/build understands this as "."
const cwd = "."

var (
	// errSkipPackage is returned by the parser when a package should be skipped.
	errSkipPackage = errors.New("Skipping package")
	// errImportPathNotFound is returned when the import path cannot be found in
	// any GOPATH.
	errImportPathNotFound = errors.New("import path not found")
	// errNotInGOPATH is returned when the target directory is detected to not
	// be inside any of the $GOPATH.
	errNotInGOPATH = errors.New("target directory not in $GOPATH")
)

// Checker is used to check for changes between two versions of a package.
type Checker struct {
	vcs         VCS
	vlog        io.Writer
	path        string         // import path
	recurse     bool           // scan paths recursively
	excludeFile *regexp.Regexp // exclude files
	excludeDir  *regexp.Regexp // exclude directory

	b map[string]pkg
	a map[string]pkg
}

// New returns a Checker with the given options.
func New(options ...func(*Checker)) *Checker {
	c := &Checker{}
	for _, option := range options {
		option(c)
	}
	return c
}

// SetVCS is an option to New that sets the VCS for the checker.
func SetVCS(vcs VCS) func(*Checker) {
	return func(c *Checker) {
		c.vcs = vcs
	}
}

// SetVLog is an option to New that sets the logger for the checker.
func SetVLog(w io.Writer) func(*Checker) {
	return func(c *Checker) {
		c.vlog = w
	}
}

// SetExcludeFile excludes checking of files based on regexp pattern
func SetExcludeFile(pattern string) func(*Checker) {
	return func(c *Checker) {
		c.excludeFile = regexp.MustCompile(pattern)
	}
}

// SetExcludeDir excludes checking of a directory based on regexp pattern.
// Usually only help when running recursively.
func SetExcludeDir(pattern string) func(*Checker) {
	return func(c *Checker) {
		c.excludeDir = regexp.MustCompile(pattern)
	}
}

// Check an import path and before and after revision for changes. Import path
// maybe empty, if so, the current working directory will be used. If a
// revision is blank, the default VCS revision is used.
func (c *Checker) Check(rel string, recurse bool, beforeRev, afterRev string) ([]Change, error) {
	// If revision is unset use VCS's default revision
	dBefore, dAfter := c.vcs.DefaultRevision()
	if beforeRev == "" {
		beforeRev = dBefore
	}
	if afterRev == "" {
		afterRev = dAfter
	}
	c.recurse = recurse

	var err error
	c.path, err = importPathTo(rel)
	if err != nil {
		return nil, err
	}

	c.logf("import path: %q before: %q after: %q recursive: %v\n", c.path, beforeRev, afterRev, c.recurse)

	// Parse revisions from VCS into go/ast
	start := time.Now()
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

func importPathTo(rel string) (string, error) {
	gopaths := filepath.SplitList(os.Getenv("GOPATH"))
	for _, gopath := range gopaths {
		abs, err := filepath.Abs(rel)
		if err != nil {
			return "", err
		}
		if strings.HasPrefix(abs, gopath) {
			return abs[len(gopath)+2+len("src"):], nil
		}
	}
	return "", errImportPathNotFound
}

// RelativePathToTarget returns the relative path to the given path, wether it's
// an import path or direct path and also returns if the path had recursion
// requested (/...).
func RelativePathToTarget(path string) (rel string, recurse bool, err error) {
	// Detect recursion
	if strings.HasSuffix(path, string(os.PathSeparator)+"...") {
		recurse = true
		path = path[:len(path)-len(string(os.PathSeparator)+"...")]
	}

	// If path is unset, use local directory
	if path == "" || path == "." {
		return ".", recurse, nil
	}

	if _, err := os.Stat(path); err != nil {
		if perr, ok := err.(*os.PathError); ok {
			if serr, ok := perr.Err.(syscall.Errno); ok {
				if serr == syscall.ENOENT { // we might be given a import path.
					var err error
					path, err = findRelativeFromImport(path)
					if err != nil {
						return "", false, err
					}
					return path, recurse, nil
				}
			}
		}
		return "", false, err
	}
	return path, recurse, nil
}

func findRelativeFromImport(path string) (string, error) {
	gopaths := filepath.SplitList(os.Getenv("GOPATH"))
	for _, gopath := range gopaths {
		fullpath := filepath.Join(gopath, "src", path)
		if _, err := os.Stat(fullpath); err == nil {
			wd, err := os.Getwd()
			if err != nil {
				return "", err
			}
			rel, err := filepath.Rel(wd, fullpath)
			if err != nil {
				return "", err
			}
			return rel, nil
		}
	}
	return "", errImportPathNotFound
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
		dir, err := findGOPATH(c.path)
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(dir, "src")
		var prefix string
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
		if c.excludeDir != nil && c.excludeDir.MatchString(path) {
			c.logf("Excluding path: %s\n", path)
			continue
		}
		if strings.Contains(path, "internal/") || strings.Contains(path, "vendor/") {
			c.logf("Excluding path: %s\n", path)
			continue
		}

		p, err := c.parseDir(rev, path)
		if err != nil {
			if err == errSkipPackage {
				continue
			}
			// skip errors if we're recursing and the error is no buildable sources
			if !c.recurse || !strings.Contains(err.Error(), "no buildable") {
				return pkgs, err
			}
		}
		pkgs[p.importPath] = p
	}
	return pkgs, nil
}

func findGOPATH(path string) (string, error) {
	for _, gopath := range filepath.SplitList(os.Getenv("GOPATH")) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		if strings.HasPrefix(abs, gopath) {
			return gopath, nil
		}
	}
	return "", errNotInGOPATH
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

	// Use go/build to get the list of files relevant for a specific OS and ARCH
	ctx := build.Default
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

	if ipkg.Name == "main" {
		return pkg{}, errSkipPackage
	}

	var (
		fset     = token.NewFileSet()
		pkgFiles []*ast.File
	)
	for _, file := range ipkg.GoFiles {
		if c.excludeFile != nil && c.excludeFile.MatchString(file) {
			c.logf("Excluding file: %s\n", file)
			continue
		}

		contents, err := c.vcs.OpenFile(rev, filepath.Join(ipkg.Dir, file))
		if err != nil {
			return pkg{}, fmt.Errorf("could not read file %q at revision %q: %s", file, rev, err)
		}

		filename, err := filepath.Rel(wd, filepath.Join(ipkg.Dir, file))
		if err != nil {
			return pkg{}, fmt.Errorf("could not make path relative for revision %q: %s", rev, err)
		}
		if rev != revisionFS {
			// prefix revision to file's path when reading from vcs and not file system
			filename = rev + ":" + filename
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
// exported functions.
//
// Remove struct's private members and separate indentifier lists
// into one per declaration.
// from: struct { p1, p2 int, P3, P4 uint }
// into: struct { P3 uint, P4 uint }
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

						// Expand multiple names for a type and remove unexported from structs
						switch t := s.Type.(type) {
						case *ast.StructType:
							expandFieldList(t.Fields, true)
						case *ast.InterfaceType:
							for _, m := range t.Methods.List {
								if ftype, ok := m.Type.(*ast.FuncType); ok {
									expandFieldList(ftype.Params, false)
									expandFieldList(ftype.Results, false)
								}
							}
						}
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
					id   = d.Name.Name
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

				// Expand the shorthand type notation
				expandFieldList(d.Type.Params, false)
				expandFieldList(d.Type.Results, false)

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
			dotIndex := strings.IndexRune(rid, '.')
			if dotIndex < 0 {
				continue
			}
			pid, pfunc := rid[:dotIndex], rid[dotIndex+1:]
			if id == pid && ast.IsExported(pfunc) {
				decls[rid] = decl
			}
		}
	}
	return decls
}

// expandFieldList expands an ast.FieldList's shorthand notation:
// (a, b int) to (a int, b int). A ast.FieldList could be function's signature
// struct, interface etc. If isStruct is true, only exported idents are
// returned.
func expandFieldList(fl *ast.FieldList, isStruct bool) {
	if fl == nil || fl.List == nil {
		return
	}
	var newList []*ast.Field
	for _, field := range fl.List {
		fnew := ast.Field{Doc: field.Doc, Type: field.Type, Tag: field.Tag, Comment: field.Comment}
		if len(field.Names) == 0 {
			// Unnamed type, like func() error {}, embedded struct etc
			if keepField(field.Type, isStruct) {
				newList = append(newList, &fnew)
			}
		}
		for _, fname := range field.Names {
			if keepField(fname, isStruct) {
				fcopy := fnew
				fcopy.Names = []*ast.Ident{fname}
				newList = append(newList, &fcopy)
			}
		}
	}
	fl.List = newList
	return
}

func keepField(expr ast.Expr, isStruct bool) bool {
	if !isStruct {
		// Keep all fields
		return true
	}

	// This is a expr from a struct, only keep the fields that are exported.
	// SelectorExpr is always exported, as it wouldn't be accessible otherwise.

	switch etype := expr.(type) {
	case *ast.StarExpr:
		switch estar := etype.X.(type) {
		case *ast.SelectorExpr:
			return true
		case *ast.Ident:
			return ast.IsExported(estar.Name)
		}
	case *ast.SelectorExpr:
		return true
	case *ast.Ident:
		//
		return ast.IsExported(etype.Name)
	}
	panic("this shouldn't happen") // if i had a dollar every time i heard this
}

// Change is the ast declaration containing the before and after
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
	var fset token.FileSet // only require non-nil fset
	var buf bytes.Buffer
	pcfg := printer.Config{Mode: printer.RawFormat, Indent: 1}

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
				c := Change{Pkg: pkgName, ID: id, Change: Breaking, Msg: "declaration removed", Pos: pos(bpkg.fset, bDecl.End()), Before: bDecl}
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
				Pos:    pos(apkg.fset, change.Pos),
				Before: bDecl,
				After:  aDecl,
			})
		}

		for id, aDecl := range apkg.decls {
			if _, ok := bpkg.decls[id]; !ok {
				// in after, not in before, therefore it was added
				c := Change{Pkg: pkgName, ID: id, Change: NonBreaking, Msg: "declaration added", Pos: pos(apkg.fset, aDecl.End()), After: aDecl}
				changes = append(changes, c)
			}
		}
	}
	return changes, nil
}

// pos returns the declaration's position within a file.
func pos(fset *token.FileSet, p token.Pos) string {
	pos := fset.Position(p)
	return fmt.Sprintf("%s:%d", pos.Filename, pos.Line)
}
