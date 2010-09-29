package symbolTable

import (
	"go/ast"
	"container/vector"
	"strconv"
)

/***Heplful stuff***/


//Predeclared type identifiers;
//TypeSymbols for these will not be added to symbolTable root
var predeclaredTypes map[string]bool = map[string]bool{"bool": true, "byte": true, "float32": true, "float64": true, "int8": true, "int16": true, "int32": true, "int64": true, "string": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "float": true, "int": true, "uint": true, "uintptr": true, "complex": true, "complex64": true, "complex128": true}

//Is name a predeclared type identifier
func IsPredeclaredType(name string) (r bool) {
	_, r = predeclaredTypes[name]
	return
}


//Represents a local SymbolTable with a number of opened scopes
type SymbolTable struct {
	Scope        *ast.Node         //pointer to corresponding ast.tree node or nil (Global table)
	Table        map[string]Symbol //symbol table
	OpenedScopes *vector.Vector    //vector of opened scopes
}

/*^^SymbolTable Methods and Functions^^*/

//Creates a new empty(but ready for work) SymbolTable and returns pointer to it
func NewSymbolTable() *SymbolTable {
	st := &SymbolTable{nil, make(map[string]Symbol), new(vector.Vector)}
	if STB != nil {
		STB.SymbolTablePool.Push(st)
	}
	return st
}

//Adds a scope to opened scopes list
func (st *SymbolTable) AddOpenedScope(scope *SymbolTable) {
	if scope == nil {
		return
	}
	st.OpenedScopes.Push(scope)
}

//Adds a symbol to local symbol table
func (st *SymbolTable) AddSymbol(sym Symbol) bool {
	name := sym.Name()
	st.Table[name] = sym
	return true
}

//Searches symbol table and it's opened scopes for a symbol with a given name
func (st SymbolTable) FindSymbolByName(name string) (sym Symbol, found bool) {
	if sym, found = st.Table[name]; !found {

		for x := range st.OpenedScopes.Iter() {
			v, _ := x.(*SymbolTable)
			if sym, found = v.FindSymbolByName(name); found {
				return
			}
		}
	}
	return
}

//Searches symbol table and it's opened scopes for a pointer type
//with a specified base type and depth
func (st SymbolTable) FindPointerType(base *ast.Object, depth int) (sym *PointerTypeSymbol, found bool) {
	for _, ss := range st.Table {
		if sym, ok := ss.(*PointerTypeSymbol); ok {
			if found = (sym.Obj.Name == base.Name && sym.PointerDepth == depth); found {
				return sym, found
			}
		}
	}
	for x := range st.OpenedScopes.Iter() {
		v, _ := x.(*SymbolTable)
		if sym, found = v.FindPointerType(base, depth); found {
			return
		}
	}
	return
}

func (st SymbolTable) FindTypeSwitchVar() (*VariableSymbol, bool) {
	for _, ss := range st.Table {
		if sym, ok := ss.(*VariableSymbol); ok && sym.Obj.Kind == -1 {
			return sym, ok
		}
	}
	return nil, false
}

//Searches symbol table and it's opened scopes for a symbol with a given name
func GetOrAddPointer(base ITypeSymbol, curSt *SymbolTable, toAdd *SymbolTable) (res *PointerTypeSymbol, added bool) {

	pd, found := 0, false
	added = false

	if p, ok := base.(*PointerTypeSymbol); ok {
		pd = p.PointerDepth
	}
	if base.Object() != nil {
		if res, found = curSt.FindPointerType(base.Object(), pd+1); found {
			return
		}
	}

	r := &PointerTypeSymbol{&TypeSymbol{Obj: base.Object(), Meths: NewSymbolTable(), Posits: new(vector.Vector)}, base, pd + 1, NewSymbolTable()}

	if str, ok := r.IsStructPointer(); ok {
		r.Fields.AddOpenedScope(str.Fields)
	}
	if base.Object() != nil {
		if !IsPredeclaredType(base.Object().Name) {
			r.Methods().AddOpenedScope(base.Methods())
			toAdd.AddSymbol(r)
			added = true
		}
	}
	res = r
	return
}
//String representation for printing
func (st SymbolTable) String() string {
	s := ""
	for _, sym := range st.Table {
		s += sym.String()
		if ts, ok := sym.(ITypeSymbol); ok {
			if ts.Methods() != nil {
				s += " <" + strconv.Itoa(ts.Methods().OpenedScopes.Len()) + ">"
			}
		}
		s += "\n\n"
	}
	return s
}

func (st SymbolTable) FindSymbolByPosition(fileName string, line int, column int) (sym Symbol, found bool) {
	for _, sym = range st.Table {
		for p := range sym.Positions().Iter() {
			pos := p.(Occurence).Pos
			if pos.Filename == fileName && pos.Line == line && pos.Column == column {
				found = true
				return
			}
		}
	}
	sym, found = nil, false
	return
}
