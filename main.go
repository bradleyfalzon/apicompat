package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
)

func main() {
	const (
		oldRevID = "HEAD~1"
		newRevID = "HEAD"
	)

	// new vcs
	var vcs git

	oldDecls, err := parse(vcs, oldRevID)
	if err != nil {
		fmt.Printf("Error parsing %s: %s\n", oldRevID, err.Error())
		os.Exit(1)
	}

	newDecls, err := parse(vcs, newRevID)
	if err != nil {
		fmt.Printf("Error parsing %s: %s\n", newRevID, err.Error())
		os.Exit(1)
	}

	for pkgName, decls := range oldDecls {
		if _, ok := newDecls[pkgName]; ok {
			changes := diff(decls, newDecls[pkgName])
			sort.Sort(byID(changes))
			for _, change := range changes {
				fmt.Println(change)
			}
		}
	}
}

func parse(vcs vcs, revision string) (map[string]decls, error) {
	files, err := vcs.ReadDir(revision, "")
	if err != nil {
		return nil, err
	}

	pkgs, err := parseFiles(vcs, revision, files)
	if err != nil {
		return nil, err
	}

	decls := make(map[string]decls) // package to id to decls
	for pkgName, pkg := range pkgs {
		for _, file := range pkg.Files {
			if decls[pkgName] == nil {
				decls[pkgName] = make(map[string]ast.Decl)
			}
			for id, decl := range getDecls(file.Decls) {
				decls[pkgName][id] = decl
			}
		}
	}

	return decls, nil
}

// TODO(bradleyfalzon): move this to a method, which already has vcs and other options set
func parseFiles(vcs vcs, rev string, files []string) (map[string]*ast.Package, error) {
	fset := token.NewFileSet()
	pkgs := make(map[string]*ast.Package)
	for _, file := range files {
		contents, err := vcs.ReadFile(rev, file)
		if err != nil {
			return nil, fmt.Errorf("could not read file %s at revision %s: %s", file, rev, err)
		}

		filename := rev + ":" + file
		src, err := parser.ParseFile(fset, filename, contents, 0)
		if err != nil {
			return nil, fmt.Errorf("could not parse file %s at revision %s: %s", file, rev, err)
		}

		pkgName := src.Name.Name
		pkg, found := pkgs[pkgName]
		if !found {
			pkg = &ast.Package{
				Name:  pkgName,
				Files: make(map[string]*ast.File),
			}
			pkgs[pkgName] = pkg
		}
		pkg.Files[filename] = src
	}

	return pkgs, nil
}

func getDecls(astDecls []ast.Decl) decls {
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
			if d.Recv != nil {
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
