package packageParser

import (
	"go/ast"
	"container/vector"
	"st"
)

import "fmt"


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
			ts = &st.AliasTypeSymbol{&st.TypeSymbol{Obj: tsp.Name.Obj, Posits: new(vector.Vector),PackFrom:tv.Parser.Package}, ts}
		}

		ts.AddPosition(st.NewOccurence(tsp.Name.Pos()))
		tv.Parser.RootSymbolTable.AddSymbol(ts)

		//fmt.Printf("parsed: " + ts.String());
	}
	return tv
}

func (pp *packageParser) fixRootTypes() {

	pp.visited = make(map[string]bool)
	pp.fixTypesInSymbolTable(pp.RootSymbolTable)
}

func (pp *packageParser) fixTypesInSymbolTable(table *st.SymbolTable) {
	if table == nil {
		return
	}
	for sym := range table.Iter() {
		if uts, ok := sym.(*st.UnresolvedTypeSymbol); ok {
			pp.CurrentFileName = uts.Declaration.Pos().Filename; 
			table.AddSymbol(pp.parseTypeSymbol(uts.Declaration));
			fmt.Printf("rewrited %v \n", sym.Name())
		} else {
			//Start recursive walk
			pp.fixType(sym)
		}
	}
}

func (pp *packageParser) fixAliasTypeSymbol(t *st.AliasTypeSymbol) {
	if uts, ok := t.BaseType.(*st.UnresolvedTypeSymbol); ok {
		pp.CurrentFileName = uts.Declaration.Pos().Filename; 
		t.BaseType = pp.parseTypeSymbol(uts.Declaration);
	} else {
		pp.fixType(t.BaseType)
	}
}

func (pp *packageParser) fixPointerTypeSymbol(t *st.PointerTypeSymbol) {
	if uts, ok := t.BaseType.(*st.UnresolvedTypeSymbol); ok {
		pp.CurrentFileName = uts.Declaration.Pos().Filename; 
		t.BaseType = pp.parseTypeSymbol(uts.Declaration);
		fmt.Printf("%s %s set to %s\n",pp.Package.AstPackage.Name,uts.Name(),t.BaseType.Name());
	} else {
		pp.fixType(t.BaseType)
	}
}

func (pp *packageParser) fixArrayTypeSymbol(t *st.ArrayTypeSymbol) {
	if uts, ok := t.ElemType.(*st.UnresolvedTypeSymbol); ok {
		pp.CurrentFileName = uts.Declaration.Pos().Filename; 
		t.ElemType = pp.parseTypeSymbol(uts.Declaration);
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
	if uts, ok := t.KeyType.(*st.UnresolvedTypeSymbol); ok {
		pp.CurrentFileName = uts.Declaration.Pos().Filename; 
		t.KeyType = pp.parseTypeSymbol(uts.Declaration);
	} else {
		pp.fixType(t.KeyType)
	}

	if uts, ok := t.ValueType.(*st.UnresolvedTypeSymbol); ok {
		pp.CurrentFileName = uts.Declaration.Pos().Filename; 
		t.ValueType = pp.parseTypeSymbol(uts.Declaration);
	} else {
		pp.fixType(t.ValueType)
	}
}

func (pp *packageParser) fixChanTypeSymbol(t *st.ChanTypeSymbol) {
	if uts, ok := t.ValueType.(*st.UnresolvedTypeSymbol); ok {
		pp.CurrentFileName = uts.Declaration.Pos().Filename; 
		t.ValueType = pp.parseTypeSymbol(uts.Declaration);
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
	fmt.Printf("%s %s has type %T\n",pp.Package.AstPackage.Name,t.VariableType.Name(),t.VariableType);
	if uts, ok := t.VariableType.(*st.UnresolvedTypeSymbol); ok {
		pp.CurrentFileName = uts.Declaration.Pos().Filename; 
		t.VariableType = pp.parseTypeSymbol(uts.Declaration);
	} else {
		pp.fixType(t.VariableType)
	}
}
func (pp *packageParser) fixFunctionSymbol(t *st.FunctionSymbol) {
	if uts, ok := t.FunctionType.(*st.UnresolvedTypeSymbol); ok {
		pp.CurrentFileName = uts.Declaration.Pos().Filename; 
		t.FunctionType = pp.parseTypeSymbol(uts.Declaration);
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
	
	if(sym.PackageFrom() != pp.Package){
		return;
	}
	if pp.checkIsVisited(sym) {
		return
	}
	
	fmt.Printf("%s fixing %v %T\n",pp.Package.AstPackage.Name, sym.Name(), sym)
	

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

	if pp.checkIsVisited(sym) {
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
			for variable := range t.Parameters.Iter() {
				v := variable.(*st.VariableSymbol)
				pp.openFields(v.VariableType)
			}
		}
		if t.Results != nil {
			for variable := range t.Results.Iter() {
				v := variable.(*st.VariableSymbol)
				pp.openFields(v.VariableType)
			}
		}
	case *st.InterfaceTypeSymbol:
		if t.Meths != nil {
			for sym := range t.Meths.Iter() {
				if _, ok := sym.(*st.FunctionSymbol); !ok {
					//EmbeddedInterface
					ts := sym.(*st.InterfaceTypeSymbol)
					t.Meths.AddOpenedScope(ts.Meths)
					//Delete functionSymbol which is now useles from interface
					t.Meths.RemoveSymbol(sym.Name())
				}
				pp.openFields(sym)
			}
		}
	case *st.StructTypeSymbol:
		for variable := range t.Fields.Iter() {
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

func (pp *packageParser)checkIsVisited(sym st.Symbol) bool {
	if _,ok := sym.(st.ITypeSymbol);!ok{
		return false;
	}
	symName := sym.Name()
	if v, ok := pp.visited[symName]; ok && v { //Symbol already checked
		return true
	} else if _,ok := sym.(st.ITypeSymbol); symName!="" && ok { //Mark as checked
		pp.visited[symName] = true
	}
	return false
}
