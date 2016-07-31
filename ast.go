package abicheck

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
	"strconv"
)

const (
	Unknown     = "unknown change"
	None        = "no change"
	NonBreaking = "non-breaking change"
	Breaking    = "breaking change"
)

type DeclChange struct {
	Change string
	Msg    string
}

type DeclChecker struct {
	btypes *types.Checker
	atypes *types.Checker
}

func NewDeclChecker(btypes, atypes *types.Checker) *DeclChecker {
	return &DeclChecker{
		btypes: btypes,
		atypes: atypes,
	}
}

func nonBreaking(msg string) (*DeclChange, error) { return &DeclChange{NonBreaking, msg}, nil }
func breaking(msg string) (*DeclChange, error)    { return &DeclChange{Breaking, msg}, nil }
func unknown(msg string) (*DeclChange, error)     { return &DeclChange{Unknown, msg}, nil }
func none() (*DeclChange, error)                  { return &DeclChange{None, ""}, nil }

// equal compares two declarations and returns true if they do not have
// incompatible changes. For example, comments aren't compared, names of
// arguments aren't compared etc.
func (c DeclChecker) Check(before, after ast.Decl) (*DeclChange, error) {
	// compare types, ignore comments etc, so reflect.DeepEqual isn't good enough

	if reflect.TypeOf(before) != reflect.TypeOf(after) {
		// Declaration type changed, such as GenDecl to FuncDecl (eg var/const to func)
		return breaking("changed declaration")
	}

	switch b := before.(type) {
	case *ast.GenDecl:
		a := after.(*ast.GenDecl)

		// getDecls flattened var/const blocks, so .Specs should contain just 1

		if reflect.TypeOf(b.Specs[0]) != reflect.TypeOf(a.Specs[0]) {
			// Spec changed, such as ValueSpec to TypeSpec (eg var/const to struct)
			return breaking("changed spec")
		}

		switch bspec := b.Specs[0].(type) {
		case *ast.ValueSpec:
			aspec := a.Specs[0].(*ast.ValueSpec)

			if bspec.Type == nil || aspec.Type == nil {
				// eg: var ErrSomeError = errors.New("Some Error")
				// cannot currently determine the type

				return unknown("cannot currently determine type")
			}

			if reflect.TypeOf(bspec.Type) != reflect.TypeOf(aspec.Type) {
				// eg change from int to []int
				return breaking("changed value spec type")
			}

			// var / const
			switch btype := bspec.Type.(type) {
			case *ast.Ident, *ast.SelectorExpr, *ast.StarExpr:
				// int/string/etc or bytes.Buffer/etc or *int/*bytes.Buffer/etc
				if c.btypes.TypeOf(bspec.Type) != c.atypes.TypeOf(aspec.Type) {
					// type changed
					return breaking("changed type")
				}
			case *ast.ArrayType:
				// slice/array
				atype := aspec.Type.(*ast.ArrayType)
				if !c.exprEqual(btype, atype) {
					// change of length, or between array and slice or type
					return breaking("changed array/slice's length or type")
				}
			case *ast.MapType:
				// map
				atype := aspec.Type.(*ast.MapType)

				if !c.exprEqual(btype.Key, atype.Key) {
					return breaking("changed map's key's type")
				}
				if !c.exprEqual(btype.Value, atype.Value) {
					return breaking("changed map's value's type")
				}
			case *ast.InterfaceType:
				// this is a special case for just interface{}
				atype := aspec.Type.(*ast.InterfaceType)
				return c.checkInterface(btype, atype)
			case *ast.ChanType:
				// channel
				atype := aspec.Type.(*ast.ChanType)
				return c.checkChan(btype, atype)
			case *ast.FuncType:
				// func
				atype := aspec.Type.(*ast.FuncType)
				return c.checkFunc(btype, atype)
			case *ast.StructType:
				// anonymous struct
				atype := aspec.Type.(*ast.StructType)
				return c.checkStruct(btype, atype)
			default:
				return nil, fmt.Errorf("unknown val spec type: %T", btype)
			}
		case *ast.TypeSpec:
			aspec := a.Specs[0].(*ast.TypeSpec)

			// type struct/interface/aliased

			if reflect.TypeOf(bspec.Type) != reflect.TypeOf(aspec.Type) {
				// Spec change, such as from StructType to InterfaceType or different aliased types
				return breaking("changed type of value spec")
			}

			switch btype := bspec.Type.(type) {
			case *ast.InterfaceType:
				atype := aspec.Type.(*ast.InterfaceType)
				return c.checkInterface(btype, atype)
			case *ast.StructType:
				atype := aspec.Type.(*ast.StructType)
				return c.checkStruct(btype, atype)
			case *ast.Ident:
				// alias
				atype := aspec.Type.(*ast.Ident)
				if btype.Name != atype.Name {
					// Alias typing changed underlying types
					return breaking("alias changed its underlying type")
				}
			}
		}
	case *ast.FuncDecl:
		a := after.(*ast.FuncDecl)
		return c.checkFunc(b.Type, a.Type)
	default:
		return nil, fmt.Errorf("unknown declaration type: %T", before)
	}
	return none()
}

func (c DeclChecker) checkChan(before, after *ast.ChanType) (*DeclChange, error) {
	if !c.exprEqual(before.Value, after.Value) {
		return breaking("changed channel's type")
	}

	// If we're specifying a direction and it's not the same as before
	// (if we remove direction then that change isn't breaking)
	if before.Dir != after.Dir {
		if after.Dir != ast.SEND && after.Dir != ast.RECV {
			return nonBreaking("removed channel's direction")
		}
		return breaking("changed channel's direction")
	}
	return none()
}

func (c DeclChecker) checkInterface(before, after *ast.InterfaceType) (*DeclChange, error) {
	// interfaces don't care if methods are removed
	added, removed, changed := c.diffFields(before.Methods.List, after.Methods.List)
	if len(added) > 0 {
		// Fields were added
		return breaking("members added")
	} else if len(changed) > 0 {
		// Fields changed types
		return breaking("members changed types")
	} else if len(removed) > 0 {
		return nonBreaking("members removed")
	}

	return none()
}

func (c DeclChecker) checkStruct(before, after *ast.StructType) (*DeclChange, error) {
	// structs don't care if fields were added
	added, removed, changed := c.diffFields(before.Fields.List, after.Fields.List)
	if len(removed) > 0 {
		// Fields were removed
		return breaking("members removed")
	} else if len(changed) > 0 {
		// Fields changed types
		return breaking("members changed types")
	} else if len(added) > 0 {
		return nonBreaking("members added")
	}
	return none()
}

func (c DeclChecker) checkFunc(before, after *ast.FuncType) (*DeclChange, error) {
	// don't compare argument names
	bparams := stripNames(before.Params.List)
	aparams := stripNames(after.Params.List)

	added, removed, changed := c.diffFields(bparams, aparams)
	if len(added) > 0 || len(removed) > 0 || len(changed) > 0 {
		return breaking("parameters types changed")
	}

	if before.Results != nil {
		if after.Results == nil {
			// removed return parameter
			return breaking("removed return parameter")
		}

		// don't compare argument names
		bresults := stripNames(before.Results.List)
		aresults := stripNames(after.Results.List)

		// Adding return parameters to a function, when it didn't have any before is
		// ok, so only check if for breaking changes if there was parameters before
		if len(before.Results.List) > 0 {
			added, removed, changed := c.diffFields(bresults, aresults)
			if len(added) > 0 || len(removed) > 0 || len(changed) > 0 {
				return breaking("return parameters changed")
			}
		}
	}

	return none()
}

// stripNames strips the names from a fieldlist, which is usually a function's
// (or method's) parameter or results list, these are internal to the function.
// This returns a good-enough copy of the field list, but isn't a complete copy
// as some pointers remain, but no other modifications are made, so it's ok.
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

func (c DeclChecker) diffFields(before, after []*ast.Field) (added, removed, changed []*ast.Field) {
	// Presort after for quicker matching of fieldname -> type, may not be worthwhile
	AfterMembers := make(map[string]*ast.Field)
	for i, field := range after {
		AfterMembers[fieldKey(field, i)] = field
	}

	for i, bfield := range before {
		bkey := fieldKey(bfield, i)
		if afield, ok := AfterMembers[bkey]; ok {
			if !c.exprEqual(bfield.Type, afield.Type) {
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
func (c DeclChecker) exprEqual(before, after ast.Expr) bool {
	if reflect.TypeOf(before) != reflect.TypeOf(after) {
		return false
	}

	switch btype := before.(type) {
	case *ast.ChanType:
		atype := after.(*ast.ChanType)
		change, _ := c.checkChan(btype, atype)
		return change.Change != Breaking
	}

	return types.Identical(c.btypes.TypeOf(before), c.atypes.TypeOf(after))
}
