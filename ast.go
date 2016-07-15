package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"reflect"
)

type changeType uint8

const (
	changeNone changeType = iota
	changeNonBreaking
	changeBreaking
)

func (c changeType) String() string {
	switch c {
	case changeNone:
		return "no change"
	case changeNonBreaking:
		return "non-breaking change"
	}
	return "breaking change"
}

type operation uint8

const (
	opAdd operation = iota
	opRemove
	opChange
)

func (op operation) String() string {
	switch op {
	case opAdd:
		return "added"
	case opRemove:
		return "removed"
	}
	return "changed"
}

// change is the ast declaration containing the before and after
type change struct {
	id         string
	summary    string
	op         operation
	changeType changeType
	before     ast.Decl
	after      ast.Decl
}

func (c change) String() string {
	fset := &token.FileSet{} // only require non-nil fset
	pcfg := printer.Config{Mode: printer.RawFormat, Indent: 1}

	buf := bytes.NewBufferString("")
	if c.op == opChange {
		fmt.Fprintf(buf, "%s (%s - %s)\n", c.op, c.changeType, c.summary)
	} else {
		fmt.Fprintln(buf, c.op)
	}

	if c.before != nil {
		pcfg.Fprint(buf, fset, c.before)
		fmt.Fprintln(buf)
	}
	if c.after != nil {
		pcfg.Fprint(buf, fset, c.after)
		fmt.Fprintln(buf)
	}
	return buf.String()
}

// byID implements sort.Interface for []change based on the id field
type byID []change

func (a byID) Len() int           { return len(a) }
func (a byID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byID) Less(i, j int) bool { return a[i].id < a[j].id }

// decls is a map of an identifier to actual ast, where the id is a unique
// name to match declarations for before and after
type decls map[string]ast.Decl

func diff(bdecls, adecls decls) []change {
	fmt.Println("determining differences...")

	var changes []change
	for id, decl := range bdecls {
		if _, ok := adecls[id]; !ok {
			// in before, not in after, therefore it was removed
			changes = append(changes, change{id: id, op: opRemove, before: decl})
			continue
		}

		// in before and in after, check if there's a difference
		changeType, summary := compareDecl(bdecls[id], adecls[id])
		if changeType == changeNone {
			continue
		}

		changes = append(changes, change{
			id:         id,
			op:         opChange,
			changeType: changeType,
			summary:    summary,
			before:     decl,
			after:      adecls[id]},
		)
	}

	for id, decl := range adecls {
		if _, ok := bdecls[id]; !ok {
			// in after, not in before, therefore it was added
			changes = append(changes, change{id: id, op: opAdd, after: decl})
		}
	}

	return changes
}

// equal compares two declarations and returns true if they do not have
// incompatible changes. For example, comments aren't compared, names of
// arguments aren't compared etc.
func compareDecl(before, after ast.Decl) (changeType, string) {
	// compare types, ignore comments etc, so reflect.DeepEqual isn't good enough

	if reflect.TypeOf(before) != reflect.TypeOf(after) {
		// Declaration type changed, such as GenDecl to FuncDecl (eg var/const to func)
		return changeBreaking, "changed declaration"
	}

	switch b := before.(type) {
	case *ast.GenDecl:
		a := after.(*ast.GenDecl)

		if reflect.TypeOf(b.Specs[0]) != reflect.TypeOf(a.Specs[0]) {
			// Spec changed, such as ValueSpec to TypeSpec (eg var/const to struct)
			return changeBreaking, "changed spec"
		}

		switch bspec := b.Specs[0].(type) {
		case *ast.ValueSpec:
			aspec := a.Specs[0].(*ast.ValueSpec)

			// var / const
			if bspec.Type.(*ast.Ident).Name != aspec.Type.(*ast.Ident).Name {
				// type changed
				return changeBreaking, "changed type"
			}
		case *ast.TypeSpec:
			aspec := a.Specs[0].(*ast.TypeSpec)

			// type struct/interface/aliased

			if reflect.TypeOf(bspec.Type) != reflect.TypeOf(aspec.Type) {
				// Spec change, such as from StructType to InterfaceType or different aliased types
				return changeBreaking, "changed type of value spec"
			}

			switch btype := bspec.Type.(type) {
			case *ast.InterfaceType:
				atype := aspec.Type.(*ast.InterfaceType)

				// interfaces don't care if methods are removed
				added, removed, changed := diffFields(btype.Methods.List, atype.Methods.List)
				if len(added) > 0 {
					// Fields were added
					return changeBreaking, "members added"
				} else if len(changed) > 0 {
					// Fields changed types
					return changeBreaking, "members changed types"
				} else if len(removed) > 0 {
					return changeNonBreaking, "members removed"
				}
			case *ast.StructType:
				atype := aspec.Type.(*ast.StructType)

				// structs don't care if fields were added
				added, removed, changed := diffFields(btype.Fields.List, atype.Fields.List)
				if len(removed) > 0 {
					// Fields were removed
					return changeBreaking, "members removed"
				} else if len(changed) > 0 {
					// Fields changed types
					return changeBreaking, "members changed types"
				} else if len(added) > 0 {
					return changeNonBreaking, "members added"
				}
			case *ast.Ident:
				// alias
				panic("not yet implemented")
			}
		}
	case *ast.FuncDecl:
		a := after.(*ast.FuncDecl)
		if !equalFieldTypes(b.Type.Params.List, a.Type.Params.List) {
			return changeBreaking, "parameters types changed"
		}

		if b.Type.Results != nil {
			if a.Type.Results == nil {
				// removed return parameter
				return changeBreaking, "removed return parameter"
			}

			// Only check if we're changing/removing return parameters
			if !equalFieldTypes(b.Type.Results.List, a.Type.Results.List) {
				return changeBreaking, "changed or removed return parameter"
			}
		}
	default:
		panic(fmt.Errorf("Unknown type: %T", before))
	}
	return changeNone, ""
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
