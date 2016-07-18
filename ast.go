package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"reflect"
	"strconv"
)

type changeType uint8

const (
	changeUnknown changeType = iota
	changeNone
	changeNonBreaking
	changeBreaking
)

func (c changeType) String() string {
	switch c {
	case changeUnknown:
		return "unknowable"
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
	fset := token.FileSet{} // only require non-nil fset
	pcfg := printer.Config{Mode: printer.RawFormat, Indent: 1}
	buf := bytes.Buffer{}

	if c.op == opChange {
		fmt.Fprintf(&buf, "%s (%s - %s)\n", c.op, c.changeType, c.summary)
	} else {
		fmt.Fprintln(&buf, c.op)
	}

	if c.before != nil {
		pcfg.Fprint(&buf, &fset, c.before)
		fmt.Fprintln(&buf)
	}
	if c.after != nil {
		pcfg.Fprint(&buf, &fset, c.after)
		fmt.Fprintln(&buf)
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
	var changes []change
	for id, decl := range bdecls {
		if _, ok := adecls[id]; !ok {
			// in before, not in after, therefore it was removed
			changes = append(changes, change{id: id, op: opRemove, before: decl})
			continue
		}

		// in before and in after, check if there's a difference
		changeType, summary := compareDecl(bdecls[id], adecls[id])
		if changeType == changeNone || changeType == changeUnknown {
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

		if len(b.Specs) != 1 {
			// getDecls should've already flattened var/const blocks
			panic("unexpected number of specs")
		}

		switch bspec := b.Specs[0].(type) {
		case *ast.ValueSpec:
			aspec := a.Specs[0].(*ast.ValueSpec)

			if bspec.Type == nil || aspec.Type == nil {
				// eg: var ErrSomeError = errors.New("Some Error")
				// cannot currently determine the type
				return changeUnknown, "cannot currently determine type"
			}

			// TODO perhaps just make this entire thing use
			// exprEqual(bspec.Type, aspect.Type) but we'll lose some details

			if reflect.TypeOf(bspec.Type) != reflect.TypeOf(aspec.Type) {
				// eg change from int to []int
				return changeBreaking, "changed value spec type"
			}

			// var / const
			switch btype := bspec.Type.(type) {
			case *ast.Ident, *ast.SelectorExpr, *ast.StarExpr:
				// int/string/etc or bytes.Buffer/etc or *int/*bytes.Buffer/etc
				if !exprEqual(bspec.Type, aspec.Type) {
					// type changed
					return changeBreaking, "changed type"
				}
			case *ast.ArrayType:
				// slice/array
				atype := aspec.Type.(*ast.ArrayType)
				// compare length
				if !exprEqual(btype.Len, atype.Len) {
					// change of length, or between array and slice
					return changeBreaking, "changed of array's length"
				}
				// compare array's element's type
				if !exprEqual(btype.Elt, atype.Elt) {
					return changeBreaking, "changed of array's element's type"
				}
			case *ast.MapType:
				// map
				atype := aspec.Type.(*ast.MapType)

				if !exprEqual(btype.Key, atype.Key) {
					return changeBreaking, "changed map's key's type"
				}
				if !exprEqual(btype.Value, atype.Value) {
					return changeBreaking, "changed map's value's type"
				}
			default:
				panic(fmt.Errorf("Unknown val spec type: %T, source: %s", btype, astString(btype)))
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
				atype := aspec.Type.(*ast.Ident)
				if btype.Name != atype.Name {
					// Alias typing changed underlying types
					return changeBreaking, "alias changed its underlying type"
				}
			}
		}
	case *ast.FuncDecl:
		a := after.(*ast.FuncDecl)

		// don't compare argument names
		bparams := stripNames(b.Type.Params.List)
		aparams := stripNames(a.Type.Params.List)

		added, removed, changed := diffFields(bparams, aparams)
		if len(added) > 0 || len(removed) > 0 || len(changed) > 0 {
			return changeBreaking, "parameters types changed"
		}

		if b.Type.Results != nil {
			if a.Type.Results == nil {
				// removed return parameter
				return changeBreaking, "removed return parameter"
			}

			// don't compare argument names
			bresults := stripNames(b.Type.Results.List)
			aresults := stripNames(a.Type.Results.List)

			_, removed, changed := diffFields(bresults, aresults)
			// Only check if we're changing/removing return parameters
			if len(removed) > 0 || len(changed) > 0 {
				return changeBreaking, "changed or removed return parameter"
			}
		}
	default:
		panic(fmt.Errorf("Unknown type: %T", before))
	}
	return changeNone, ""
}

// stripNames strips the names from a fieldlist, which is usually a function's
// (or method's) parameter or results list, these are internal to the function.
// This returns a good-enough copy of the field list, but isn't a complete copy.
func stripNames(fields []*ast.Field) []*ast.Field {
	stripped := make([]*ast.Field, 0, len(fields))
	for _, f := range fields {
		stripped = append(stripped, &ast.Field{
			Doc:     f.Doc,
			Names:   nil, // nil the names
			Type:    f.Type,
			Tag:     f.Tag,
			Comment: f.Comment,
		})
	}
	return stripped
}

func diffFields(before, after []*ast.Field) (added, removed, changed []*ast.Field) {
	// Presort after for quicker matching of fieldname -> type, may not be worthwhile
	AfterMembers := make(map[string]*ast.Field)
	for i, field := range after {
		AfterMembers[fieldKey(field, i)] = field
	}

	for i, bfield := range before {
		bkey := fieldKey(bfield, i)
		if afield, ok := AfterMembers[bkey]; ok {
			if !exprEqual(bfield.Type, afield.Type) {
				// changed
				changed = append(changed, bfield)
			}
			delete(AfterMembers, bkey)
			continue
		}

		// Removed
		removed = append(removed, bfield)
	}

	// What's left in afterMembers has added
	for _, afield := range AfterMembers {
		added = append(added, afield)
	}

	return added, removed, changed
}

// Return an appropriate identifier for a field, if it has an ident (name)
// such as in the case of a struct/interface member, else, use it's provided
// position i, such as the case of a function's parameter or result list
func fieldKey(field *ast.Field, i int) string {
	if len(field.Names) > 0 {
		return field.Names[0].Name
	}
	// No name, probably a function, return position
	return strconv.FormatInt(int64(i), 10)
}

// exprEqual compares two ast.Expr to determine if they are equal
func exprEqual(before, after ast.Expr) bool {
	// For the moment just use typeToString and compare strings
	return typeToString(before) == typeToString(after)
}

// typeToString returns a type, such as ident or function and returns a string
// representation (without superfluous variable names when necessary).
//
// This is designed to make comparisons simpler by not having to handle all
// the various ast permutations, but this is the slowest method and may have
// its own set of undesirable properties (including a performance penalty).
// See the equivalent func equalFieldTypes in b3b41cc470d4258b38372b87f22d87845ecfecb6
// for an example of what it might have been (it was missing some checks though)
func typeToString(ident ast.Expr) string {
	fset := token.FileSet{} // only require non-nil fset
	pcfg := printer.Config{Mode: printer.RawFormat}
	buf := bytes.Buffer{}

	switch v := ident.(type) {
	case *ast.FuncType:
		// strip variable names in methods
		v.Params.List = stripNames(v.Params.List)
		if v.Results != nil {
			v.Results.List = stripNames(v.Results.List)
		}
	}
	pcfg.Fprint(&buf, &fset, ident)

	return buf.String()
}

// astString is a debug helper to return the go source of an ast
func astString(ast interface{}) string {
	pcfg := printer.Config{Mode: printer.RawFormat}
	buf := bytes.Buffer{}
	pcfg.Fprint(&buf, &token.FileSet{}, ast)
	return buf.String()
}
