package st

import (
	"container/vector"
)

import "fmt"
import "sort"


//Represents a local SymbolTable with a number of opened scopes
type SymbolTable struct {
	Table        *vector.Vector //symbol table
	OpenedScopes *vector.Vector //vector of opened scopes
	Package      *Package       //package that table belongs to

	lookUpIn chan struct {
		name     string
		fileName string
	}
	lookUpOut chan Symbol

	addIn chan Symbol

	replaceIn chan struct {
		replace string
		with    Symbol
	}

	removeIn chan string

	iterIn  chan func(sym Symbol)
	iterOut chan bool

	lookUpPointerIn chan struct {
		name  string
		depth int
	}
	lookUpPointerOut chan *PointerTypeSymbol

	addOpenedScopeIn chan *SymbolTable

	iterOpenedScopesIn  chan func(scope *SymbolTable)
	iterOpenedScopesOut chan bool
}

/*^^SymbolTable Methods and Functions^^*/
func (table *SymbolTable) mainCycle() {
	for {
		select {
		case str := <-table.lookUpIn:
			table.lookUpOut <- table.lookUp(str.name, str.fileName)
		case str := <-table.lookUpPointerIn:
			table.lookUpPointerOut <- table.lookUpPointerType(str.name, str.depth)
		case sym := <-table.addIn:
			table.addSymbol(sym)
		case st := <-table.addOpenedScopeIn:
			table.addOpenedScope(st)
		case str := <-table.replaceIn:
			table.replaceSymbol(str.replace, str.with)
		case name := <-table.removeIn:
			table.removeSymbol(name)
		case toDo := <-table.iterOpenedScopesIn:
			table.forEachOpenedScope(toDo)
			table.iterOpenedScopesOut <- true
		case toDo := <-table.iterIn:
			table.forEach(toDo)
			table.iterOut <- true
		}
	}
}

//Creates a new empty(but ready for work) SymbolTable and returns pointer to it
func NewSymbolTable(p *Package) *SymbolTable {
	st := &SymbolTable{Table: new(vector.Vector), OpenedScopes: new(vector.Vector), Package: p}

	st.lookUpIn = make(chan struct {
		name     string
		fileName string
	})
	st.lookUpOut = make(chan Symbol)

	st.addIn = make(chan Symbol)

	st.replaceIn = make(chan struct {
		replace string
		with    Symbol
	})

	st.removeIn = make(chan string)

	st.iterIn = make(chan func(sym Symbol))
	st.iterOut = make(chan bool)

	st.lookUpPointerIn = make(chan struct {
		name  string
		depth int
	})
	st.lookUpPointerOut = make(chan *PointerTypeSymbol)

	st.addOpenedScopeIn = make(chan *SymbolTable)

	st.iterOpenedScopesIn = make(chan func(scope *SymbolTable))
	st.iterOpenedScopesOut = make(chan bool)

	go st.mainCycle()
	return st
}

func (table *SymbolTable) ForEach(toDo func(sym Symbol)) {
	table.iterIn <- toDo
	<-table.iterOut
}

func (table *SymbolTable) forEach(toDo func(sym Symbol)) {
	for i := 0; i < len(*table.Table); i++ {
		sym := table.Table.At(i).(Symbol)
		toDo(sym)
	}
}

//use only in blocked mode
func (table *SymbolTable) forEachStoppable(toDo func(sym Symbol) bool) (Symbol, bool) {
	for i := 0; i < len(*table.Table); i++ {
		sym := table.Table.At(i).(Symbol)
		if done := toDo(sym); done {
			return sym, true
		}
	}
	return nil, false
}
//use only in blocked mode
func (table *SymbolTable) forEachStoppableReverse(toDo func(sym Symbol) bool) (Symbol, bool) {
	for i := len(*table.Table) - 1; i >= 0; i-- {
		sym := table.Table.At(i).(Symbol)
		if done := toDo(sym); done {
			return sym, true
		}
	}
	return nil, false
}

func (table *SymbolTable) ForEachOpenedScope(toDo func(scope *SymbolTable)) {
	table.iterOpenedScopesIn <- toDo
	<-table.iterOpenedScopesOut
}

func (table *SymbolTable) forEachOpenedScope(toDo func(scope *SymbolTable)) {
	for i := 0; i < len(*table.OpenedScopes); i++ {
		scope := table.OpenedScopes.At(i).(*SymbolTable)
		toDo(scope)
	}
}


//Adds a scope to opened scopes list
func (table *SymbolTable) AddOpenedScope(scope *SymbolTable) {
	if scope == nil {
		panic("Invalid argument! Argument must not be nil")
		return
	}
	table.addOpenedScopeIn <- scope
}

func (table *SymbolTable) addOpenedScope(scope *SymbolTable) {
	table.OpenedScopes.Push(scope)
}

//Adds a symbol to local symbol table
func (table *SymbolTable) AddSymbol(sym Symbol) bool {
	if sym == nil {
		panic("Invalid argument! Argument must not be nil")
		return false
	}
	if _, ok := sym.(Symbol); !ok {
		panic("Invalid argument! Argument must implement Symbol interface")
		return false
	}
	table.addIn <- sym

	return true
}

func (table *SymbolTable) addSymbol(sym Symbol) {
	table.Table.Push(sym) //since LookUp goes in reverse order, the latest symbol will be find earlier if there's two identicaly named symbols
}
func (table *SymbolTable) ReplaceSymbol(replace string, with Symbol) {
	if with == nil {
		panic("Invalid argument! Argument must not be nil")
		return
	}
	table.replaceIn <- struct {
		replace string
		with    Symbol
	}{replace, with}

}
func (table *SymbolTable) replaceSymbol(replace string, with Symbol) {

	for i := 0; i < len(*table.Table); i++ {
		sym := table.Table.At(i).(Symbol)
		if sym.Name() == replace {
			fmt.Printf("replaced %s with %s\n", table.Table.At(i).(Symbol).Name(), with.Name())
			table.Table.Set(i, with)
		}
	}
}
// removes last symbol with given name
func (table *SymbolTable) RemoveSymbol(name string) {

	table.removeIn <- name
}
func (table *SymbolTable) removeSymbol(name string) {

	j := 0
	l := len(*table.Table)
	for i := 0; i < l-j; i++ {

		sym := table.Table.At(i).(Symbol)
		if sym.Name() == name {
			fmt.Printf("removed %s\n", table.Table.At(i).(Symbol).Name())
			table.Table.Delete(i)
			i--
			j++
		}
	}
}

func (table *SymbolTable) Iter() <-chan Symbol {
	c := make(chan Symbol)

	go func() {
		for i := 0; i < len(*table.Table); i++ {
			c <- table.Table.At(i).(Symbol)
		}
		if !closed(c) {
			close(c)
		}
	}()
	return c
}

func (table *SymbolTable) IterReverse() <-chan Symbol {
	c := make(chan Symbol)

	go func() {
		for i := len(*table.Table) - 1; i >= 0; i-- {
			c <- table.Table.At(i).(Symbol)
		}
		if !closed(c) {
			close(c)
		}
	}()
	return c
}

//Searches symbol table and it's opened scopes for a symbol with a given name
func (table *SymbolTable) LookUp(name string, fileName string) (Symbol, bool) {

	table.lookUpIn <- struct {
		name     string
		fileName string
	}{name, fileName}
	sym := <-table.lookUpOut
	if sym != nil {
		return sym, true
	}
	return nil, false
}
func (table *SymbolTable) lookUp(name string, fileName string) (sym Symbol) {

	var found bool

	if sym, found = table.forEachStoppableReverse(func(s Symbol) bool { return s.Name() == name }); found {
		return sym
	}

	if !found {
		for _, x := range *table.OpenedScopes {
			v, _ := x.(*SymbolTable)
			if sym = v.lookUp(name, fileName); sym != nil {
				return sym
			}
		}
		if table.Package != nil {
			if imps, ok := table.Package.Imports[fileName]; ok && imps != nil {
				for _, e := range *imps {
					ps := e.(*PackageSymbol)
					if name == ps.Obj.Name {
						return ps
					}
				}
			}
		}
	}

	return nil
}

//Searches symbol table and it's opened scopes for a pointer type
//with a specified base type and depth
func (table *SymbolTable) LookUpPointerType(name string, depth int) (sym *PointerTypeSymbol, found bool) {

	table.lookUpPointerIn <- struct {
		name  string
		depth int
	}{name, depth}
	sym = <-table.lookUpPointerOut
	if sym != nil {
		return sym, true
	}
	return nil, false
}

func (table *SymbolTable) lookUpPointerType(name string, depth int) (psym *PointerTypeSymbol) {

	var found bool
	var sym Symbol
	if sym, found = table.forEachStoppableReverse(func(ss Symbol) bool {
		if s, ok := ss.(*PointerTypeSymbol); ok {
			return (s.BaseName() == name && s.Depth() == depth)
		}
		return false
	}); found {
		return sym.(*PointerTypeSymbol)
	}

	for _, x := range *table.OpenedScopes {
		v, _ := x.(*SymbolTable)
		if sym = v.lookUpPointerType(name, depth); sym != nil {
			return sym.(*PointerTypeSymbol)
		}
	}
	return
}

func (table *SymbolTable) FindTypeSwitchVar() (*VariableSymbol, bool) {
	c := table.IterReverse()
	for ss := range c {
		if sym, ok := ss.(*VariableSymbol); ok && sym.Obj.Kind == -1 {
			close(c)
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

func (table *SymbolTable) String() *vector.StringVector {

	var res = new(vector.StringVector)
	var s = new(vector.StringVector)

	for sym := range table.Iter() {
		if _, ok := sym.(*PackageSymbol); ok {
			s.Push("   package " + sym.String() + "\n")
		}
	}

	sort.Sort(s)
	s.Insert(0, "packages:\n")
	res.AppendVector(s)

	s = new(vector.StringVector)
	for sym := range table.Iter() {
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
	for sym := range table.Iter() {
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
	for sym := range table.Iter() {
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


func (table *SymbolTable) FindSymbolByPosition(fileName string, line int, column int) (sym Symbol, found bool) {
	c := table.IterReverse()
	for sym := range c {
		for _, p := range *sym.Positions() {
			pos := p.(Occurence).Pos
			if pos.Filename == fileName && pos.Line == line && pos.Column == column {
				found = true
				close(c)
				return
			}
		}
	}
	sym, found = nil, false
	return
}
