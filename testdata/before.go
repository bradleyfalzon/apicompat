package library

import (
	"bytes"
	"errors"
	"io"
	tmpl "text/template"
	tmplX "text/template"
)

func init() {
	_ = 1
}

// ConstMultiSpec* checks support for multiple specs
const (
	ConstMultiSpecA int = 0

//ConstMultiSpecB int = 0 // will be added
)

// ConstAdded detects something being added
//const ConstAdded int = 0 // will be added

// ConstRemoved detects removals
const ConstRemoved int = 0

// GenFuncDeclChange detects a change from a constant into a function
const GenFuncDeclChange int = 1

// GenDeclSpecChange detects a change from a ValueSpec to TypeSpec
const GenDeclSpecChange int = 1

// ConstChangeType detects a change of type for a constant
const ConstChangeType int = 0

// AliasedImport checks for support for aliases imports
var AliasedImportChange tmpl.Template
var AliasedImportRename tmplX.Template

type AliasedImportChangeS struct{ T tmpl.Template }
type AliasedImportRenameS struct{ T tmplX.Template }

// ValInferredType checks for support for inferred types
var ValInferredType = "string"
var ValInferredTypeBuiltIn = errors.New("some error")
var ValInferredTypePackage = bytes.NewBufferString("some error")

// ValChangeMulti detects a change in multi assignments
var _, ValChangeMultiZeroState int
var _, ValChangeMulti = 1, 1

// ValChangeType detects a change of type for a constant
var VarChangeType int

// VarChangeValSpecType detects a change between val spec types
var VarChangeValSpecType int

// VarChangeTypeStruct detects support for var (anonymous) struct
var VarChangeTypeStruct struct{}

// VarChangeTypeInterface detects support for var interface{}
var VarChangeTypeInterface interface{}

// VarChangeTypeChan detects changes in var chan
var VarChangeTypeChan chan int

// VarChangeTypeChanDirection detects changes in chan direction
var VarChangeTypeChanDir chan int

// VarChangeTypeChanDirection detects removing chan direction
var VarChangeTypeChanDirRelax <-chan int

// VarChangeTypeFunc detects support for var funcs
var VarChangeTypeFunc func(arg1 int) (err error)

// VarChangeTypeFuncInferredtests for ignorance of shorthand type syntax
var VarChangeTypeFuncInferred func(arg1 int, arg2 int) (ret1 bool, ret2 bool)

// VarChangeTypeFuncArgRename detects ignorance of argument name changes
var VarChangeTypeFuncArgRename func(arg1 int) (err1 error)

// VarChangeTypeFuncParam detects a change in a func'ss parameter list
var VarChangeTypeFuncParam func(int) error

// VarChangeTypeFuncResult detects a change in a func's return list
var VarChangeTypeFuncResult func(int) error

// VarAddTypeFuncResult detects an add in a func's return list
var VarAddTypeFuncResult func(int)

// VarRemoveTypeFuncResult detects a removal in a func's return list
var VarRemoveTypeFuncResult func(int) error

// VarChangeTypeSlice detects a change in a slice's type
var VarChangeTypeSlice []int

// VarChangeTypeArrayLen detects a change between slice and array
var VarChangeTypeSliceLen []int

// VarChangeTypeArrayLen detects a change in an array's length
var VarChangeTypeArrayLen [1]int

// VarChangeTypeArrayType detects a change in an array's type
var VarChangeTypeArrayType [1]int

// VarChangeTypeMapKey detects a change in a map's key
var VarChangeTypeMapKey map[int]int

// VarChangeTypeMapValue detects a change in a map's value
var VarChangeTypeMapValue map[int]int

// VarChangeTypeSelector detects a change in a selector.ident
var VarChangeTypeSelector bytes.Buffer

// VarChangeTypeStar detects a change in a pointer
var VarChangeTypeStar *int
var VarChangeTypeStarSelector *bytes.Buffer

// TypeSpecChange detects a change between types specs
type TypeSpecChange struct{}

// Struct checks for support of struct fields
type Struct struct{ A int }
type StructPtr struct{ A *int }
type StructPkg struct{ A bytes.Buffer }
type StructPtrPkg struct{ A *bytes.Buffer }
type StructMapPkg struct{ A map[int]bytes.Buffer }
type StructMapPtrPkg struct{ A map[int]*bytes.Buffer }

// StructAddMember detects additions of struct fields (is not a problem)
type StructAddMember struct {
	//Member1 will be added
	//Member2 will be added
}

// StructEmbedAddMember detects additions of struct fields with embedded fields (is not a problem)
type StructEmbedAddMember struct {
	//Member1 will be added
	Struct
	*StructPtr
	bytes.Buffer
	*bytes.Reader
}

// StructRemMember detects removals of struct fields
type StructRemMember struct {
	Member1 int
}

// StructRemEmbed detects removals of embedded struct fields
type StructRemEmbed struct {
	Struct
}

type structPriv struct{}

// StructRemPrivEmbed tests for ignorance in removal of elds
type StructRemPrivEmbed struct {
	structPriv
}

// StructChangeMember detects changes of struct fields
type StructChangeMember struct {
	Member1 int
}

// StructInferredMember checks for support of shorthand types
type StructChangeInferredMember struct {
	Member1, Member2 int
}

// StructRemPrivMember tests for ignorance in removal of private members
type StructRemPrivMember struct {
	private1, // will be removed
	private2 int // will be removed
	Public,
	private3 int // will be removed
}

// StructChangePrivMember tests for ignorance in changes in private members
type StructChangePrivMember struct {
	private int
}

// IfaceEmbed checks for support of interfaces with embedded values
type IfaceEmbed interface {
	io.Reader
}

// IfaceEmbedResolve tests for ignorance of embedded to non embedded of same signature
type IfaceEmbedResolve interface {
	io.Reader
}

// IfaceEmbedCompact tests for ignorance of non embedded to embedded of same signature
type IfaceEmbedCompact interface {
	Read(p []byte) (n int, err error)
}

// IfaceInferred tests for ignorance of shorthand type syntax
type IfaceInferred interface {
	Member1(arg1 int, arg2 int) (ret1 bool, ret2 bool)
}

// IfaceAddMember detects additions of interface methods
type IfaceAddMember interface {
	//Member1 will be added
}

// IfaceRemMember detects removals of interface methods (is not a problem)
type IfaceRemMember interface {
	Member1(arg1 int) (ret1 bool)
}

// IfaceChangeArgName detects argument renaming of interface methods (is not a problem)
type IfaceChangeArgName interface {
	Member1(arg1 int) (ret1 bool)
}

// IfaceChangeMemberArg detects changes of interface methods arguments
type IfaceChangeMemberArg interface {
	Member1(arg1 int) (ret1 bool)
}

// IfaceChangeMemberReturn detects changes of interface methods return params
type IfaceChangeMemberReturn interface {
	Member1(arg1 int) (ret1 bool)
}

// TypeAlias detects changes to alias types
type TypeAlias int

// FuncRetEmptyFunc tests handling of a func return bare func
func FuncRetEmptyFunc() func()

// FuncArg tests handing of function args that don't change
func FuncArg(arg1 int)                    {}
func FuncArgPtr(arg1 *int)                {}
func FuncArgPkg(arg1 bytes.Buffer)        {}
func FuncArgPtrPkg(arg1 *bytes.Buffer)    {}
func FuncArgFuncPkg(func(A bytes.Buffer)) {}

// FuncInferred tests for ignorance of shorthand type syntax
func FuncInferred(arg1 int, arg2 int) (ret1 bool, ret2 bool) {}

// FuncRenameArg tests ignorance of changes in variable names
func FuncRenameArg(arg1 int) (ret1 error) {}

// FuncAddArg detects additions of function parameter types
func FuncAddArg() {}

// FuncRemArg detects removals of function parameter types
func FuncRemArg(arg1 int) {}

// FuncChangeArg detects changes of function parameter types
func FuncChangeArg(arg1 int) {}

// FuncChangeChan detects changes of function channel parameter types
func FuncChangeChan(arg1 chan int) {}

// FuncChangeChanDir detects changes of function channel parameter types' direction
func FuncChangeChanDir(arg1 chan int) {}

// FuncChangeChanDirRelax detects relaxion of channel parameter type
func FuncChangeChanDirRelax(arg1 <-chan int) {}

// FuncAddRet detects additions of function return params (is not a problem)
func FuncAddRet() {}

// FuncAddRetMore detects additions of function return params
func FuncAddRetMore() error { return nil }

// FuncRemRet detects removals of function return params
func FuncRemRet() error { return nil }

// FuncChangeArg detects changes of function return params
func FuncChangeRet() error                     { return nil }
func FuncChangeRetStarIdent() *int             { return nil }
func FuncChangeRetStarSelector() *bytes.Buffer { return nil }

// FuncRecv tests changes to receivers
type FuncRecv struct{}

func (_ *FuncRecv) Method1(arg1 int) (ret1 error) { return nil }
func (_ FuncRecv) Method2(arg1 int) (ret1 error)  { return nil }
func (_ FuncRecv) method3(arg1 int) (ret1 error)  { return nil }

// FuncAddVariadic detects addition of a variadic argument to a function (is not a problem)
func FuncAddVariadic() {}

// FuncChangeToVariadic detects parameter change to variadic of same type (is not a problem)
func FuncChangeToVariadic(_ int) {}

// FuncChangeToVariadicDiffType detects parameter change to variadic of a different type
func FuncChangeToVariadicDiffType(_ int) {}

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
func FuncInterfaceIncompatible(_ T1) {}

// FuncInterfaceCompatible detects changes between compatible interfaces (is not a problem)
func FuncInterfaceCompatible(_ T3) {}

// FuncInterfaceCompatible2 detects changes between compatible interfaces (is not a problem)
func FuncInterfaceCompatible2(_ io.WriteCloser) {}

// FuncInterfaceCompatible3 detects changes between compatible interfaces (is not a problem)
func FuncInterfaceCompatible3(_ T2) {}

type C1 int

// FuncCustomType tests for support of comparing custom types
func FuncCustomType(_ C1) {}

// PrivateReturned detects changes in unexported, but returned types
type s struct{ Member int }

func F1() s      { return s{} }
func F2() *s     { return &s{} }
func (s) F() int { return 0 }
