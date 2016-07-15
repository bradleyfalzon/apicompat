package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"reflect"
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
	var b bytes.Buffer
	for id := range d {
		// todo have a string method on each decl
		fmt.Fprintf(&b, "declaration id: %v\n", id)
	}
	return b.String()
}

func diff(bdecls, adecls decls) []change {
	fmt.Println("determining differences...")

	var changes []change
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

	switch b := before.(type) {
	case *ast.GenDecl:
		a := after.(*ast.GenDecl)

		if reflect.TypeOf(b.Specs[0]) != reflect.TypeOf(a.Specs[0]) {
			// Spec changed, such as ValueSpec to TypeSpec (eg var/const to struct)
			return false
		}

		switch bspec := b.Specs[0].(type) {
		case *ast.ValueSpec:
			aspec := a.Specs[0].(*ast.ValueSpec)

			// var / const
			if bspec.Type.(*ast.Ident).Name != aspec.Type.(*ast.Ident).Name {
				// type changed
				return false
			}
		case *ast.TypeSpec:
			aspec := a.Specs[0].(*ast.TypeSpec)

			// type struct/interface/aliased

			if reflect.TypeOf(bspec.Type) != reflect.TypeOf(aspec.Type) {
				// Spec change, such as from StructType to InterfaceType or different aliased types
				return false
			}

			switch btype := bspec.Type.(type) {
			case *ast.InterfaceType:
				atype := aspec.Type.(*ast.InterfaceType)

				// interfaces don't care if methods are removed, so discard those
				added, _, changed := diffFields(btype.Methods.List, atype.Methods.List)

				if len(added) > 0 || len(changed) > 0 {
					// Fields were removed or changed types
					return false
				}
			case *ast.StructType:
				atype := aspec.Type.(*ast.StructType)

				// structs don't care if fields were added, so discard those
				_, removed, changed := diffFields(btype.Fields.List, atype.Fields.List)

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
		a := after.(*ast.FuncDecl)

		if !equalFieldTypes(b.Type.Params.List, a.Type.Params.List) {
			return false
		}

		if b.Type.Results != nil {
			if a.Type.Results == nil {
				// removed return parameter
				return false
			}

			// Only check if we're changing/removing return parameters
			if !equalFieldTypes(b.Type.Results.List, a.Type.Results.List) {
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
	switch v := ident.(type) {
	case *ast.Ident:
		// perhaps a struct
		return v.Name
	case *ast.FuncType:
		// perhaps interface/func
		var params, results bytes.Buffer

		for i, list := range v.Params.List {
			if i != 0 {
				fmt.Fprint(&params, ", ")
			}
			fmt.Fprint(&params, list.Type.(*ast.Ident).Name)
		}
		if v.Results != nil {
			for i, list := range v.Results.List {
				if i != 0 {
					fmt.Fprint(&results, ", ")
				}
				fmt.Fprint(&results, list.Type.(*ast.Ident).Name)
			}
		}
		return fmt.Sprintf("(%s) (%s)", params.String(), results.String())
	default:
		panic(fmt.Errorf("Unknown decl type: %T", ident))
	}
}
