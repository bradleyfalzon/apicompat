package main

import (
	"fmt"
	"go/ast"
	"log"
	"reflect"
	"strings"
)

// change is the ast declaration containing the before and after
type change struct {
	before ast.Decl
	after  ast.Decl
}

// decls is a map of an identifier to actual ast, where the id is a unique
// name to match declarations for before and after
type decls map[string]ast.Decl

func (d decls) String() string {
	var out string
	for id := range d {
		// todo have a string method on each decl
		out += fmt.Sprintf("declaration id: %v\n", id)
	}
	return out
}

func diff(bdecls map[string]ast.Decl, adecls map[string]ast.Decl) (changes []change) {
	log.Println("determining differences...")

	for id, decl := range bdecls {
		if _, ok := adecls[id]; !ok {
			// in before, not in after, therefore it was removed
			changes = append(changes, change{before: decl})
			continue
		}

		// in before and in after, check if there's a difference
		if equal(bdecls[id], adecls[id]) {
			// no changes
			continue
		}
		changes = append(changes, change{before: decl, after: adecls[id]})
	}

	for id, decl := range adecls {
		if _, ok := bdecls[id]; !ok {
			// in after, not in before, therefore it was added
			changes = append(changes, change{after: decl})
		}
	}

	return changes
}

// equal compares two declarations and returns true if they do not have
// incompatible changes. For example, comments aren't compared, names of
// arguments aren't compared etc.
func equal(before, after ast.Decl) bool {
	// compare types, ignore comments etc, so reflect.DeepEqual isn't good enough

	if reflect.TypeOf(before) != reflect.TypeOf(after) {
		// Declaration type changed, such as GenDecl to FuncDecl (eg var/const to func)
		return false
	}

	switch before.(type) {
	case *ast.GenDecl:
		if reflect.TypeOf(before.(*ast.GenDecl).Specs[0]) != reflect.TypeOf(after.(*ast.GenDecl).Specs[0]) {
			// Spec changed, such as ValueSpec to TypeSpec (eg var/const to struct)
			return false
		}

		switch before.(*ast.GenDecl).Specs[0].(type) {
		case *ast.ValueSpec:
			// var / const
			if before.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Type.(*ast.Ident).Name != after.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Type.(*ast.Ident).Name {
				// type changed
				return false
			}
		case *ast.TypeSpec:
			// type struct/interface/aliased

			if reflect.TypeOf(before.(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type) != reflect.TypeOf(after.(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type) {
				// Spec change, such as from StructType to InterfaceType or different aliased types
				return false
			}

			switch before.(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(type) {
			case *ast.InterfaceType:
				beforeIface := before.(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(*ast.InterfaceType)
				afterIface := after.(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(*ast.InterfaceType)

				// interfaces don't care if methods are removed, so discard those
				added, _, changed := diffFields(beforeIface.Methods.List, afterIface.Methods.List)

				if len(added) > 0 || len(changed) > 0 {
					// Fields were removed or changed types
					return false
				}
			case *ast.StructType:
				beforeStruct := before.(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(*ast.StructType)
				afterStruct := after.(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(*ast.StructType)

				// structs don't care if fields were added, so discard those
				_, removed, changed := diffFields(beforeStruct.Fields.List, afterStruct.Fields.List)

				if len(removed) > 0 || len(changed) > 0 {
					// Fields were removed or changed types
					return false
				}
			case *ast.Ident:
				// alias
				panic("not yet implemented")
			}
		}
	case *ast.FuncDecl:
		if !equalFieldTypes(before.(*ast.FuncDecl).Type.Params.List, after.(*ast.FuncDecl).Type.Params.List) {
			return false
		}

		if before.(*ast.FuncDecl).Type.Results != nil {
			if after.(*ast.FuncDecl).Type.Results == nil {
				// removed return parameter
				return false
			}

			// Only check if we're changing/removing return parameters
			if !equalFieldTypes(before.(*ast.FuncDecl).Type.Results.List, after.(*ast.FuncDecl).Type.Results.List) {
				return false
			}
		}
	default:
		panic(fmt.Errorf("Unknown type: %T", before))
	}
	return true
}

// equalFieldTypes compares two ast.FieldLists to ensure all types match
func equalFieldTypes(a, b []*ast.Field) bool {
	if len(a) != len(b) {
		// different amount of parameters
		return false
	}

	for i, li := range a {
		if li.Type != b[i].Type {
			// type changed
			return false
		}
	}
	return true
}

func diffFields(before, after []*ast.Field) (added, removed, changed []*ast.Field) {
	// Presort after for quicker matching of fieldname -> type, may not be worthwhile
	AfterMembers := make(map[string]string)
	for _, field := range after {
		AfterMembers[field.Names[0].Name] = typeToString(field.Type)
	}

	for _, field := range before {
		if afterType, ok := AfterMembers[field.Names[0].Name]; ok {
			if afterType != typeToString(field.Type) {
				// changed
				changed = append(changed, field)
			}
			delete(AfterMembers, field.Names[0].Name)
			continue
		}

		// Removed
		removed = append(removed, field)
	}

	// What's left in afterFields has added
	for member := range AfterMembers {
		for _, field := range after {
			if field.Names[0].Name == member {
				added = append(added, field)
			}
		}
	}

	return added, removed, changed
}

// typeToString returns a string representation of a fields type (if it's an
// ident) or if it's a funcType, the params and return types
func typeToString(ident ast.Expr) string {
	switch ident.(type) {
	case *ast.Ident:
		// perhaps a struct
		return ident.(*ast.Ident).Name
	case *ast.FuncType:
		// perhaps interface/func
		// TODO change to buffer
		var (
			params  []string
			results []string
		)
		for _, list := range ident.(*ast.FuncType).Params.List {
			params = append(params, list.Type.(*ast.Ident).Name)
		}
		if ident.(*ast.FuncType).Results != nil {
			for _, list := range ident.(*ast.FuncType).Results.List {
				results = append(results, list.Type.(*ast.Ident).Name)
			}
		}
		return fmt.Sprintf("(%s) (%s)", strings.Join(params, ","), strings.Join(results, ","))
	}
	panic(fmt.Errorf("Unknown decl type: %T", ident))
}
