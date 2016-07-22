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

	newFset, newDecls, err := parse(vcs, newRevID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %s\n", newRevID, err.Error())
		os.Exit(1)
	}

	oldFset, oldDecls, err := parse(vcs, oldRevID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing %s: %s\n", oldRevID, err.Error())
		os.Exit(1)
	}

	for pkgName, decls := range oldDecls {
		if _, ok := newDecls[pkgName]; ok {
			err, changes := diff(decls, newDecls[pkgName])
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				if derr, ok := err.(*diffError); ok {
					ast.Fprint(os.Stderr, oldFset, derr.bdecl, ast.NotNilFilter)
					ast.Fprint(os.Stderr, newFset, derr.adecl, ast.NotNilFilter)
				}
				return
			}

			sort.Sort(byID(changes))
			for _, change := range changes {
				fmt.Println(change)
			}
		}
	}
}

func parse(vcs vcs, rev string) (*token.FileSet, map[string]decls, error) {
	files, err := vcs.ReadDir(rev, "")
	if err != nil {
		return nil, nil, err
	}

	fset := token.NewFileSet()
	decls := make(map[string]decls) // package to id to decls
	for _, file := range files {
		contents, err := vcs.ReadFile(rev, file)
		if err != nil {
			return nil, nil, fmt.Errorf("could not read file %s at revision %s: %s", file, rev, err)
		}

		filename := rev + ":" + file
		src, err := parser.ParseFile(fset, filename, contents, 0)
		if err != nil {
			return nil, nil, fmt.Errorf("could not parse file %s at revision %s: %s", file, rev, err)
		}

		pkgName := src.Name.Name

		if decls[pkgName] == nil {
			decls[pkgName] = make(map[string]ast.Decl)
		}
		for id, decl := range getDecls(src.Decls) {
			decls[pkgName][id] = decl
		}
	}

	return fset, decls, nil
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
