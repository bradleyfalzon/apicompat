package library

import "bytes"

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

// ValDynamicType checks for (lack of) support types parser can't easily detect
var ValDynamicType = bytes.NewBufferString("some error")

// ValChangeType detects a change of type for a constant
var VarChangeType int

// VarChangeValSpecType detects a change between val spec types
var VarChangeValSpecType int

// VarChangeTypeFunc detects support for var funcs
var VarChangeTypeFunc func(arg1 int) (err error)

// VarChangeTypeFuncArgRename detects ignorance of argument name changes
var VarChangeTypeFuncArgRename func(arg1 int) (err1 error)

// VarChangeTypeFuncParam detects a change in a func'ss parameter list
var VarChangeTypeFuncParam func(int) error

// VarChangeTypeFuncResult detects a change in a func's return list
var VarChangeTypeFuncResult func(int) error

// VarAddTypeFuncResult detects an add in a func's return list (this is ok)
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

// StructAddMember detects additions of struct fields (is not a problem)
type StructAddMember struct {
	//Member1 will be added
	//Member2 will be added
}

// StructRemMember detects removals of struct fields
type StructRemMember struct {
	Member1 int
}

// StructChangeMember detects changes of struct fields
type StructChangeMember struct {
	Member1 int
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

// FuncRenameArg tests ignorance of changes in variable names
func FuncRenameArg(arg1 int) (ret1 error) {}

// FuncAddArg detects additions of function parameter types
func FuncAddArg() {}

// FuncRemArg detects removals of function parameter types
func FuncRemArg(arg1 int) {}

// FuncChangeArg detects changes of function parameter types
func FuncChangeArg(arg1 int) {}

// FuncAddRet detects additions of function return params (is not a problem)
func FuncAddRet() {}

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

//var VarExp int = 1
//var varPriv int = 1

//// VarToConst tests vars to consts don't error
//var VarToConst int = 1

//// CommentChange tests comments can change before
//var CommentChange int = 1

//type StructExp struct {
//MemberExp  int
//memberPriv int
//}

//type structPriv struct {
//MemberExp  int
//memberPriv int
//}

//type IfaceExp interface {
//MemberExp(int) error
//}

//type ifacePriv interface {
//MemberExp(int) error
//}

//func FuncExp(a int) error {
//return nil
//}

//func FuncAnonReturn() error { return nil }

//// Adding a return param doesn't break abi
//func FuncExp1(a int) {}

//// Func2Recv tests changes to receivers
//type Func2Recv struct{}

//func (_ *Func2Recv) Method(arg1 int, arg2 *int) (ret1 error) { return nil }
//func (_ *Func2Recv) methodPriv()                             {}

//func funcPriv(a int) error {
//return nil
//}
