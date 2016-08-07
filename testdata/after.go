package library

import (
	"bytes"
	"errors"
	"io"
)

// ConstMultiSpec* checks support for multiple specs
const (
	ConstMultiSpecA int = 0

	ConstMultiSpecB int = 0
)

// ConstAdded detects additions
const ConstAdded int = 0

// ConstRemoved detects removals
//const ConstRemoved int = 0

// GenFuncDeclChange detects a change from a constant into a function
func GenFuncDeclChange() {}

// GenDeclSpecChange detects a change from a ValueSpec to TypeSpec
type GenDeclSpecChange struct{}

// ConstChangeType detects a change of type for a constant
const ConstChangeType uint = 0

// ValInferredType checks for support for inferred types
var ValInferredType = "string"
var ValInferredTypeBuiltIn = errors.New("some error")
var ValInferredTypePackage = bytes.NewBufferString("some error")

// ValChangeMulti detects a change in multi assignments
var _, ValChangeMultiZeroState uint
var _, ValChangeMulti = 1, false

// ValChangeType detects a change of type for a constant
var VarChangeType uint

// VarChangeValSpecType detects a change between val spec types
var VarChangeValSpecType []int

// VarChangeTypeStruct detects support for var (anonymous) struct
var VarChangeTypeStruct struct{}

// VarChangeTypeInterface detects support for var interface{}
var VarChangeTypeInterface interface{}

// VarChangeTypeChan detects changes in var chan
var VarChangeTypeChan chan uint

// VarChangeTypeChanDirection detects changes in chan direction
var VarChangeTypeChanDir <-chan int

// VarChangeTypeChanDirection detects removing chan direction
var VarChangeTypeChanDirRelax chan int

// VarChangeTypeFunc detects support for var funcs
var VarChangeTypeFunc func(arg1 int) (err error)

// VarChangeTypeFuncArgRename detects ignorance of argument name changes
var VarChangeTypeFuncArgRename func(arg2 int) (err2 error)

// VarChangeTypeFuncParam detects a change in a func's parameter list
var VarChangeTypeFuncParam func(uint) error

// VarChangeTypeFuncResult detects a change in a func's return list
var VarChangeTypeFuncResult func(int) bool

// VarAddTypeFuncResult detects an add in a func's return list
var VarAddTypeFuncResult func(int) error

// VarRemoveTypeFuncResult detects a removal in a func's return list
var VarRemoveTypeFuncResult func(int)

// VarChangeTypeSlice detects a change in a slice's type
var VarChangeTypeSlice []uint

// VarChangeTypeArrayLen detects a change between slice and array
var VarChangeTypeSliceLen [1]int

// VarChangeTypeArrayLen detects a change in an array's length
var VarChangeTypeArrayLen [2]int

// VarChangeTypeArrayType detects a change in an array's type
var VarChangeTypeArrayType [1]uint

// VarChangeTypeMapKey detects a change in a map's key
var VarChangeTypeMapKey map[uint]int

// VarChangeTypeMapValue detects a change in a map's value
var VarChangeTypeMapValue map[int]uint

// VarChangeTypeSelector detects a change in a selector.ident
var VarChangeTypeSelector bytes.Reader

// VarChangeTypeStar detects a change in a pointer
var VarChangeTypeStar *uint
var VarChangeTypeStarSelector *bytes.Reader

// TypeSpecChange detects a change between types specs
type TypeSpecChange interface{}

// StructAddMember detects additions of struct fields (is not a problem)
type StructAddMember struct {
	Member1 int
	Member2 []int
}

// StructRemMember detects removals of struct fields
type StructRemMember struct {
	//Member1 was removed
}

// StructChangeMember detects changes of struct fields
type StructChangeMember struct {
	Member1 uint
}

// IfaceAddMember detects additions of interface methods
type IfaceAddMember interface {
	Member1(arg1 int) (ret1 bool)
}

// IfaceRemMember detects removals of interface methods (is not a problem)
type IfaceRemMember interface {
	//Member1 was removed
}

// IfaceChangeArgName detects argument renaming of interface methods (is not a problem)
type IfaceChangeArgName interface {
	Member1(renamedArg1 int) (renamedArg2 bool)
}

// IfaceChangeMemberArg detects changes of interface methods arguments
type IfaceChangeMemberArg interface {
	Member1(arg1 uint) (ret1 bool)
}

// IfaceChangeMemberReturn detects changes of interface methods return params
type IfaceChangeMemberReturn interface {
	Member1(arg1 int) (ret1 int)
}

// TypeAlias detects changes to alias types
type TypeAlias uint

// FuncRetEmptyFunc tests handling of a func return bare func
func FuncRetEmptyFunc() func()

// FuncArg tests handing of function args that don't change
func FuncArg(arg1 int)                 {}
func FuncArgPtr(arg1 *int)             {}
func FuncArgPkg(arg1 bytes.Buffer)     {}
func FuncArgPtrPkg(arg1 *bytes.Buffer) {}

// FuncRenameArg tests ignorance of changes in variable names
func FuncRenameArg(arg2 int) (ret2 error) {}

// FuncAddArg detects additions of function parameter types
func FuncAddArg(arg1 int) {}

// FuncRemArg detects removals of function parameter types
func FuncRemArg() {}

// FuncChangeArg detects changes of function parameter types
func FuncChangeArg(param uint) {}

// FuncChangeChan detects changes of function channel parameter types
func FuncChangeChan(arg1 chan uint) {}

// FuncChangeChanDir detects changes of function channel parameter types direction
func FuncChangeChanDir(arg1 <-chan int) {}

// FuncChangeChanDirRelax detects relaxion of channel parameter type
func FuncChangeChanDirRelax(arg1 chan int) {}

// FuncAddRet detects additions of function return params (is not a problem)
func FuncAddRet() error { return nil }

// FuncAddRetMore detects additions of function return params
func FuncAddRetMore() (error, bool) { return nil, false }

// FuncRemRet detects removals of function return params
func FuncRemRet() {}

// FuncChangeArg detects changes of function return params
func FuncChangeRet() bool                      { return false }
func FuncChangeRetStarIdent() *uint            { return nil }
func FuncChangeRetStarSelector() *bytes.Reader { return nil }

// FuncRecv tests changes to receivers
type FuncRecv struct{}

func (_ *FuncRecv) Method1(arg1 bool) (ret1 int) { return 1 }
func (_ FuncRecv) Method2(arg1 bool) (ret1 int)  { return 1 }

// FuncAddVariadic detects addition of a variadic argument to a function (is not a problem)
func FuncAddVariadic(_ ...int) {}

// FuncChangeToVariadic detects parameter change to variadic of same type (is not a problem)
func FuncChangeToVariadic(_ ...int) {}

// FuncChangeToVariadicDiffType detects parameter change to variadic of a different type
func FuncChangeToVariadicDiffType(_ ...uint) {}

type T1 interface{}
type T2 interface {
	Error() string
}
type T3 interface {
	Member()
}

// FuncInterface tests for support of comparing interfaces (is not a problem)
func FuncInterface(_ T1) {}

// FuncInterfaceIncompatible detects changes in interfaces
func FuncInterfaceIncompatible(_ T3) {}

// FuncInterfaceCompatible detects changes between compatible interfaces (is not a problem)
func FuncInterfaceCompatible(_ T1) {}

// FuncInterfaceCompatible2 detects changes between compatible interfaces (is not a problem)
func FuncInterfaceCompatible2(_ io.Writer) {}

// FuncInterfaceCompatible3 detects changes between compatible interfaces (is not a problem)
func FuncInterfaceCompatible3(_ error) {}

type C1 int

// FuncCustomType tests for support of comparing custom types
func FuncCustomType(_ C1) {}
