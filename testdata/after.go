package library

import (
	"bytes"
	"errors"
	tmpl "html/template"
	"io"
	tmplY "text/template"
)

func init() {
	_ = 1
}

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

// AliasedImport checks for support for aliases imports
var AliasedImportChange tmpl.Template
var AliasedImportRename tmplY.Template

type AliasedImportChangeS struct{ T tmpl.Template }
type AliasedImportRenameS struct{ T tmplY.Template }

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

// VarChangeTypeFuncInferredtests for ignorance of shorthand type syntax
var VarChangeTypeFuncInferred func(arg1, arg2 int) (ret1, ret2 bool)

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

// Struct checks for support of struct fields
type Struct struct{ A int }
type StructPtr struct{ A *int }
type StructPkg struct{ A bytes.Buffer }
type StructPtrPkg struct{ A *bytes.Buffer }
type StructMapPkg struct{ A map[int]bytes.Buffer }
type StructMapPtrPkg struct{ A map[int]*bytes.Buffer }

// StructAddMember detects additions of struct fields (is not a problem)
type StructAddMember struct {
	Member1 int
	Member2 []int
}

// StructEmbedAddMember detects additions of struct fields with embedded fields (is not a problem)
type StructEmbedAddMember struct {
	Member1 int
	Struct
	*StructPtr
	bytes.Buffer
	*bytes.Reader
}

// StructRemMember detects removals of struct fields
type StructRemMember struct {
	//Member1 was removed
}

// StructRemEmbed detects removals of embedded struct fields
type StructRemEmbed struct {
	//Struct was removed
}

//type structPriv struct{} removed

// StructRemPrivEmbed tests for ignorance in removal of elds
type StructRemPrivEmbed struct {
	//structPriv was removed
}

// StructChangeMember detects changes of struct fields
type StructChangeMember struct {
	Member1 uint
}

// StructInferredMember checks for support of shorthand types
type StructChangeInferredMember struct {
	Member1, Member2 int
}

// StructRemPrivMember tests for ignorance in removal of private members
type StructRemPrivMember struct {
	// private1 was removed
	// private2 was removed
	Public int
	// private3 was removed
}

// StructChangePrivMember tests for ignorance in changes in private members
type StructChangePrivMember struct {
	private uint
}

// IfaceEmbed checks for support of interfaces with embedded values
type IfaceEmbed interface {
	io.Reader
}

// IfaceEmbedResolve tests for ignorance of embedded to non embedded of same signature
type IfaceEmbedResolve interface {
	Read(p []byte) (n int, err error)
}

// IfaceEmbedCompact tests for ignorance of non embedded to embedded of same signature
type IfaceEmbedCompact interface {
	io.Reader
}

// IfaceInferred tests for ignorance of shorthand type syntax
type IfaceInferred interface {
	Member1(arg1, arg2 int) (ret1, ret2 bool)
}

// IfaceAddMember detects additions of interface methods
type IfaceAddMember interface {
	Member1(arg1 int) (ret1 bool)
}

// IfaceRemMember detects removals of interface methods
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
func FuncArg(arg1 int)                    {}
func FuncArgPtr(arg1 *int)                {}
func FuncArgPkg(arg1 bytes.Buffer)        {}
func FuncArgPtrPkg(arg1 *bytes.Buffer)    {}
func FuncArgFuncPkg(func(A bytes.Buffer)) {}

// FuncInferred tests for ignorance of shorthand type syntax
func FuncInferred(arg1, arg2 int) (ret1, ret2 bool) {}

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
func (_ FuncRecv) method3(arg1 bool) (ret1 int)  { return 1 }

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

// PrivateReturned detects changes in unexported, but returned types
type s struct{ Member uint }

func F1() s       { return s{} }
func F2() *s      { return &s{} }
func (s) F() uint { return 0 }
