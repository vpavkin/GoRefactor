package symbolTable

import (
	"go/ast"
	"container/vector"
)

//Represents an ast.Visitor, walking along ast.tree and registering all the types met
type TypeVisitor struct {
	Stb          *SymbolTableBuilder //pointer to parent SymbolTableBuilder
	Types        chan ITypeSymbol    //Channel, recieving new types from RegisterTypes goroutine
	Ack          chan int            //Channel, sendind acknowledments to RegisterTypes goroutine
	TypesVisited map[string]bool     //to avoid recursive loops while registering new type
}

/*^^TypeVisitor Methods^^*/

//A GoRoutine which recieves new types from TypeVisitor and registers them in symbolTable
//(replacing forward declarations)
func (tv TypeVisitor) RegisterTypes(CurrentSymbolTable *SymbolTable) {
	for {
		t := <-tv.Types
		if t == nil {
			break
		}
		for name, _ := range tv.TypesVisited {
			tv.TypesVisited[name] = false
		}
		tv.TypesVisited[t.Name()] = false

		CurrentSymbolTable.AddSymbol(t)

		CurrentSymbolTable.CheckFillType(t, tv.TypesVisited)
		tv.Ack <- 1
	}
	for _, sym := range CurrentSymbolTable.Table {
		tv.TypesVisited[sym.Name()] = false
	}
	for _, sym := range CurrentSymbolTable.Table {
		ts, _ := sym.(ITypeSymbol)

		// some structs were imported when pointers were created,so:
		if pt, ok := ts.(*PointerTypeSymbol); ok {
			if str, is := pt.IsStructPointer(); is {
				if pt.Fields.OpenedScopes.Len() == 0 {
					pt.Fields.AddOpenedScope(str.Fields)
				}
			}
		}
	}

	for _, sym := range CurrentSymbolTable.Table {
		OpenMethods(sym, tv.TypesVisited, CurrentSymbolTable)
	}
	tv.Ack <- 1
}

//ast.Visitor.Visit(). Looks for ast.TypeSpec nodes of ast.Tree to register new types
func (tv TypeVisitor) Visit(node interface{}) (w ast.Visitor) {
	if t, ok := node.(*ast.TypeSpec); ok {
		//ACK from RegisterTypes()
		<-tv.Ack
		ts := tv.Stb.BuildTypeSymbol(t.Type)

		switch ts.(type) {

		case *ImportedType, *PointerTypeSymbol, *ArrayTypeSymbol, *StructTypeSymbol, *InterfaceTypeSymbol, *MapTypeSymbol, *ChanTypeSymbol, *FunctionTypeSymbol, *TypeSymbol:
			if ts.Object() == nil {
				//No such symbol in CurrentSymbolTable
				ts.SetObject(CopyObject(t.Name))
				//ts.SetMethods(NewSymbolTable())
			} else {
				//There is an equal type symbol with different name => create alias
				ts = &AliasTypeSymbol{&TypeSymbol{Obj: CopyObject(t.Name), Meths: NewSymbolTable(), Posits: new(vector.Vector)}, ts}
			}
		}
		ts.Positions().Push(NewOccurence(t.Name.Pos(), ts.Object()))
		//Register type
		tv.Types <- ts
	}
	return tv
}

//Recuresively goes through each symbol of the table
//and replaces matching forward declarations with a given type symbol
func (st *SymbolTable) CheckFillType(newType ITypeSymbol, visited map[string]bool) {
	for _, sym := range st.Table {

		if _, ok := sym.(*ImportedType); ok {
			if sym.Name() == newType.Name() {
				if newType.Positions().Len() == 1 {
					newType.Positions().AppendVector(sym.Positions())
				}
				st.Table[sym.Name()] = newType
			}
		} else {
			//Start recursive walk
			CheckFillType(newType, sym, visited)
		}
	}
}

//Recuresively goes through each field of symbol "sym"
//and replaces matching forward declarations with a given type symbol
func CheckFillType(newType ITypeSymbol, sym Symbol, visited map[string]bool) {
	if sym == nil {
		return
	}
	if sym.Object() != nil {

		nn := sym.Name()

		if v, ok := visited[nn]; ok && v {
			//Symbol already checked
			return
		} else if ok {
			visited[nn] = true
		}
	}
	switch t := sym.(type) {
	case *AliasTypeSymbol:
		if _, ok := t.BaseType.(*ImportedType); ok {
			if t.BaseType.Name() == newType.Name() {
				if newType.Positions().Len() == 1 {
					newType.Positions().AppendVector(t.BaseType.Positions())
				}
				t.BaseType = newType
			}
		} else {
			CheckFillType(newType, t.BaseType, visited)
		}
	case *PointerTypeSymbol:
		if _, ok := t.BaseType.(*ImportedType); ok {
			if t.BaseType.Name() == newType.Name() {
				if newType.Positions().Len() == 1 {
					newType.Positions().AppendVector(t.BaseType.Positions())
				}
				t.BaseType = newType
			}
		} else {
			CheckFillType(newType, t.BaseType, visited)
		}
	case *ArrayTypeSymbol:
		if _, ok := t.ElemType.(*ImportedType); ok {
			if t.ElemType.Name() == newType.Name() {
				if newType.Positions().Len() == 1 {
					newType.Positions().AppendVector(t.ElemType.Positions())
				}
				t.ElemType = newType
			}
		} else {
			CheckFillType(newType, t.ElemType, visited)
		}
	case *StructTypeSymbol:
		t.Fields.CheckFillType(newType, visited)
	case *InterfaceTypeSymbol:
		t.Meths.CheckFillType(newType, visited)
	case *MapTypeSymbol:
		if _, ok := t.KeyType.(*ImportedType); ok {
			if t.KeyType.Name() == newType.Name() {
				if newType.Positions().Len() == 1 {
					newType.Positions().AppendVector(t.KeyType.Positions())
				}
				t.KeyType = newType
			}
		} else {
			CheckFillType(newType, t.KeyType, visited)
		}
		if _, ok := t.ValueType.(*ImportedType); ok {
			if t.ValueType.Name() == newType.Name() {
				if newType.Positions().Len() == 1 {
					newType.Positions().AppendVector(t.ValueType.Positions())
				}
				t.ValueType = newType
			}
		} else {
			CheckFillType(newType, t.ValueType, visited)
		}
	case *ChanTypeSymbol:
		if _, ok := t.ValueType.(*ImportedType); ok {
			if t.ValueType.Name() == newType.Name() {
				if newType.Positions().Len() == 1 {
					newType.Positions().AppendVector(t.ValueType.Positions())
				}
				t.ValueType = newType
			}
		} else {
			CheckFillType(newType, t.ValueType, visited)
		}
	case *FunctionTypeSymbol:
		t.Parameters.CheckFillType(newType, visited)
		t.Results.CheckFillType(newType, visited)
		t.Reciever.CheckFillType(newType, visited)
	case *VariableSymbol:
		if _, ok := t.VariableType.(*ImportedType); ok {
			if t.VariableType.Name() == newType.Name() {
				if newType.Positions().Len() == 1 {
					newType.Positions().AppendVector(t.VariableType.Positions())
				}
				t.VariableType = newType
			}
		} else {
			CheckFillType(newType, t.VariableType, visited)
		}
	case *FunctionSymbol:
		if _, ok := t.FunctionType.(*ImportedType); ok {
			if t.FunctionType.Name() == newType.Name() {
				if newType.Positions().Len() == 1 {
					newType.Positions().AppendVector(t.FunctionType.Positions())
				}
				t.FunctionType = newType
			}
		} else {
			CheckFillType(newType, t.FunctionType, visited)
		}
	}
}

func OpenMethods(sym Symbol, visited map[string]bool, CurrentSymbolTable *SymbolTable) {

	nn := sym.Name()
	if v, ok := visited[nn]; ok && v {
		//Symbol already checked
		return
	} else if ok {
		visited[nn] = true
	}
	switch t := sym.(type) {
	case *ArrayTypeSymbol:
		OpenMethods(t.ElemType, visited, CurrentSymbolTable)
	case *ChanTypeSymbol:
		OpenMethods(t.ValueType, visited, CurrentSymbolTable)
	case *FunctionTypeSymbol:
		for _, variable := range t.Parameters.Table {
			v := variable.(*VariableSymbol)
			OpenMethods(v.VariableType, visited, CurrentSymbolTable)
		}
		for _, variable := range t.Results.Table {
			v := variable.(*VariableSymbol)
			OpenMethods(v.VariableType, visited, CurrentSymbolTable)
		}
	case *MapTypeSymbol:
		OpenMethods(t.KeyType, visited, CurrentSymbolTable)
		OpenMethods(t.ValueType, visited, CurrentSymbolTable)
	case *InterfaceTypeSymbol:
		for _, sym := range t.Meths.Table {
			if _, ok := sym.(*FunctionSymbol); !ok {

				if t.Methods() == nil {
					t.SetMethods(NewSymbolTable())
				}
				ts := sym.(ITypeSymbol)
				t.Meths.AddOpenedScope(ts.Methods())
				t.Meths.Table[sym.Name()] = nil, false
			}
			OpenMethods(sym, visited, CurrentSymbolTable)
		}
	case *StructTypeSymbol:
		for _, variable := range t.Fields.Table {
			if _, ok := variable.(*VariableSymbol); !ok {
				//Embedded struct
				if t.Methods() == nil {
					//Anonymous struct
					t.SetMethods(NewSymbolTable())
				}
				ts := variable.(ITypeSymbol)
				t.Methods().AddOpenedScope(ts.Methods())
				if st, ok := ts.(*StructTypeSymbol); ok {
					t.Fields.AddOpenedScope(st.Fields)
				}
				if pt, ok := ts.(*PointerTypeSymbol); ok {
					if st, is := pt.IsStructPointer(); is {
						t.Fields.AddOpenedScope(st.Fields)
					}
				}

				typeVar := &VariableSymbol{ts.Object(), ts, ts.Positions()}

				t.Fields.Table[variable.Name()] = nil, false
				t.Fields.Table[variable.Object().Name] = typeVar

				OpenMethods(variable, visited, CurrentSymbolTable)
			}
		}
	case *PointerTypeSymbol:
		if t.Meths == nil {
			t.Meths = NewSymbolTable()
		}
		if t.BaseType.Methods() == nil {
			t.BaseType.SetMethods(NewSymbolTable())
		}
		t.Meths.AddOpenedScope(t.BaseType.Methods())
		OpenMethods(t.BaseType, visited, CurrentSymbolTable)
	}

}
