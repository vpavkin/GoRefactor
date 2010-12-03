package st
//Package contains definitions of various symbol types, descripting GO entities

import (
	"go/ast"
	"go/token"
	"container/vector"
)
import "strconv"
import "fmt"

func makeTypedVar(name string, t ITypeSymbol) *VariableSymbol {
	return &VariableSymbol{Obj: &ast.Object{Name: name}, VariableType: t}
}

func init() {

	PredeclaredTypes = make(map[string]*BasicTypeSymbol)
	for _, s := range ast.BasicTypes {
		PredeclaredTypes[s] = &BasicTypeSymbol{
			&TypeSymbol{Obj: &ast.Object{Kind: ast.Typ, Name: s}, Posits: make(map[string]token.Position)}}
	}

	PredeclaredConsts = make(map[string]*VariableSymbol)

	b := PredeclaredTypes["bool"]
	n := &InterfaceTypeSymbol{&TypeSymbol{Obj: nil, Meths: nil, Posits: make(map[string]token.Position)}}

	PredeclaredConsts["true"] = &VariableSymbol{&ast.Object{Name: "true"}, b, make(map[string]token.Position), nil, false}
	PredeclaredConsts["false"] = &VariableSymbol{&ast.Object{Name: "false"}, b, make(map[string]token.Position), nil, false}
	PredeclaredConsts["nil"] = &VariableSymbol{&ast.Object{Name: "nil"}, n, make(map[string]token.Position), nil, false}
	PredeclaredConsts["iota"] = &VariableSymbol{&ast.Object{Name: "iota"}, PredeclaredTypes["int"], make(map[string]token.Position), nil, false}

	//make,new,cmplx,imag,real,append - in concrete occasion
	//print, println - nothing interesting

	capFts := &FunctionTypeSymbol{Parameters: NewSymbolTable(nil), Results: NewSymbolTable(nil)}
	capFts.Parameters.addSymbol(makeTypedVar("_", &InterfaceTypeSymbol{}))
	capFts.Results.addSymbol(makeTypedVar("_", PredeclaredTypes["int"]))

	closeFts := &FunctionTypeSymbol{Parameters: NewSymbolTable(nil)}
	closeFts.Parameters.addSymbol(makeTypedVar("_", &ChanTypeSymbol{}))

	closedFts := &FunctionTypeSymbol{Parameters: NewSymbolTable(nil), Results: NewSymbolTable(nil)}
	closedFts.Parameters.addSymbol(makeTypedVar("_", &ChanTypeSymbol{}))
	closedFts.Results.addSymbol(makeTypedVar("_", PredeclaredTypes["bool"]))

	copyFts := &FunctionTypeSymbol{Parameters: NewSymbolTable(nil), Results: NewSymbolTable(nil)}
	copyFts.Parameters.addSymbol(makeTypedVar("p1", &ArrayTypeSymbol{}))
	copyFts.Parameters.addSymbol(makeTypedVar("p2", &ArrayTypeSymbol{}))
	copyFts.Results.addSymbol(makeTypedVar("_", PredeclaredTypes["int"]))

	panicFts := &FunctionTypeSymbol{Parameters: NewSymbolTable(nil)}
	panicFts.Parameters.addSymbol(makeTypedVar("_", &InterfaceTypeSymbol{}))

	recoverFts := &FunctionTypeSymbol{Results: NewSymbolTable(nil)}
	recoverFts.Results.addSymbol(makeTypedVar("_", &InterfaceTypeSymbol{}))

	lenFts := &FunctionTypeSymbol{Results: NewSymbolTable(nil)}
	lenFts.Results.addSymbol(makeTypedVar("_", PredeclaredTypes["int"]))

	noResultsFts := &FunctionTypeSymbol{}

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

	PredeclaredFunctions = make(map[string]*FunctionSymbol)
	for _, s := range builtIn {
		PredeclaredFunctions[s] = &FunctionSymbol{Obj: &ast.Object{Kind: ast.Fun, Name: s}, FunctionType: predeclaredFunctionTypes[s], Posits: make(map[string]token.Position)}
	}

}

var builtIn []string = []string{"cap", "close", "closed", "cmplx", "copy", "imag", "len", "make", "new", "panic",
	"print", "println", "real", "recover"}
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
	Positions() map[string]token.Position
	Object() *ast.Object       //Returns corresponding object
	SetObject(obj *ast.Object) //Defines an object for symbol
	Name() string
	AddPosition(token.Position)
	IsReadOnly() bool
	PackageFrom() *Package

	String() string
}

//Interface for symbols that describe types
type ITypeSymbol interface {
	Symbol //Every type symbol is a symbol
	SetMethods(*SymbolTable)
	Methods() *SymbolTable //Returns type's methods
	AddMethod(meth Symbol) //Adds a method to the type symbol
	Copy() ITypeSymbol     //Returns a value-copy of a type symbol
}

/***TypeSymbols. Implement ITypeSymbol***/

//Common part (base) embedded in any type symbol
type TypeSymbol struct {
	Obj *ast.Object //Corresponding program entity such as a package, constant, type, variable, or function/method
	//List of type's methods
	Meths    *SymbolTable
	Posits   map[string]token.Position
	PackFrom *Package
	ReadOnly bool
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
	Obj          *ast.Object               //local name of package (for unnamed - real name)
	Path         string                    // "go/ast", "fmt" etc.
	Posits       map[string]token.Position //local file name occurances
	Package      *Package                  //package entitie that's described by symbol
	PackFrom     *Package                  //package where symbol is declared
	HasLocalName bool
}

type Package struct { //NOT A SYMBOL
	QualifiedPath string //full filesystem path to package src folder

	Symbols         *SymbolTable   //top level declarations
	SymbolTablePool *vector.Vector //links to all symbol tables including nested

	AstPackage  *ast.Package              //ast tree 
	Imports     map[string]*vector.Vector //map[file] *[]packageSymbol
	IsGoPackage bool                      //true if package source is in $GOROOT/src/pkg/

	Communication chan int
}

func NewPackage(qualifiedPath string, astPackage *ast.Package) *Package {
	p := &Package{QualifiedPath: qualifiedPath, AstPackage: astPackage}

	p.Symbols = NewSymbolTable(p)
	p.SymbolTablePool = new(vector.Vector)
	p.SymbolTablePool.Push(p.Symbols)
	p.Imports = make(map[string]*vector.Vector)
	p.Communication = make(chan int)
	return p
}

//Symbol represents a function or a method
type FunctionSymbol struct {
	Obj          *ast.Object
	FunctionType ITypeSymbol  //FunctionTypeSymbol
	Locals       *SymbolTable //Local variables
	Posits       map[string]token.Position
	PackFrom     *Package
	ReadOnly     bool //true if symbol can't be renamed
}

//Symbol Represents a variable
type VariableSymbol struct {
	Obj          *ast.Object
	VariableType ITypeSymbol
	Posits       map[string]token.Position
	PackFrom     *Package
	ReadOnly     bool
}


func makePositionKey(pos token.Position) string {
	return pos.Filename + ": " + strconv.Itoa(pos.Line) + "," + strconv.Itoa(pos.Column)
}
/*^^Other Symbol Methods^^*/

func (ts *TypeSymbol) Name() string {
	if ts.Obj != nil {
		return ts.Obj.Name
	}
	return ""
}

func (bts *BasicTypeSymbol) Name() string {
	if bts.Obj == nil {
		fmt.Printf("AAAAAAAAAA\n")
	}
	return bts.Obj.Name
}

func (pts *PointerTypeSymbol) Name() string {

	s := "*"
	return s + pts.BaseType.Name()
}
func (pts *PointerTypeSymbol) BaseName() string {

	s := pts.BaseType.Name()
	for s[0] == '*' {
		s = s[1:]
	}
	return s
}

func (ps *PackageSymbol) Name() string {
	return ps.Obj.Name // will not be nil
}

func (vs *VariableSymbol) Name() string {
	if vs.Obj != nil {
		return vs.Obj.Name
	}
	return ""
}

func (fs *FunctionSymbol) Name() string {
	if fs.Obj != nil {
		return fs.Obj.Name
	}
	return ""
}

func (ats *ArrayTypeSymbol) Name() string {
	if ats.Obj != nil {
		return ats.Obj.Name
	}
	return ""
}

func (cts *ChanTypeSymbol) Name() string {
	if cts.Obj != nil {
		return cts.Obj.Name
	}
	return ""
}

func (fs *FunctionTypeSymbol) Name() string {
	if fs.Obj != nil {
		return fs.Obj.Name
	}
	return "^^^"
}

func (its *InterfaceTypeSymbol) Name() string {
	if its.Obj != nil {
		return its.Obj.Name
	}
	return ""
}

func (mts *MapTypeSymbol) Name() string {
	if mts.Obj != nil {
		return mts.Obj.Name
	}
	return ""
}

func (ts *StructTypeSymbol) Name() string {
	if ts.Obj != nil {
		return ts.Obj.Name
	}
	return ""
	// 	s := ""
	// 	for _, v := range ts.Fields.Table {
	// 		if vv, ok := v.(*VariableSymbol); ok {
	// 			s += vv.Name() + " " + vv.VariableType.Name() + ","
	// 		}
	// 	}
	// 	return "struct {" + s + "}"
}


func (ts TypeSymbol) PackageFrom() *Package     { return ts.PackFrom }
func (ps PackageSymbol) PackageFrom() *Package  { return ps.PackFrom }
func (fs FunctionSymbol) PackageFrom() *Package { return fs.PackFrom }
func (vs VariableSymbol) PackageFrom() *Package { return vs.PackFrom }

func (ts TypeSymbol) Positions() map[string]token.Position        { return ts.Posits }
func (ps PackageSymbol) Positions() map[string]token.Position     { return ps.Posits }
func (fs FunctionSymbol) Positions() map[string]token.Position    { return fs.Posits }
func (vs VariableSymbol) Positions() map[string]token.Position    { return vs.Posits }
func (ts PointerTypeSymbol) Positions() map[string]token.Position { return ts.BaseType.Positions() }


func (ts TypeSymbol) Object() *ast.Object     { return ts.Obj }
func (ps PackageSymbol) Object() *ast.Object  { return ps.Obj }
func (fs FunctionSymbol) Object() *ast.Object { return fs.Obj }
func (vs VariableSymbol) Object() *ast.Object { return vs.Obj }


func (ts *TypeSymbol) SetObject(obj *ast.Object)     { ts.Obj = obj }
func (ps *PackageSymbol) SetObject(obj *ast.Object)  { ps.Obj = obj }
func (vs *VariableSymbol) SetObject(obj *ast.Object) { vs.Obj = obj }
func (fs *FunctionSymbol) SetObject(obj *ast.Object) { fs.Obj = obj }


func (ps PackageSymbol) IsReadOnly() bool  { return false }
func (ts TypeSymbol) IsReadOnly() bool     { return ts.ReadOnly }
func (vs VariableSymbol) IsReadOnly() bool { return vs.ReadOnly }
func (fs FunctionSymbol) IsReadOnly() bool { return fs.ReadOnly }


func (ts TypeSymbol) AddPosition(p token.Position) {
	if ts.Positions() != nil {
		ts.Positions()[makePositionKey(p)] = p
	}
}
func (ts PackageSymbol) AddPosition(p token.Position) {
	if ts.Positions() != nil {
		ts.Positions()[makePositionKey(p)] = p
	}
}
func (ts VariableSymbol) AddPosition(p token.Position) {
	if ts.Positions() != nil {
		ts.Positions()[makePositionKey(p)] = p
	}
}
func (ts FunctionSymbol) AddPosition(p token.Position) {
	if ts.Positions() != nil {
		ts.Positions()[makePositionKey(p)] = p
	}
}

/* ITypeSymbol.Copy() Methods */

func (ts TypeSymbol) Copy() ITypeSymbol {
	return &TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}
}
func (ts UnresolvedTypeSymbol) Copy() ITypeSymbol {
	return &UnresolvedTypeSymbol{&TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}, ts.Declaration}
}
func (ts AliasTypeSymbol) Copy() ITypeSymbol {
	return &AliasTypeSymbol{&TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}, ts.BaseType}
}
func (ts ArrayTypeSymbol) Copy() ITypeSymbol {
	return &ArrayTypeSymbol{&TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}, ts.ElemType, ts.Len}
}
func (ts InterfaceTypeSymbol) Copy() ITypeSymbol {
	return &InterfaceTypeSymbol{&TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}}
}
func (ts ChanTypeSymbol) Copy() ITypeSymbol {
	return &ChanTypeSymbol{&TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}, ts.ValueType}
}
func (ts FunctionTypeSymbol) Copy() ITypeSymbol {
	return &FunctionTypeSymbol{&TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}, ts.Parameters, ts.Results, ts.Reciever}
}
func (ts MapTypeSymbol) Copy() ITypeSymbol {
	return &MapTypeSymbol{&TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}, ts.KeyType, ts.ValueType}
}
func (ts PointerTypeSymbol) Copy() ITypeSymbol {
	return &PointerTypeSymbol{&TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(nil), Posits: nil, PackFrom: ts.PackFrom}, ts.BaseType, ts.Fields}
}
func (ts StructTypeSymbol) Copy() ITypeSymbol {
	return &StructTypeSymbol{&TypeSymbol{Obj: ts.Obj, Meths: NewSymbolTable(nil), Posits: make(map[string]token.Position), PackFrom: ts.PackFrom}, ts.Fields}
}
func (vs VariableSymbol) Copy() *VariableSymbol {
	return &VariableSymbol{vs.Obj, vs.VariableType, vs.Posits, vs.PackFrom, vs.ReadOnly}
}
func (fs FunctionSymbol) Copy() *FunctionSymbol {
	return &FunctionSymbol{fs.Obj, fs.FunctionType, fs.Locals, fs.Posits, fs.PackFrom, fs.ReadOnly}
}
/*^^Other ITypeSymbol Methods^^*/


//ITypeSymbol.Methods()
func (ts TypeSymbol) Methods() *SymbolTable { return ts.Meths }

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
	if pt.Name() == "*Package" {
		fmt.Printf("____%s__%s__\n", t.Name(), t.PackageFrom().AstPackage.Name)
	}
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
func (ps *PackageSymbol) Copy() ITypeSymbol {
	panic("mustn't call ITypeSymbol methods on PackageSymbol")
	return nil
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
		for v := range fs.Parameters.Iter() {
			s1 += v.String() + ","
		}
	}
	if fs.Results != nil {
		for v := range fs.Results.Iter() {
			s2 += v.String() + ","
		}
	}
	return fs.Name() + "(" + s1 + ")(" + s2 + ")"

}

//Symbol.String()
func (its InterfaceTypeSymbol) String() string {
	//if its.Name() != "" { return its.Name()}
	s := "\n"
	if its.Methods() != nil {
		for sym := range its.Methods().Iter() {
			s += sym.String() + "\n"
		}
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
	for v := range ts.Fields.Iter() {
		s += v.String() + "\n"
	}
	return ts.Name() + " struct {" + s + "}"
}

//Symbol.String()
func (fs FunctionSymbol) String() string {

	s := ""
	if fts, ok := fs.FunctionType.(*FunctionTypeSymbol); ok && fts != nil {
		if fs.FunctionType.(*FunctionTypeSymbol).Reciever != nil {
			for r := range fs.FunctionType.(*FunctionTypeSymbol).Reciever.Iter() {
				s = "(" + r.String() + ")"
			}
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
	return vs.Obj.Name + " " + vs.Path
}
