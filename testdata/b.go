package library

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

// ValChangeType detects a change of type for a constant
const ValChangeType uint = 0

// TypeSpecChange detects a change between types specs
type TypeSpecChange interface{}

// StructAddMember detects additions of struct fields (is not a problem)
type StructAddMember struct {
	Member1 int
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

// FuncAddArg detects additions of function parameter types
func FuncAddArg(arg1 int) {}

// FuncRemArg detects removals of function parameter types
func FuncRemArg() {}

// FuncChangeArg detects changes of function parameter types
func FuncChangeArg(param uint) {}

// FuncAddRet detects additions of function return params (is not a problem)
func FuncAddRet() error { return nil }

// FuncRemRet detects removals of function return params
func FuncRemRet() {}

// FuncChangeArg detects changes of function return params
func FuncChangeRet() bool { return false }

// FuncRecv tests changes to receivers
type FuncRecv struct{}

func (_ *FuncRecv) Method1(arg1 bool) (ret1 int) { return 1 }
func (_ FuncRecv) Method2(arg1 bool) (ret1 int)  { return 1 }

//var VarExp int = 1
//var varPriv int = 1

//// VarToConst tests vars to consts don't error
//const VarToConst int = 1

//// CommentChange tests comments can change after
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
//func FuncExp1(a int) error { return nil }

//// Func2Recv tests changes to receivers
//type Func2Recv struct{}

//func (_ *Func2Recv) Method(arg1 int, arg2 *int) (ret1 error) { return nil }
//func (_ *Func2Recv) methodPriv()                             {}

//func funcPriv(a int) error {
//return nil
//}
