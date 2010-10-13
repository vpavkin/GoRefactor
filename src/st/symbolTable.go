package st

import (
	"container/vector"
	//"strconv"
)

import "fmt"
import "sort"


//Represents a local SymbolTable with a number of opened scopes
type SymbolTable struct {
	Table        map[string]Symbol //symbol table
	OpenedScopes *vector.Vector    //vector of opened scopes
	Package 	 *Package 	   //package that table belongs to
}

/*^^SymbolTable Methods and Functions^^*/


//Creates a new empty(but ready for work) SymbolTable and returns pointer to it
func NewSymbolTable(p *Package) *SymbolTable {
	return &SymbolTable{make(map[string]Symbol), new(vector.Vector),p}
}

//Adds a scope to opened scopes list
func (table *SymbolTable) AddOpenedScope(scope *SymbolTable) {
	if scope == nil {
		fmt.Println("NOTE: added nil scope -- AddOpenedScope, symbolTable.go")
		return
	}
	table.OpenedScopes.Push(scope)
}

//Adds a symbol to local symbol table
func (table *SymbolTable) AddSymbol(sym Symbol) bool {
	name := sym.Name()
	table.Table[name] = sym
	return true
}

//Searches symbol table and it's opened scopes for a symbol with a given name
func (table SymbolTable) LookUp(name string,fileName string) (sym Symbol, found bool) {

	if sym, found = table.Table[name]; !found {

		for _, x := range *table.OpenedScopes {
			v, _ := x.(*SymbolTable)
			if sym, found = v.LookUp(name,fileName); found {
				return
			}
		}
		if table.Package != nil{
			if imps,ok := table.Package.Imports[fileName]; ok && imps!=nil{
				for _,e := range *imps{ 
					ps:=e.(*PackageSymbol)
					if name == ps.Obj.Name{
						return ps,true;
					}
				}
			}
		}
	}
	return
}

//Searches symbol table and it's opened scopes for a pointer type
//with a specified base type and depth
func (table SymbolTable) LookUpPointerType(name string, depth int) (sym *PointerTypeSymbol, found bool) {
	for _, ss := range table.Table {
		if sym, ok := ss.(*PointerTypeSymbol); ok {
			if found = (sym.Name() == name && sym.Depth() == depth); found {
				return sym, found
			}
		}
	}
	for _, x := range *table.OpenedScopes {
		v, _ := x.(*SymbolTable)
		if sym, found = v.LookUpPointerType(name, depth); found {
			return
		}
	}
	return
}

func (table SymbolTable) FindTypeSwitchVar() (*VariableSymbol, bool) {
	for _, ss := range table.Table {
		if sym, ok := ss.(*VariableSymbol); ok && sym.Obj.Kind == -1 {
			return sym, ok
		}
	}
	return nil, false
}

//String representation for printing
// func (table SymbolTable) String() string {
// 	s := ""
// 	for _, sym := range table.Table {
// 		s += sym.Name()
// 		if ts, ok := sym.(ITypeSymbol); ok {
// 			if ts.Methods() != nil {
// 				s += " <" + strconv.Itoa(ts.Methods().OpenedScopes.Len()) + ">"
// 			}
// 		}
// 		s += "\n\n"
// 	}
// 	return s
// }

func (table SymbolTable) String() *vector.StringVector {
	var res = new(vector.StringVector)
	var s = new(vector.StringVector)

	for _, sym := range table.Table {
		if _, ok := sym.(*PackageSymbol); ok {
			s.Push("   package " + sym.String() + "\n")
		}
	}

	sort.Sort(s)
	s.Insert(0, "packages:\n")
	res.AppendVector(s)

	s = new(vector.StringVector)
	for _, sym := range table.Table {
		if ts, ok := sym.(ITypeSymbol); ok {
			if _, ok := PredeclaredTypes[ts.Name()]; !ok {
				s.Push("   type " + sym.String() + "\n")
			}
		}
	}

	sort.Sort(s)
	s.Insert(0, "types:\n")
	res.AppendVector(s)

	s = new(vector.StringVector)
	for _, sym := range table.Table {
		if ts, ok := sym.(*FunctionSymbol); ok {
			if _, ok := PredeclaredFunctions[ts.Name()]; !ok {
				s.Push("   func " + sym.String() + "\n")
			}
		}
	}
	sort.Sort(s)
	s.Insert(0, "methods:\n")
	res.AppendVector(s)

	s = new(vector.StringVector)
	for _, sym := range table.Table {
		if ts, ok := sym.(*VariableSymbol); ok {
			if _, ok := PredeclaredConsts[ts.Name()]; !ok {
				s.Push("   var " + sym.String() + "\n")
			}
		}
	}
	sort.Sort(s)
	s.Insert(0, "vars:\n")
	res.AppendVector(s)

	return res
}


func (table SymbolTable) FindSymbolByPosition(fileName string, line int, column int) (sym Symbol, found bool) {
	for _, sym = range table.Table {
		for _, p := range *sym.Positions() {
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
