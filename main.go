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
		fmt.Printf("Error parsing %s: %s", oldRevID, err.Error())
		os.Exit(1)
	}

	newDecls, err := parse(vcs, newRevID)
	if err != nil {
		fmt.Printf("Error parsing %s: %s", newRevID, err.Error())
		os.Exit(1)
	}

	for pkgName, decls := range oldDecls {
		fmt.Printf("Processing pkg %s with %d declarations\n", pkgName, len(decls))
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
		var (
			id   string // unique
			recv string
		)
		switch d := astDecl.(type) {
		case *ast.GenDecl:
			switch s := d.Specs[0].(type) {
			case *ast.ValueSpec:
				// var / const
				id = s.Names[0].Name
			case *ast.TypeSpec:
				// type struct/interface/etc
				id = s.Name.Name
			}
		case *ast.FuncDecl:
			// function or method
			id = d.Name.Name
			if d.Recv != nil {
				expr := d.Recv.List[0].Type
				switch e := expr.(type) {
				case *ast.UnaryExpr:
					recv = e.X.(*ast.Ident).Name
				case *ast.StarExpr:
					recv = e.X.(*ast.Ident).Name
				}
				id = recv + "." + id
			}
		default:
			panic(fmt.Errorf("Unknown decl type: %T", astDecl))
		}
		// If it's exported and it's either not a receiver OR the receiver is also exported
		if ast.IsExported(id) && recv == "" || ast.IsExported(recv) {
			decls[id] = astDecl
		}
	}
	return decls
}
