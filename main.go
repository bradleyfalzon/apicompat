package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"sort"
)

func main() {
	log.Println("Starting...")

	// arguments
	oldRevID := "HEAD~1"
	newRevID := "HEAD"

	log.Printf("old rev: %v, new rev: %v", oldRevID, newRevID)

	// new vcs
	vcs := git{}

	oldDecls, err := parse(vcs, oldRevID)
	if err != nil {
		log.Fatal(err)
	}

	newDecls, err := parse(vcs, newRevID)
	if err != nil {
		log.Fatal(err)
	}

	for pkgName, decls := range oldDecls {
		log.Printf("Processing pkg %s with %d declarations", pkgName, len(decls))
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

// TODO move this to a method, which already has vcs and other options set
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
		switch astDecl.(type) {
		case *ast.GenDecl:
			switch astDecl.(*ast.GenDecl).Specs[0].(type) {
			case *ast.ValueSpec:
				// var / const
				id = astDecl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names[0].Name
			case *ast.TypeSpec:
				// type struct/interface/etc
				id = astDecl.(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Name.Name
			}
		case *ast.FuncDecl:
			// function or method
			id = astDecl.(*ast.FuncDecl).Name.Name
			if astDecl.(*ast.FuncDecl).Recv != nil {
				expr := astDecl.(*ast.FuncDecl).Recv.List[0].Type
				switch expr.(type) {
				case *ast.UnaryExpr:
					recv = expr.(*ast.UnaryExpr).X.(*ast.Ident).Name
				case *ast.StarExpr:
					recv = expr.(*ast.StarExpr).X.(*ast.Ident).Name
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
