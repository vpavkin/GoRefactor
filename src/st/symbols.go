package st
//Package contains definitions of various symbol types, descripting GO entities

import (
	"go/ast"
	"go/token"
	"container/vector"
)
import "strconv"
//import "fmt"

func init() {

	PredeclaredTypes = make(map[string]*BasicTypeSymbol)
	for _, s := range ast.BasicTypes {
		PredeclaredTypes[s] = MakeBasicType(s, nil)
	}

	PredeclaredConsts = make(map[string]*VariableSymbol)

	b := PredeclaredTypes["bool"]
	n := MakeInterfaceType(NO_NAME, nil)

	PredeclaredConsts["true"] = MakeVariable("true", nil, b)
	PredeclaredConsts["false"] = MakeVariable("false", nil, b)
	PredeclaredConsts["nil"] = MakeVariable("nil", nil, n)
	PredeclaredConsts["iota"] = MakeVariable("iota", nil, PredeclaredTypes["int"])

	//make,new,cmplx,imag,real,append - in concrete occasion
	//print, println - nothing interesting

	capFts := MakeFunctionType(NO_NAME, nil)
	capFts.Parameters.addSymbol(MakeVariable("_", nil, MakeInterfaceType(NO_NAME, nil)))
	capFts.Results.addSymbol(MakeVariable("_", nil, PredeclaredTypes["int"]))

	closeFts := MakeFunctionType(NO_NAME, nil)
	closeFts.Parameters.addSymbol(MakeVariable("_", nil, MakeChannelType(NO_NAME, nil, nil)))

	closedFts := MakeFunctionType(NO_NAME, nil)
	closedFts.Parameters.addSymbol(MakeVariable("_", nil, MakeChannelType(NO_NAME, nil, nil)))
	closedFts.Results.addSymbol(MakeVariable("_", nil, PredeclaredTypes["bool"]))

	copyFts := MakeFunctionType(NO_NAME, nil)
	copyFts.Parameters.addSymbol(MakeVariable("p1", nil, MakeArrayType(NO_NAME, nil, nil, 0)))
	copyFts.Parameters.addSymbol(MakeVariable("p2", nil, MakeArrayType(NO_NAME, nil, nil, 0)))
	copyFts.Results.addSymbol(MakeVariable("_", nil, PredeclaredTypes["int"]))

	panicFts := MakeFunctionType(NO_NAME, nil)
	panicFts.Parameters.addSymbol(MakeVariable("_", nil, MakeInterfaceType(NO_NAME, nil)))

	recoverFts := MakeFunctionType(NO_NAME, nil)
	recoverFts.Results.addSymbol(MakeVariable("_", nil, MakeInterfaceType(NO_NAME, nil)))

	lenFts := MakeFunctionType(NO_NAME, nil)
	lenFts.Results.addSymbol(MakeVariable("_", nil, PredeclaredTypes["int"]))

	noResultsFts := MakeFunctionType(NO_NAME, nil)

	predeclaredFunctionTypes = make(map[string]*FunctionTypeSymbol)
	predeclaredFunctionTypes["cap"] = capFts
	predeclaredFunctionTypes["close"] = closeFts
	predeclaredFunctionTypes["closed"] = closedFts
	predeclaredFunctionTypes["copy"] = copyFts
	predeclaredFunctionTypes["panic"] = panicFts
	predeclaredFunctionTypes["recover"] = recoverFts
	predeclaredFunctionTypes["print"] = noResultsFts
	predeclaredFunctionTypes["println"] = noResultsFts
	predeclaredFunctionTypes["cmplx"] = noResultsFts
	predeclaredFunctionTypes["imag"] = noResultsFts
	predeclaredFunctionTypes["len"] = lenFts
	predeclaredFunctionTypes["make"] = noResultsFts
	predeclaredFunctionTypes["new"] = noResultsFts
	predeclaredFunctionTypes["real"] = noResultsFts
	predeclaredFunctionTypes["append"] = noResultsFts

	PredeclaredFunctions = make(map[string]*FunctionSymbol)
	for _, s := range builtIn {
		PredeclaredFunctions[s] = MakeFunction(s, nil, predeclaredFunctionTypes[s])
	}

}

var builtIn []string = []string{"cap", "close", "closed", "cmplx", "copy", "imag", "len", "make", "new", "panic",
	"print", "println", "real", "recover","append"}
var integerTypes map[string]bool = map[string]bool{"uintptr": true, "byte": true, "int8": true, "int16": true, "int32": true, "int64": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "int": true, "uint": true}
var floatTypes map[string]bool = map[string]bool{"float32": true, "float64": true, "float": true}
var complexTypes map[string]bool = map[string]bool{"complex64": true, "complex128": true, "complex": true}

var PredeclaredTypes map[string]*BasicTypeSymbol
var PredeclaredFunctions map[string]*FunctionSymbol
var PredeclaredConsts map[string]*VariableSymbol
var predeclaredFunctionTypes map[string]*FunctionTypeSymbol


/* Interfaces */
//Main interface, implemented by every symbol
type Symbol interface {
	Positions() PositionSet
	Identifiers() IdentSet //Set of Idents, representing symbol
	AddIdent(*ast.Ident)
	SetName(name string)
	Name() string
	AddPosition(token.Position)
	HasPosition(token.Position) bool
	PackageFrom() *Package
	Scope() *SymbolTable

	String() string
}

type IdentifierMap map[*ast.Ident]Symbol

func (im IdentifierMap) AddIdent(ident *ast.Ident, sym Symbol) {
	im[ident] = sym
}
func (im IdentifierMap) GetSymbol(ident *ast.Ident) Symbol {
	s, ok := im[ident]
	if !ok {
		panic("untracked ident "+ident.Name)
	}
	return s;
}

type IdentSet map[*ast.Ident]bool

func NewIdentSet() IdentSet {
	return make(map[*ast.Ident]bool)
}
func (set IdentSet) AddIdent(ident *ast.Ident) {
	set[ident] = true
}

type PositionSet map[string]token.Position

func (set PositionSet) AddPosition(p token.Position) {
	set[makePositionKey(p)] = p
}
func NewPositionSet() PositionSet {
	return make(map[string]token.Position)
}
//Interface for symbols that describe types
type ITypeSymbol interface {
	Symbol //Every type symbol is a symbol
	SetMethods(*SymbolTable)
	Methods() *SymbolTable //Returns type's methods
	AddMethod(meth Symbol) //Adds a method to the type symbol
	//Copy() ITypeSymbol     //Returns a value-copy of a type symbol
}

/***TypeSymbols. Implement ITypeSymbol***/

//Common part (base) embedded in any type symbol
type TypeSymbol struct {
	name   string
	Idents IdentSet //Corresponding program entity such as a package, constant, type, variable, or function/method
	//List of type's methods
	Meths  *SymbolTable
	Posits PositionSet
	Scope_ *SymbolTable
}
//Dummy type symbol, used in forward declarations during first pass
type UnresolvedTypeSymbol struct {
	*TypeSymbol
	Declaration ast.Expr
}

//Basic type symbol
type BasicTypeSymbol struct {
	*TypeSymbol
}

//Pseudonim type symbol
type AliasTypeSymbol struct {
	*TypeSymbol
	BaseType ITypeSymbol
}

const (
	SLICE    = -1
	ELLIPSIS = -2
)

//Array type symbol
type ArrayTypeSymbol struct {
	*TypeSymbol
	ElemType ITypeSymbol
	Len      int
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
	BaseType ITypeSymbol
	Fields   *SymbolTable
}
//Struct type symbol
type StructTypeSymbol struct {
	*TypeSymbol
	Fields *SymbolTable
}


/***Other Symbols. Implement Symbol***/

type PackageSymbol struct {
	name      string
	Idents    IdentSet     //local name of package (for unnamed - real name)
	ShortPath string       // "go/ast", "fmt" etc.
	Posits    PositionSet  //local file name occurances
	Package   *Package     //package entitie that's described by symbol
	Scope_    *SymbolTable //scope where symbol is declared
}

type Package struct { //NOT A SYMBOL
	QualifiedPath string //full filesystem path to package src folder

	Symbols         *SymbolTable   //top level declarations
	SymbolTablePool *vector.Vector //links to all symbol tables including nested

	FileSet     *token.FileSet
	AstPackage  *ast.Package              //ast tree 
	Imports     map[string]*vector.Vector //map[file] *[]packageSymbol
	IsGoPackage bool                      //true if package source is in $GOROOT/src/pkg/

	Communication chan int
}

func NewPackage(qualifiedPath string, fileSet *token.FileSet, astPackage *ast.Package) *Package {
	p := &Package{QualifiedPath: qualifiedPath, FileSet: fileSet, AstPackage: astPackage}

	p.Symbols = NewSymbolTable(p)
	p.SymbolTablePool = new(vector.Vector)
	p.SymbolTablePool.Push(p.Symbols)
	p.Imports = make(map[string]*vector.Vector)
	p.Communication = make(chan int)
	return p
}

//Symbol represents a function or a method
type FunctionSymbol struct {
	name              string
	Idents            IdentSet
	FunctionType      ITypeSymbol  //FunctionTypeSymbol or Alias
	Locals            *SymbolTable //Local variables
	Posits            PositionSet
	Scope_            *SymbolTable
	IsInterfaceMethod bool
}

//Symbol Represents a variable
type VariableSymbol struct {
	name            string
	Idents          IdentSet
	VariableType    ITypeSymbol
	IsTypeSwitchVar bool
	Posits          PositionSet
	Scope_          *SymbolTable
}


/*^^Other Symbol Methods^^*/
const NO_NAME string = ""

func (s *TypeSymbol) Name() string { return s.name }

func (s *BasicTypeSymbol) Name() string { return s.name }

func (s *PointerTypeSymbol) Name() string {

	ss := "*"
	return ss + s.BaseType.Name()
}
func (s *PointerTypeSymbol) BaseName() string {

	ss := s.BaseType.Name()
	for ss[0] == '*' {
		ss = ss[1:]
	}
	return ss
}

func (s *PackageSymbol) Name() string { return s.name }

func (s *VariableSymbol) Name() string { return s.name }

func (s *FunctionSymbol) Name() string { return s.name }

func (s *ArrayTypeSymbol) Name() string { return s.name }

func (s *ChanTypeSymbol) Name() string { return s.name }

func (s *FunctionTypeSymbol) Name() string { return s.name }

func (s *InterfaceTypeSymbol) Name() string { return s.name }

func (s *MapTypeSymbol) Name() string { return s.name }

func (s *StructTypeSymbol) Name() string { return s.name }


func makePositionKey(pos token.Position) string {
	return pos.Filename + ": " + strconv.Itoa(pos.Line) + "," + strconv.Itoa(pos.Column)
}

func hasPosition(sym Symbol, pos token.Position) bool {
	if _, ok := sym.Positions()[makePositionKey(pos)]; ok {
		return true
	}
	return false
}

func (ts *TypeSymbol) HasPosition(pos token.Position) bool {
	return hasPosition(ts, pos)
}
func (ts *PackageSymbol) HasPosition(pos token.Position) bool {
	return hasPosition(ts, pos)
}
func (ts *FunctionSymbol) HasPosition(pos token.Position) bool {
	return hasPosition(ts, pos)
}
func (ts *VariableSymbol) HasPosition(pos token.Position) bool {
	return hasPosition(ts, pos)
}

func (ts *TypeSymbol) PackageFrom() *Package {
	if ts.Scope_ != nil {
		return ts.Scope_.Package
	}
	return nil
}
func (ps *PackageSymbol) PackageFrom() *Package {
	if ps.Scope_ != nil {
		return ps.Scope_.Package
	}
	return nil
}
func (fs *FunctionSymbol) PackageFrom() *Package {
	if fs.Scope_ != nil {
		return fs.Scope_.Package
	}
	return nil
}
func (vs *VariableSymbol) PackageFrom() *Package {
	if vs.Scope_ != nil {
		return vs.Scope_.Package
	}
	return nil
}

func (ts *TypeSymbol) Scope() *SymbolTable     { return ts.Scope_ }
func (ps *PackageSymbol) Scope() *SymbolTable  { return ps.Scope_ }
func (fs *FunctionSymbol) Scope() *SymbolTable { return fs.Scope_ }
func (vs *VariableSymbol) Scope() *SymbolTable { return vs.Scope_ }

func (ts *TypeSymbol) Positions() PositionSet        { return ts.Posits }
func (ps *PackageSymbol) Positions() PositionSet     { return ps.Posits }
func (fs *FunctionSymbol) Positions() PositionSet    { return fs.Posits }
func (vs *VariableSymbol) Positions() PositionSet    { return vs.Posits }
func (ts *PointerTypeSymbol) Positions() PositionSet { return ts.BaseType.Positions() }


func (ts *TypeSymbol) Identifiers() IdentSet     { return ts.Idents }
func (ps *PackageSymbol) Identifiers() IdentSet  { return ps.Idents }
func (fs *FunctionSymbol) Identifiers() IdentSet { return fs.Idents }
func (vs *VariableSymbol) Identifiers() IdentSet { return vs.Idents }


func (ts *TypeSymbol) SetName(name string)     { ts.name = name }
func (ps *PackageSymbol) SetName(name string)  { ps.name = name }
func (vs *VariableSymbol) SetName(name string) { vs.name = name }
func (fs *FunctionSymbol) SetName(name string) { fs.name = name }

func addPosition(sym Symbol, p token.Position) {
	if sym.Positions() != nil {
		sym.Positions().AddPosition(p)
	}
}

func (ts *TypeSymbol) AddPosition(p token.Position) {
	addPosition(ts, p)
}
func (ts *PackageSymbol) AddPosition(p token.Position) {
	addPosition(ts, p)
}
func (ts *VariableSymbol) AddPosition(p token.Position) {
	addPosition(ts, p)
}
func (ts *FunctionSymbol) AddPosition(p token.Position) {
	addPosition(ts, p)
}


func addIdent(sym Symbol, ident *ast.Ident) {
	sym.Identifiers().AddIdent(ident)
}

func (s *TypeSymbol) AddIdent(ident *ast.Ident) {
	addIdent(s, ident)
}
func (s *PackageSymbol) AddIdent(ident *ast.Ident) {
	addIdent(s, ident)
}
func (s *VariableSymbol) AddIdent(ident *ast.Ident) {
	addIdent(s, ident)
}
func (s *FunctionSymbol) AddIdent(ident *ast.Ident) {
	addIdent(s, ident)
}
/* ITypeSymbol.Copy() Methods */

// func (ts TypeSymbol) Copy() ITypeSymbol {
// 	return &TypeSymbol{Ident: ts.Ident, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}
// }
// func (ts UnresolvedTypeSymbol) Copy() ITypeSymbol {
// 	return &UnresolvedTypeSymbol{&TypeSymbol{Ident: ts.Ident, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}, ts.Declaration}
// }
// func (ts AliasTypeSymbol) Copy() ITypeSymbol {
// 	return &AliasTypeSymbol{&TypeSymbol{Ident: ts.Ident, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}, ts.BaseType}
// }
// func (ts ArrayTypeSymbol) Copy() ITypeSymbol {
// 	return &ArrayTypeSymbol{&TypeSymbol{Ident: ts.Ident, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}, ts.ElemType, ts.Len}
// }
// func (ts InterfaceTypeSymbol) Copy() ITypeSymbol {
// 	return &InterfaceTypeSymbol{&TypeSymbol{Ident: ts.Ident, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}}
// }
// func (ts ChanTypeSymbol) Copy() ITypeSymbol {
// 	return &ChanTypeSymbol{&TypeSymbol{Ident: ts.Ident, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}, ts.ValueType}
// }
// func (ts FunctionTypeSymbol) Copy() ITypeSymbol {
// 	return &FunctionTypeSymbol{&TypeSymbol{Ident: ts.Ident, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}, ts.Parameters, ts.Results, ts.Reciever}
// }
// func (ts MapTypeSymbol) Copy() ITypeSymbol {
// 	return &MapTypeSymbol{&TypeSymbol{Ident: ts.Ident, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}, ts.KeyType, ts.ValueType}
// }
// func (ts PointerTypeSymbol) Copy() ITypeSymbol {
// 	return &PointerTypeSymbol{&TypeSymbol{Ident: ts.Ident, Meths: NewSymbolTable(nil), Posits: nil, PackFrom: ts.PackFrom}, ts.BaseType, ts.Fields}
// }
// func (ts StructTypeSymbol) Copy() ITypeSymbol {
// 	return &StructTypeSymbol{&TypeSymbol{Ident: ts.Ident, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}, ts.Fields}
// }
// func (vs VariableSymbol) Copy() *VariableSymbol {
// 	return &VariableSymbol{vs.Ident, vs.VariableType, vs.Posits, vs.PackFrom, vs.ReadOnly}
// }
// func (fs FunctionSymbol) Copy() *FunctionSymbol {
// 	return &FunctionSymbol{fs.Ident, fs.FunctionType, fs.Locals, fs.Posits, fs.PackFrom, fs.ReadOnly}
// }

/*^^Other ITypeSymbol Methods^^*/


//ITypeSymbol.Methods()
func (ts *TypeSymbol) Methods() *SymbolTable { return ts.Meths }

//ITypeSymbol.AddMethod()
func (ts *TypeSymbol) AddMethod(meth Symbol) {
	ts.Meths.AddSymbol(meth)
}
//ITypeSymbol.SetMethods()
func (ts *TypeSymbol) SetMethods(table *SymbolTable) {
	ts.Meths = table
}


func GetBaseType(sym ITypeSymbol) (ITypeSymbol, bool) {
	switch sym.(type) {
	case *PointerTypeSymbol, *AliasTypeSymbol:
		visitedTypes := make(map[string]ITypeSymbol)
		return getBaseType(sym, visitedTypes)
	}
	return sym, false

}
func GetBaseTypeOnlyPointer(sym ITypeSymbol) (ITypeSymbol, bool) {
	switch sym.(type) {
	case *PointerTypeSymbol:
		visitedTypes := make(map[string]ITypeSymbol)
		return getBaseTypeOnlyPointer(sym, visitedTypes)
	}
	return sym, false
}

func getBaseTypeOnlyPointer(sym ITypeSymbol, visited map[string]ITypeSymbol) (ITypeSymbol, bool) {
	if _, ok := visited[sym.Name()]; ok {
		return nil, true
	}
	if sym.Name() != "" {
		visited[sym.Name()] = sym
	}
	switch t := sym.(type) {
	case *PointerTypeSymbol:
		return getBaseType(t.BaseType, visited)
	}
	return sym, false
}


func getBaseType(sym ITypeSymbol, visited map[string]ITypeSymbol) (ITypeSymbol, bool) {
	if _, ok := visited[sym.Name()]; ok {
		return nil, true
	}
	if sym.Name() != "" {
		visited[sym.Name()] = sym
	}
	switch t := sym.(type) {
	case *PointerTypeSymbol:
		return getBaseType(t.BaseType, visited)
	case *AliasTypeSymbol:
		return getBaseType(t.BaseType, visited)
	}
	return sym, false
}

func (pt *PointerTypeSymbol) GetBaseStruct() (*StructTypeSymbol, bool) {
	t, _ := GetBaseType(pt)
	// 	if pt.Name() == "*Package" {
	// 		fmt.Printf("____%s__%s__\n", t.Name(), t.PackageFrom().AstPackage.Name)
	// 	}
	s, ok := t.(*StructTypeSymbol)
	return s, ok
}

func (pt *PointerTypeSymbol) Depth() int {
	switch t := pt.BaseType.(type) {
	case *PointerTypeSymbol:
		return t.Depth() + 1
	}
	return 1
}

func (ps *PackageSymbol) SetMethods(*SymbolTable) {
	panic("mustn't call ITypeSymbol methods on PackageSymbol")
}
func (ps *PackageSymbol) Methods() *SymbolTable {
	panic("mustn't call ITypeSymbol methods on PackageSymbol")
	return nil
}
func (ps *PackageSymbol) AddMethod(meth Symbol) {
	panic("mustn't call ITypeSymbol methods on PackageSymbol")
}


func SetIdentObject(ident *ast.Ident) *ast.Object {
	ident.Obj = &ast.Object{Name: ident.Name}
	return ident.Obj
}

func IsPredeclaredIdentifier(name string) bool {
	if _, ok := PredeclaredTypes[name]; ok {
		return true
	}
	if _, ok := PredeclaredFunctions[name]; ok {
		return true
	}
	if _, ok := PredeclaredConsts[name]; ok {
		return true
	}
	return false
}

func IsIntegerType(name string) (r bool) {
	_, r = integerTypes[name]
	return
}
func IsFloatType(name string) (r bool) {
	_, r = floatTypes[name]
	return
}
func IsComplexType(name string) (r bool) {
	_, r = complexTypes[name]
	return
}

/*^^Symbol.String() Methods^^*/


func (ts TypeSymbol) String() string {
	return ts.Name()
}

//Symbol.String()
func (ts BasicTypeSymbol) String() string {

	return ts.Name()
}

//Symbol.String()
func (ut UnresolvedTypeSymbol) String() string {

	return "?" + ut.Name() + "?"
}

//Symbol.String()
func (ats AliasTypeSymbol) String() string {
	return ats.Name() + " " + ats.BaseType.Name()
}

//Symbol.String()
func (ats ArrayTypeSymbol) String() string {
	return ats.Name() + " [" + strconv.Itoa(ats.Len) + "]" + ats.ElemType.Name()
}

//Symbol.String()
func (cts ChanTypeSymbol) String() string { return "chan " + cts.ValueType.Name() }

//Symbol.String()
func (fs FunctionTypeSymbol) String() string {

	s1 := ""
	s2 := ""
	if fs.Parameters != nil {
		fs.Parameters.forEach(func(v Symbol) {
			s1 += v.String() + ","
		})
	}
	if fs.Results != nil {
		fs.Results.forEach(func(v Symbol) {
			s2 += v.String() + ","
		})
	}
	return fs.Name() + "(" + s1 + ")(" + s2 + ")"

}

//Symbol.String()
func (its InterfaceTypeSymbol) String() string {
	//if its.Name() != "" { return its.Name()}
	s := "\n"
	if its.Methods() != nil {
		its.Methods().forEach(func(sym Symbol) {
			s += sym.String() + "\n"
		})
	}
	return its.Name() + " interface {" + s + "}"
}

//Symbol.String()
func (mts MapTypeSymbol) String() string {
	//fmt.Printf("BBBB")
	return mts.Name() + " map[" + mts.KeyType.Name() + "]" + mts.ValueType.Name()
}

//Symbol.String()
func (pts PointerTypeSymbol) String() string {

	return pts.Name()
}

//Symbol.String()
func (ts StructTypeSymbol) String() string {

	s := "\n"
	ts.Fields.forEach(func(v Symbol) {
		s += v.String() + "\n"
	})
	return ts.Name() + " struct {" + s + "}"
}

//Symbol.String()
func (fs FunctionSymbol) String() string {

	s := ""
	if fts, ok := fs.FunctionType.(*FunctionTypeSymbol); ok && fts != nil {
		if fts.Reciever != nil {
			fts.Reciever.forEach(func(r Symbol) {
				s = "(" + r.String() + ")"
			})
		}
	}
	fss := ""
	if fs.FunctionType.Name() == "" {
		fss += fs.FunctionType.String()
	} else {
		fss += fs.FunctionType.Name()
	}
	return fs.Name() + s + " " + fss
}

//Symbol.String()
func (vs VariableSymbol) String() string {
	return vs.Name() + " " + vs.VariableType.Name()
}

//Symbol.String()
func (vs PackageSymbol) String() string {
	return vs.name + " " + vs.ShortPath
}
