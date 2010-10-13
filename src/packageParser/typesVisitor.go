package packageParser

import (
	"go/ast"
	"container/vector"
	"st"
)

import "fmt"


var visited map[string]bool
//Represents an ast.Visitor, walking along ast.tree and registering all the types met
type typesVisitor struct {
	Parser *packageParser
}


//ast.Visitor.Visit(). Looks for ast.TypeSpec nodes of ast.Tree to register new types
func (tv *typesVisitor) Visit(node interface{}) (w ast.Visitor) {
	if tsp, ok := node.(*ast.TypeSpec); ok {
		ts := tv.Parser.parseTypeSymbol(tsp.Type)

		tsp.Name.Obj = &ast.Object{Kind: ast.Typ, Name: tsp.Name.Name}

		if ts.Object() == nil {
			//No such symbol in CurrentSymbolTable
			ts.SetObject(tsp.Name.Obj)
		} else {
			//There is an equal type symbol with different name => create alias
			ts = &st.AliasTypeSymbol{&st.TypeSymbol{Obj: tsp.Name.Obj, Posits: new(vector.Vector)}, ts}
		}

		ts.AddPosition(st.NewOccurence(tsp.Name.Pos()))
		tv.Parser.RootSymbolTable.AddSymbol(ts)

		//fmt.Printf("parsed: " + ts.String());
	}
	return tv
}

func (pp *packageParser) fixRootTypes() {

	visited = make(map[string]bool)
	pp.fixTypesInSymbolTable(pp.RootSymbolTable)
}

func (pp *packageParser) fixTypesInSymbolTable(table *st.SymbolTable) {
	if table == nil {
		return
	}
	for _, sym := range table.Table {
		if uts, ok := sym.(*st.UnresolvedTypeSymbol); ok {
			if newType, ok := pp.RootSymbolTable.LookUp(sym.Name(),uts.Declaration.Pos().Filename); ok {
				newType.Positions().AppendVector(sym.Positions())
				//rewrites sym because names are identical
				fmt.Printf("rewrited %v with %v\n", sym.Name(), newType.Name())
				table.AddSymbol(newType)
			}
		} else {
			//Start recursive walk
			//fmt.Printf("fixing %v\n",sym.Name());
			pp.fixType(sym)
			//fmt.Printf("fixing %v finished\n",sym.Name());
		}
	}
}

func (pp *packageParser) fixAliasTypeSymbol(t *st.AliasTypeSymbol) {
	if _, ok := t.BaseType.(*st.UnresolvedTypeSymbol); ok {
		if newType, ok := pp.RootSymbolTable.LookUp(t.BaseType.Name(),uts.Declaration.Pos().Filename); ok {
			newType.Positions().AppendVector(t.BaseType.Positions())
			t.BaseType = newType.(st.ITypeSymbol)
		}
	} else {
		pp.fixType(t.BaseType)
	}
}

func (pp *packageParser) fixPointerTypeSymbol(t *st.PointerTypeSymbol) {
	if _, ok := t.BaseType.(*st.UnresolvedTypeSymbol); ok {
		if newType, ok := pp.RootSymbolTable.LookUp(t.BaseType.Name(),uts.Declaration.Pos().Filename); ok {
			newType.Positions().AppendVector(t.BaseType.Positions())
			t.BaseType = newType.(st.ITypeSymbol)
		}
	} else {
		pp.fixType(t.BaseType)
	}
}

func (pp *packageParser) fixArrayTypeSymbol(t *st.ArrayTypeSymbol) {
	if _, ok := t.ElemType.(*st.UnresolvedTypeSymbol); ok {
		if newType, ok := pp.RootSymbolTable.LookUp(t.ElemType.Name(),uts.Declaration.Pos().Filename); ok {
			newType.Positions().AppendVector(t.ElemType.Positions())
			t.ElemType = newType.(st.ITypeSymbol)
		}
	} else {
		pp.fixType(t.ElemType)
	}
}

func (pp *packageParser) fixStructTypeSymbol(t *st.StructTypeSymbol) {
	pp.fixTypesInSymbolTable(t.Fields)
}

func (pp *packageParser) fixInterfaceTypeSymbol(t *st.InterfaceTypeSymbol) {
	pp.fixTypesInSymbolTable(t.Meths)
}

func (pp *packageParser) fixMapTypeSymbol(t *st.MapTypeSymbol) {
	if _, ok := t.KeyType.(*st.UnresolvedTypeSymbol); ok {
		if newType, ok := pp.RootSymbolTable.LookUp(t.KeyType.Name(),uts.Declaration.Pos().Filename); ok {
			newType.Positions().AppendVector(t.KeyType.Positions())
			t.KeyType = newType.(st.ITypeSymbol)
		}
	} else {
		pp.fixType(t.KeyType)
	}

	if _, ok := t.ValueType.(*st.UnresolvedTypeSymbol); ok {
		if newType, ok := pp.RootSymbolTable.LookUp(t.ValueType.Name(),uts.Declaration.Pos().Filename); ok {
			newType.Positions().AppendVector(t.ValueType.Positions())
			t.ValueType = newType.(st.ITypeSymbol)
		}
	} else {
		pp.fixType(t.ValueType)
	}
}

func (pp *packageParser) fixChanTypeSymbol(t *st.ChanTypeSymbol) {
	if _, ok := t.ValueType.(*st.UnresolvedTypeSymbol); ok {
		if newType, ok := pp.RootSymbolTable.LookUp(t.ValueType.Name(),uts.Declaration.Pos().Filename); ok {
			newType.Positions().AppendVector(t.ValueType.Positions())
			t.ValueType = newType.(st.ITypeSymbol)
		}
	} else {
		pp.fixType(t.ValueType)
	}
}

func (pp *packageParser) fixFunctionTypeSymbol(t *st.FunctionTypeSymbol) {
	pp.fixTypesInSymbolTable(t.Parameters)
	pp.fixTypesInSymbolTable(t.Results)
	//fixTypesInSymbolTable(t.Reciever)
}
func (pp *packageParser) fixVariableSymbol(t *st.VariableSymbol) {
	if _, ok := t.VariableType.(*st.UnresolvedTypeSymbol); ok {
		if newType, ok := pp.RootSymbolTable.LookUp(t.VariableType.Name(),uts.Declaration.Pos().Filename); ok {
			newType.Positions().AppendVector(t.VariableType.Positions())
			t.VariableType = newType.(st.ITypeSymbol)
		}
	} else {
		pp.fixType(t.VariableType)
	}
}
func (pp *packageParser) fixFunctionSymbol(t *st.FunctionSymbol) {
	if _, ok := t.FunctionType.(*st.UnresolvedTypeSymbol); ok {
		if newType, ok := pp.RootSymbolTable.LookUp(t.FunctionType.Name(),uts.Declaration.Pos().Filename); ok {
			newType.Positions().AppendVector(t.FunctionType.Positions())
			t.FunctionType = newType.(st.ITypeSymbol)
		}
	} else {
		pp.fixType(t.FunctionType)
	}
}

//Fixes Type and its' subtypes recursively
func (pp *packageParser) fixType(sym st.Symbol) {

	if sym == nil {
		fmt.Printf("ERROR: sym == nil. func fixType, typesVisitor.go")
	}
	if st.IsPredeclaredIdentifier(sym.Name()) {
		return
	}
	fmt.Printf("fixing %v %T\n", sym.Name(), sym)
	if checkIsVisited(sym) {
		return
	}

	switch t := sym.(type) {
	case *st.AliasTypeSymbol:
		pp.fixAliasTypeSymbol(t)
	case *st.PointerTypeSymbol:
		pp.fixPointerTypeSymbol(t)
	case *st.ArrayTypeSymbol:
		pp.fixArrayTypeSymbol(t)
	case *st.StructTypeSymbol:
		pp.fixStructTypeSymbol(t)
	case *st.InterfaceTypeSymbol:
		pp.fixInterfaceTypeSymbol(t)
	case *st.MapTypeSymbol:
		pp.fixMapTypeSymbol(t)
	case *st.ChanTypeSymbol:
		pp.fixChanTypeSymbol(t)
	case *st.FunctionTypeSymbol:
		pp.fixFunctionTypeSymbol(t)
	case *st.VariableSymbol:
		pp.fixVariableSymbol(t)
	case *st.FunctionSymbol:
		pp.fixFunctionSymbol(t)
	}
}

func (pp *packageParser) openFields(sym st.Symbol) {

	if checkIsVisited(sym) {
		return
	}
	if st.IsPredeclaredIdentifier(sym.Name()) {
		return
	}

	switch t := sym.(type) {
	case *st.ArrayTypeSymbol:
		pp.openFields(t.ElemType)
	case *st.ChanTypeSymbol:
		pp.openFields(t.ValueType)
	case *st.MapTypeSymbol:
		pp.openFields(t.KeyType)
		pp.openFields(t.ValueType)
	case *st.FunctionTypeSymbol:
		if t.Parameters != nil {
			for _, variable := range t.Parameters.Table {
				v := variable.(*st.VariableSymbol)
				pp.openFields(v.VariableType)
			}
		}
		if t.Results != nil {
			for _, variable := range t.Results.Table {
				v := variable.(*st.VariableSymbol)
				pp.openFields(v.VariableType)
			}
		}
	case *st.InterfaceTypeSymbol:
		if t.Meths != nil {
			for _, sym := range t.Meths.Table {
				if _, ok := sym.(*st.FunctionSymbol); !ok {
					//EmbeddedInterface
					ts := sym.(*st.InterfaceTypeSymbol)
					t.Meths.AddOpenedScope(ts.Meths)
					//Delete now useles functionSymbol from interface
					t.Meths.Table[sym.Name()] = nil, false
				}
				pp.openFields(sym)
			}
		}
	case *st.StructTypeSymbol:
		for _, variable := range t.Fields.Table {
			if _, ok := variable.(*st.VariableSymbol); !ok {
				//Embedded struct
				ts := variable.(st.ITypeSymbol)
				if table, ok := ts.(*st.StructTypeSymbol); ok {
					t.Fields.AddOpenedScope(table.Fields)
				}
				if pt, ok := ts.(*st.PointerTypeSymbol); ok {
					if table, ok1 := pt.GetBaseStruct(); ok1 {
						t.Fields.AddOpenedScope(table.Fields)
					}
				}
				//still need type thumb, for openMethods

				pp.openFields(variable) //it's a type
			}
		}
	case *st.PointerTypeSymbol:
		if table, ok := t.GetBaseStruct(); ok {
			t.Fields = pp.registerNewSymbolTable()
			t.Fields.AddOpenedScope(table.Fields)
		}
		pp.openFields(t.BaseType)
	}
}

func checkIsVisited(sym st.Symbol) bool {
	symName := sym.Name()
	if v, ok := visited[symName]; ok && v { //Symbol already checked
		return true
	} else if symName != "" { //Mark as checked
		visited[symName] = true
	}
	return false
}
