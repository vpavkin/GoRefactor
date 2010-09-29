package symbolTable
//Package contains definitions of various symbol types, descripting GO entities

import (
	"go/ast"
	"go/token"
	"container/vector"
)
/***Interfaces***/

//Main interface, implemented by every symbol
type Symbol interface {
	Positions() *vector.Vector
	Object() *ast.Object       //Returns corresponding object
	SetObject(obj *ast.Object) //Defines an object for symbol
	String() string            //Returns string representation for printing
	Name() string
}

//Interface for symbols that describe types
type ITypeSymbol interface {
	Symbol //Every type symbol is a symbol
	SetMethods(st *SymbolTable)
	Methods() *SymbolTable //Returns type's methods
	AddMethod(meth Symbol) //Adds a method to the type symbol
	Copy() ITypeSymbol     //Returns a value-copy of a type symbol
}

/***TypeSymbols. Implement ITypeSymbol***/

//Common part (base) embedded in any type symbol
type TypeSymbol struct {
	Obj *ast.Object //Corresponding program entity such as a package, constant, type, variable, or function/method
	//List of type's methods
	Meths  *SymbolTable
	Posits *vector.Vector
}
//Dummy type symbol, used in forward declarations during first pass
type ImportedType struct {
	*TypeSymbol
}
//Pseudonim type symbol
type AliasTypeSymbol struct {
	*TypeSymbol
	BaseType ITypeSymbol
}
//Array type symbol
type ArrayTypeSymbol struct {
	*TypeSymbol
	ElemType ITypeSymbol
}
//Channel type symbol
type ChanTypeSymbol struct {
	*TypeSymbol
	ValueType ITypeSymbol
}
//Function type symbol
type FunctionTypeSymbol struct {
	*TypeSymbol
	Parameters *SymbolTable //(incoming) parameters
	Results    *SymbolTable //(outgoing) results
	Reciever   *SymbolTable //Reciever (if empty - function)
}
//Interface type symbol
type InterfaceTypeSymbol struct {
	*TypeSymbol //Interface methods stored in base.Methods
}
//Map type symbol
type MapTypeSymbol struct {
	*TypeSymbol
	KeyType   ITypeSymbol
	ValueType ITypeSymbol
}
//Pointer type symbol
type PointerTypeSymbol struct {
	*TypeSymbol
	BaseType     ITypeSymbol
	PointerDepth int
	Fields       *SymbolTable
}
//Struct type symbol
type StructTypeSymbol struct {
	*TypeSymbol
	Fields *SymbolTable
}


/***Other Symbols. Implement Symbol***/

//Symbol represents a function or a method
type FunctionSymbol struct {
	Obj          *ast.Object
	FunctionType ITypeSymbol  //FunctionTypeSymbol
	Locals       *SymbolTable //Local variables
	Posits       *vector.Vector
}

//Symbol Represents a variable
type VariableSymbol struct {
	Obj          *ast.Object
	VariableType ITypeSymbol
	Posits       *vector.Vector
}


//Occurence type
type Occurence struct {
	Pos token.Position
	Obj *ast.Object
}

func NewOccurence(pos token.Position, obj *ast.Object) Occurence {
	return Occurence{pos, obj}
}
/*^^Symbol.String() Methods^^*/

//Symbol.String()
func (ts TypeSymbol) String() string {
	if ts.Obj != nil {
		return ts.Obj.Name
	}
	return "anonymous"
}

//Symbol.String()
func (ut ImportedType) String() string {
	if ut.Obj != nil {
		return "#" + ut.Obj.Name + "#"
	}
	return "##"
}

//Symbol.String()
func (ats AliasTypeSymbol) String() string {
	if ats.Obj != nil {
		return ats.Obj.Name
	}
	return "anonymous" + "(" + ats.BaseType.String() + ")"
}

//Symbol.String()
func (ats ArrayTypeSymbol) String() string {
	if ats.Obj != nil {
		return ats.Obj.Name
	}
	return "[]" + ats.ElemType.String()
}

//Symbol.String()
func (cts ChanTypeSymbol) String() string { return "chan " + cts.ValueType.String() }

//Symbol.String()
func (fs FunctionTypeSymbol) String() string {
	s1 := ""
	s2 := ""
	if fs.Parameters != nil && fs.Results != nil {
		for _, v := range fs.Parameters.Table {
			s1 += v.String() + ","
		}
		for _, v := range fs.Results.Table {
			s2 += v.String() + ","
		}
		return "(" + s1 + ")(" + s2 + ")"
	}
	return ""

}

//Symbol.String()
func (its InterfaceTypeSymbol) String() string {

	if its.Obj != nil {
		return its.Obj.Name
	}
	return "interface " + "{}"

}

//Symbol.String()
func (mts MapTypeSymbol) String() string {
	if mts.Obj != nil {
		return mts.Obj.Name
	}
	return "map [" + mts.KeyType.String() + "]" + mts.ValueType.String()
}

//Symbol.String()
func (pts PointerTypeSymbol) String() string {
	s := "*"
	if pts.Obj != nil {
		return s + pts.BaseType.String()
	}
	return s + "anonymous"
}

//Symbol.String()
func (ts StructTypeSymbol) String() string {
	if ts.Obj != nil {
		return ts.Obj.Name
	}
	s := ""
	for _, v := range ts.Fields.Table {
		s += v.String() + ","
	}
	return "struct {" + s + "}"
}

//Symbol.String()
func (fs FunctionSymbol) String() string {
	if fs.Obj != nil {
		if fs.FunctionType == nil {
			//Interface inherited
			return fs.Obj.Name
		}
		if len(fs.FunctionType.(*FunctionTypeSymbol).Reciever.Table) == 0 {
			return "func " + fs.Obj.Name + " " + fs.FunctionType.String()
		} else {
			for _, s := range fs.FunctionType.(*FunctionTypeSymbol).Reciever.Table {
				return "func " + "(" + s.String() + ")" + fs.Obj.Name + " " + fs.FunctionType.String()
			}
		}

	}
	return "func anonymous " + fs.FunctionType.String()
}

//Symbol.String()
func (vs VariableSymbol) String() string {
	if vs.Obj != nil {
		return vs.Obj.Name + " " + vs.VariableType.String()
	}
	return "var anonymous " + vs.VariableType.String()
}


/*^^Other Symbol Methods^^*/

//Symbol.Name()
func (ts TypeSymbol) Name() string {
	if ts.Obj != nil {
		return ts.Obj.Name
	}
	return "anonymous"
}
//Symbol.Name()
func (pts PointerTypeSymbol) Name() string {
	s := "*"
	return s + pts.BaseType.Name()
}
//Symbol.Name()
func (vs VariableSymbol) Name() string {
	if vs.Obj != nil {
		return vs.Obj.Name
	}
	return "anonymous"
}
//Symbol.Name()
func (fs FunctionSymbol) Name() string {
	if fs.Obj != nil {
		return fs.Obj.Name
	}
	return "anonymous"
}

//Symbol.Name()
func (ats ArrayTypeSymbol) Name() string {
	if ats.Obj != nil {
		return ats.Obj.Name
	}
	return "[]" + ats.ElemType.Name()
}

//Symbol.Name()
func (cts ChanTypeSymbol) Name() string { return "chan " + cts.ValueType.Name() }

//Symbol.Name()
func (fs FunctionTypeSymbol) Name() string {
	if fs.Obj != nil {
		return fs.Obj.Name
	}
	return "func"
}

//Symbol.Name()
func (its InterfaceTypeSymbol) Name() string {
	if its.Obj != nil {
		return its.Obj.Name
	}
	return "interface " + "{}"
}

//Symbol.Name()
func (mts MapTypeSymbol) Name() string {
	if mts.Obj != nil {
		return mts.Obj.Name
	}
	return "map [" + mts.KeyType.Name() + "]" + mts.ValueType.Name()
}

//Symbol.Name()
func (ts StructTypeSymbol) Name() string {
	if ts.Obj != nil {
		return ts.Obj.Name
	}
	s := ""
	for _, v := range ts.Fields.Table {
		if vv, ok := v.(*VariableSymbol); ok {
			s += vv.Name() + " " + vv.VariableType.Name() + ","
		}
	}
	return "struct {" + s + "}"
}


//Symbol.Positions()
func (ts TypeSymbol) Positions() *vector.Vector { return ts.Posits }
//Symbol.Positions()
func (fs FunctionSymbol) Positions() *vector.Vector { return fs.Posits }
//Symbol.Positions()
func (vs VariableSymbol) Positions() *vector.Vector { return vs.Posits }
//Symbol.Positions()
func (ts PointerTypeSymbol) Positions() *vector.Vector { return ts.BaseType.Positions() }

//Symbol.Object()
func (ts TypeSymbol) Object() *ast.Object { return ts.Obj }

//Symbol.Object()
func (fs FunctionSymbol) Object() *ast.Object { return fs.Obj }

//Symbol.Object()
func (vs VariableSymbol) Object() *ast.Object { return vs.Obj }

//Symbol.SetObject()
func (ts *TypeSymbol) SetObject(obj *ast.Object) {
	ts.Obj = obj
}
//Symbol.SetObject()
func (vs *VariableSymbol) SetObject(obj *ast.Object) {
	vs.Obj = obj
}
//Symbol.SetObject()
func (fs *FunctionSymbol) SetObject(obj *ast.Object) {
	fs.Obj = obj
}


/*^^ITypeSymbol.Copy() Methods^^*/

//ITypeSymbol.Copy()
func (ts TypeSymbol) Copy() ITypeSymbol {
	return &TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(), Posits: new(vector.Vector)}
}
//ITypeSymbol.Copy()
func (ts AliasTypeSymbol) Copy() ITypeSymbol {
	return &AliasTypeSymbol{&TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(), Posits: new(vector.Vector)}, ts.BaseType}
}
//ITypeSymbol.Copy()
func (ts ArrayTypeSymbol) Copy() ITypeSymbol {
	return &ArrayTypeSymbol{&TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(), Posits: new(vector.Vector)}, ts.ElemType}
}
//ITypeSymbol.Copy()
func (ts InterfaceTypeSymbol) Copy() ITypeSymbol {
	return &InterfaceTypeSymbol{&TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(), Posits: new(vector.Vector)}}
}
//ITypeSymbol.Copy()
func (ts ChanTypeSymbol) Copy() ITypeSymbol {
	return &ChanTypeSymbol{&TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(), Posits: new(vector.Vector)}, ts.ValueType}
}
//ITypeSymbol.Copy()
func (ts FunctionTypeSymbol) Copy() ITypeSymbol {
	return &FunctionTypeSymbol{&TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(), Posits: new(vector.Vector)}, ts.Parameters, ts.Results, ts.Reciever}
}
//ITypeSymbol.Copy()
func (ts MapTypeSymbol) Copy() ITypeSymbol {
	return &MapTypeSymbol{&TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(), Posits: new(vector.Vector)}, ts.KeyType, ts.ValueType}
}
//ITypeSymbol.Copy()
func (ts PointerTypeSymbol) Copy() ITypeSymbol {
	return &PointerTypeSymbol{&TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(), Posits: new(vector.Vector)}, ts.BaseType, ts.PointerDepth, ts.Fields}
}
//ITypeSymbol.Copy()
func (ts StructTypeSymbol) Copy() ITypeSymbol {
	return &StructTypeSymbol{&TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(), Posits: new(vector.Vector)}, ts.Fields}
}

/*^^Other ITypeSymbol Methods^^*/


//ITypeSymbol.Methods()
func (ts TypeSymbol) Methods() *SymbolTable { return ts.Meths }

//ITypeSymbol.AddMethod()
func (ts *TypeSymbol) AddMethod(meth Symbol) {
	ts.Meths.AddSymbol(meth)
}
//ITypeSymbol.SetMethods()
func (ts *TypeSymbol) SetMethods(st *SymbolTable) {
	ts.Meths = st
}


func ContainsImportedType(sym ITypeSymbol) bool {
	switch t := sym.(type) {
	case *PointerTypeSymbol:
		return ContainsImportedType(t.BaseType)
	case *ImportedType:
		return true
	}
	return false
}

func GetBaseType(sym ITypeSymbol) ITypeSymbol {
	switch t := sym.(type) {
	case *PointerTypeSymbol:
		return GetBaseType(t.BaseType)
	}
	return sym
}

func (pt *PointerTypeSymbol) IsStructPointer() (*StructTypeSymbol, bool) {
	t := GetBaseType(pt)
	s, ok := t.(*StructTypeSymbol)
	return s, ok
}

func CopyObject(ident *ast.Ident) *ast.Object {
	ident.Obj = &ast.Object{Name: ident.Name()}
	return ident.Obj
}
