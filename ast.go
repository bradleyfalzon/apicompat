package abicheck

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"reflect"
	"strconv"
)

type ChangeType uint8

const (
	ChangeError ChangeType = iota
	ChangeUnknown
	ChangNone
	ChangeNonBreaking
	ChangeBreaking
)

func (c ChangeType) String() string {
	switch c {
	case ChangeError:
		return "parse error"
	case ChangeUnknown:
		return "unknowable"
	case ChangNone:
		return "no change"
	case ChangeNonBreaking:
		return "non-breaking change"
	case ChangeBreaking:
		return "breaking change"
	}
	panic(fmt.Sprintf("unknown ChangeType: %d", c))
}

type OpType uint8

const (
	OpAdd OpType = iota
	OpRemove
	OpChange
)

func (op OpType) String() string {
	switch op {
	case OpAdd:
		return "added"
	case OpRemove:
		return "removed"
	case OpChange:
		return "changed"
	}
	panic(fmt.Sprintf("unknown operation type: %d", op))
}

// change is the ast declaration containing the before and after
type Change struct {
	Pkg     string
	ID      string
	Summary string
	Op      OpType
	Change  ChangeType
	Before  ast.Decl
	After   ast.Decl
}

func (c Change) String() string {
	fset := token.FileSet{} // only require non-nil fset
	pcfg := printer.Config{Mode: printer.RawFormat, Indent: 1}
	buf := bytes.Buffer{}

	if c.Op == OpChange {
		fmt.Fprintf(&buf, "%s (%s - %s)\n", c.Op, c.Change, c.Summary)
	} else {
		fmt.Fprintln(&buf, c.Op)
	}

	if c.Before != nil {
		pcfg.Fprint(&buf, &fset, c.Before)
		fmt.Fprintln(&buf)
	}
	if c.After != nil {
		pcfg.Fprint(&buf, &fset, c.After)
		fmt.Fprintln(&buf)
	}
	return buf.String()
}

// byID implements sort.Interface for []change based on the id field
type byID []Change

func (a byID) Len() int           { return len(a) }
func (a byID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byID) Less(i, j int) bool { return a[i].ID < a[j].ID }

// revDecls is a map between a package to an id to ast.Decl, where the id is
// a unique name to match declarations for before and after
type revDecls map[string]map[string]ast.Decl

type diffError struct {
	summary string
	bdecl,
	adecl ast.Decl
	bpos,
	apos token.Pos
}

func (e diffError) Error() string {
	return e.summary
}

func compareRevs(bRevDecls, aRevDecls revDecls) (error, []Change) {
	var changes []Change

	for pkg, bDecls := range bRevDecls {
		aDecls, ok := aRevDecls[pkg]
		if !ok {
			continue
		}

		for id, bDecl := range bDecls {
			aDecl, ok := aDecls[id]
			if !ok {
				// in before, not in after, therefore it was removed
				changes = append(changes, Change{Pkg: pkg, ID: id, Op: OpRemove, Before: bDecl})
				continue
			}

			// in before and in after, check if there's a difference
			changeType, summary := compareDecl(bDecl, aDecl)

			switch changeType {
			case ChangNone, ChangeUnknown:
				continue
			case ChangeError:
				err := &diffError{summary: summary, bdecl: bDecl, adecl: aDecl}
				return err, changes
			}

			changes = append(changes, Change{
				Pkg:     pkg,
				ID:      id,
				Op:      OpChange,
				Change:  changeType,
				Summary: summary,
				Before:  bDecl,
				After:   aDecl,
			})
		}

		for id, aDecl := range aDecls {
			if _, ok := bDecls[id]; !ok {
				// in after, not in before, therefore it was added
				changes = append(changes, Change{Pkg: pkg, ID: id, Op: OpAdd, After: aDecl})
			}
		}
	}

	return nil, changes
}

// equal compares two declarations and returns true if they do not have
// incompatible changes. For example, comments aren't compared, names of
// arguments aren't compared etc.
func compareDecl(before, after ast.Decl) (ChangeType, string) {
	// compare types, ignore comments etc, so reflect.DeepEqual isn't good enough

	if reflect.TypeOf(before) != reflect.TypeOf(after) {
		// Declaration type changed, such as GenDecl to FuncDecl (eg var/const to func)
		return ChangeBreaking, "changed declaration"
	}

	switch b := before.(type) {
	case *ast.GenDecl:
		a := after.(*ast.GenDecl)

		// getDecls flattened var/const blocks, so .Specs should contain just 1

		if reflect.TypeOf(b.Specs[0]) != reflect.TypeOf(a.Specs[0]) {
			// Spec changed, such as ValueSpec to TypeSpec (eg var/const to struct)
			return ChangeBreaking, "changed spec"
		}

		switch bspec := b.Specs[0].(type) {
		case *ast.ValueSpec:
			aspec := a.Specs[0].(*ast.ValueSpec)

			if bspec.Type == nil || aspec.Type == nil {
				// eg: var ErrSomeError = errors.New("Some Error")
				// cannot currently determine the type
				return ChangeUnknown, "cannot currently determine type"
			}

			// TODO perhaps just make this entire thing use
			// exprEqual(bspec.Type, aspect.Type) but we'll lose some details

			if reflect.TypeOf(bspec.Type) != reflect.TypeOf(aspec.Type) {
				// eg change from int to []int
				return ChangeBreaking, "changed value spec type"
			}

			// var / const
			switch btype := bspec.Type.(type) {
			case *ast.Ident, *ast.SelectorExpr, *ast.StarExpr:
				// int/string/etc or bytes.Buffer/etc or *int/*bytes.Buffer/etc
				if !exprEqual(bspec.Type, aspec.Type) {
					// type changed
					return ChangeBreaking, "changed type"
				}
			case *ast.ArrayType:
				// slice/array
				atype := aspec.Type.(*ast.ArrayType)
				// compare length
				if !exprEqual(btype.Len, atype.Len) {
					// change of length, or between array and slice
					return ChangeBreaking, "changed of array's length"
				}
				// compare array's element's type
				if !exprEqual(btype.Elt, atype.Elt) {
					return ChangeBreaking, "changed of array's element's type"
				}
			case *ast.MapType:
				// map
				atype := aspec.Type.(*ast.MapType)

				if !exprEqual(btype.Key, atype.Key) {
					return ChangeBreaking, "changed map's key's type"
				}
				if !exprEqual(btype.Value, atype.Value) {
					return ChangeBreaking, "changed map's value's type"
				}
			case *ast.InterfaceType:
				// this is a special case for just interface{}
				atype := aspec.Type.(*ast.InterfaceType)
				return compareInterfaceType(btype, atype)
			case *ast.ChanType:
				// channel
				atype := aspec.Type.(*ast.ChanType)
				return compareChanType(btype, atype)
			case *ast.FuncType:
				// func
				atype := aspec.Type.(*ast.FuncType)
				return compareFuncType(btype, atype)
			case *ast.StructType:
				// anonymous struct
				atype := aspec.Type.(*ast.StructType)
				return compareStructType(btype, atype)
			default:
				return ChangeError, fmt.Sprintf("Unknown val spec type: %T, source: %s", btype, typeToString(before))
			}
		case *ast.TypeSpec:
			aspec := a.Specs[0].(*ast.TypeSpec)

			// type struct/interface/aliased

			if reflect.TypeOf(bspec.Type) != reflect.TypeOf(aspec.Type) {
				// Spec change, such as from StructType to InterfaceType or different aliased types
				return ChangeBreaking, "changed type of value spec"
			}

			switch btype := bspec.Type.(type) {
			case *ast.InterfaceType:
				atype := aspec.Type.(*ast.InterfaceType)
				return compareInterfaceType(btype, atype)
			case *ast.StructType:
				atype := aspec.Type.(*ast.StructType)
				return compareStructType(btype, atype)
			case *ast.Ident:
				// alias
				atype := aspec.Type.(*ast.Ident)
				if btype.Name != atype.Name {
					// Alias typing changed underlying types
					return ChangeBreaking, "alias changed its underlying type"
				}
			}
		}
	case *ast.FuncDecl:
		a := after.(*ast.FuncDecl)
		return compareFuncType(b.Type, a.Type)
	default:
		return ChangeError, fmt.Sprintf("Unknown declaration type: %T, source: %s", before, typeToString(before))
	}
	return ChangNone, ""
}

func compareChanType(before, after *ast.ChanType) (ChangeType, string) {
	if !exprEqual(before.Value, after.Value) {
		return ChangeBreaking, "changed channel's type"
	}

	// If we're specifying a direction and it's not the same as before
	// (if we remove direction then that change isn't breaking)
	if before.Dir != after.Dir {
		if after.Dir != ast.SEND && after.Dir != ast.RECV {
			return ChangeNonBreaking, "removed channel's direction"
		}
		return ChangeBreaking, "changed channel's direction"
	}
	return ChangNone, ""
}

func compareInterfaceType(before, after *ast.InterfaceType) (ChangeType, string) {
	// interfaces don't care if methods are removed
	added, removed, changed := diffFields(before.Methods.List, after.Methods.List)
	if len(added) > 0 {
		// Fields were added
		return ChangeBreaking, "members added"
	} else if len(changed) > 0 {
		// Fields changed types
		return ChangeBreaking, "members changed types"
	} else if len(removed) > 0 {
		return ChangeNonBreaking, "members removed"
	}

	return ChangNone, ""
}
func compareStructType(before, after *ast.StructType) (ChangeType, string) {
	// structs don't care if fields were added
	added, removed, changed := diffFields(before.Fields.List, after.Fields.List)
	if len(removed) > 0 {
		// Fields were removed
		return ChangeBreaking, "members removed"
	} else if len(changed) > 0 {
		// Fields changed types
		return ChangeBreaking, "members changed types"
	} else if len(added) > 0 {
		return ChangeNonBreaking, "members added"
	}
	return ChangNone, ""
}
func compareFuncType(before, after *ast.FuncType) (ChangeType, string) {
	// don't compare argument names
	bparams := stripNames(before.Params.List)
	aparams := stripNames(after.Params.List)

	added, removed, changed := diffFields(bparams, aparams)
	if len(added) > 0 || len(removed) > 0 || len(changed) > 0 {
		return ChangeBreaking, "parameters types changed"
	}

	if before.Results != nil {
		if after.Results == nil {
			// removed return parameter
			return ChangeBreaking, "removed return parameter"
		}

		// don't compare argument names
		bresults := stripNames(before.Results.List)
		aresults := stripNames(after.Results.List)

		// Adding return parameters to a function, when it didn't have any before is
		// ok, so only check if for breaking changes if there was parameters before
		if len(before.Results.List) > 0 {
			added, removed, changed := diffFields(bresults, aresults)
			if len(added) > 0 || len(removed) > 0 || len(changed) > 0 {
				return ChangeBreaking, "return parameters changed"
			}
		}
	}

	return ChangNone, ""
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
	if reflect.TypeOf(before) != reflect.TypeOf(after) {
		return false
	}

	switch btype := before.(type) {
	case *ast.ChanType:
		atype := after.(*ast.ChanType)
		change, _ := compareChanType(btype, atype)
		return change != ChangeBreaking
	}

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
func typeToString(ident interface{}) string {
	fset := token.FileSet{} // only require non-nil fset
	// TODO do i need to use the printer? ast has print functions, do they just wrap this?
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
