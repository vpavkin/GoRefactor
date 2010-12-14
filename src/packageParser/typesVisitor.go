package packageParser

import (
	"go/ast"
	//"go/token"
	"st"
	//"strings"
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

		if ts.Name() == st.NO_NAME {
			//No such symbol in CurrentSymbolTable
			ts.SetName(tsp.Name.Name)
		} else {
			//There is an equal type symbol with different name => create alias
			ts = st.MakeAliasType(tsp.Name.Name,tv.Parser.Package, ts)
		}
		
		tv.Parser.registerIdent(ts,tsp.Name);
				
		tv.Parser.RootSymbolTable.AddSymbol(ts)
	
	}
	return tv
}

func (pp *packageParser) resolveType(uts *st.UnresolvedTypeSymbol) (result st.ITypeSymbol) {
	var res st.Symbol
	var found bool

	if uts.PackageFrom() != pp.Package {

		if res, found = uts.PackageFrom().Symbols.LookUp(uts.Name(), ""); !found {
			panic("symbol " + uts.PackageFrom().AstPackage.Name + "." + uts.Name() + " unresolved")
		}
	} else if res, found = pp.RootSymbolTable.LookUp(uts.Name(), ""); !found {
		for _, pos := range uts.Positions() {
			panic("symbol" + uts.Name() + "at " + pos.String() + " unresolved")
		}
	}
	return res.(st.ITypeSymbol)
}
func (pp *packageParser) moveData(resolvedType st.ITypeSymbol, unresType st.ITypeSymbol) {
	for ident,_ := range unresType.Identifiers() {
		resolvedType.AddIdent(ident)
		pp.IdentMap.AddIdent(ident,resolvedType)
	}
	for _, pos := range unresType.Positions() {
		resolvedType.AddPosition(pos)
	}
}
func (pp *packageParser) fixRootTypes() {

	pp.visited = make(map[string]bool)
	pp.fixTypesInSymbolTable(pp.RootSymbolTable)
}

func (pp *packageParser) fixTypesInSymbolTable(table *st.SymbolTable) {
	if table == nil {
		return
	}
	table.ForEachNoLock(func(sym st.Symbol) {

		if uts, ok := sym.(*st.UnresolvedTypeSymbol); ok {

			res := pp.resolveType(uts)
			table.ReplaceSymbol(uts.Name(), res)
			pp.moveData(res, uts)
			fmt.Printf("rewrited %s with %s from %s \n", sym.Name(), res.Name(), res.PackageFrom().AstPackage.Name)
			pp.fixType(res)
		} else {
			//Start recursive walk
			pp.fixType(sym)
		}
	})
}

func (pp *packageParser) fixAliasTypeSymbol(t *st.AliasTypeSymbol) {
	if uts, ok := t.BaseType.(*st.UnresolvedTypeSymbol); ok {
		pp.CurrentFileName = pp.Package.FileSet.Position(uts.Declaration.Pos()).Filename
		t.BaseType = pp.resolveType(uts)
		pp.moveData(t.BaseType, uts)
	} else {
		pp.fixType(t.BaseType)
	}
}

func (pp *packageParser) fixPointerTypeSymbol(t *st.PointerTypeSymbol) {
	if uts, ok := t.BaseType.(*st.UnresolvedTypeSymbol); ok {
		pp.CurrentFileName = pp.Package.FileSet.Position(uts.Declaration.Pos()).Filename
		t.BaseType = pp.resolveType(uts)
		pp.moveData(t.BaseType, uts)
	} else {
		pp.fixType(t.BaseType)
	}
}

func (pp *packageParser) fixArrayTypeSymbol(t *st.ArrayTypeSymbol) {
	if uts, ok := t.ElemType.(*st.UnresolvedTypeSymbol); ok {
		pp.CurrentFileName = pp.Package.FileSet.Position(uts.Declaration.Pos()).Filename
		t.ElemType = pp.resolveType(uts)
		pp.moveData(t.ElemType, uts)
	} else {
		pp.fixType(t.ElemType)
	}
}

func (pp *packageParser) fixStructTypeSymbol(t *st.StructTypeSymbol) {
	pp.fixTypesInSymbolTable(t.Fields)
}

func (pp *packageParser) fixInterfaceTypeSymbol(t *st.InterfaceTypeSymbol) {
	if t.Name() == "Error" {
		fmt.Printf("YEAHH os.Error:\n %s", *t.Methods().String())
	}
	pp.fixTypesInSymbolTable(t.Methods())
}

func (pp *packageParser) fixMapTypeSymbol(t *st.MapTypeSymbol) {
	if uts, ok := t.KeyType.(*st.UnresolvedTypeSymbol); ok {
		pp.CurrentFileName = pp.Package.FileSet.Position(uts.Declaration.Pos()).Filename
		t.KeyType = pp.resolveType(uts)
		pp.moveData(t.KeyType, uts)
	} else {
		pp.fixType(t.KeyType)
	}

	if uts, ok := t.ValueType.(*st.UnresolvedTypeSymbol); ok {
		pp.CurrentFileName = pp.Package.FileSet.Position(uts.Declaration.Pos()).Filename
		t.ValueType = pp.resolveType(uts)
		pp.moveData(t.ValueType, uts)
	} else {
		pp.fixType(t.ValueType)
	}
}

func (pp *packageParser) fixChanTypeSymbol(t *st.ChanTypeSymbol) {
	if uts, ok := t.ValueType.(*st.UnresolvedTypeSymbol); ok {
		pp.CurrentFileName = pp.Package.FileSet.Position(uts.Declaration.Pos()).Filename
		t.ValueType = pp.resolveType(uts)
		pp.moveData(t.ValueType, uts)
	} else {
		pp.fixType(t.ValueType)
	}
}

func (pp *packageParser) fixFunctionTypeSymbol(t *st.FunctionTypeSymbol) {
	pp.fixTypesInSymbolTable(t.Parameters)
	pp.fixTypesInSymbolTable(t.Results)
	pp.fixTypesInSymbolTable(t.Reciever)
}
func (pp *packageParser) fixVariableSymbol(t *st.VariableSymbol) {
	//fmt.Printf("%s %s has type %T\n",pp.Package.AstPackage.Name,t.VariableType.Name(),t.VariableType);

	if uts, ok := t.VariableType.(*st.UnresolvedTypeSymbol); ok {
		pp.CurrentFileName = pp.Package.FileSet.Position(uts.Declaration.Pos()).Filename
		t.VariableType = pp.resolveType(uts)
		pp.moveData(t.VariableType, uts)

	} else {
		pp.fixType(t.VariableType)
	}
}
func (pp *packageParser) fixFunctionSymbol(t *st.FunctionSymbol) {
	if uts, ok := t.FunctionType.(*st.UnresolvedTypeSymbol); ok {
		pp.CurrentFileName = pp.Package.FileSet.Position(uts.Declaration.Pos()).Filename
		t.FunctionType = pp.resolveType(uts)
		pp.moveData(t.FunctionType, uts)
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

	if sym.PackageFrom() != pp.Package {
		return
	}
	if pp.checkIsVisited(sym) {
		return
	}

	fmt.Printf("%s fixing %v %T\n", pp.Package.AstPackage.Name, sym.Name(), sym)

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


func (pp *packageParser) checkIsVisited(sym st.Symbol) bool {
	if _, ok := sym.(st.ITypeSymbol); !ok {
		return false
	}
	symName := sym.Name()
	if v, ok := pp.visited[symName]; ok && v { //Symbol already checked
		return true
	} else if _, ok := sym.(st.ITypeSymbol); symName != "" && ok { //Mark as checked
		pp.visited[symName] = true
	}
	return false
}
