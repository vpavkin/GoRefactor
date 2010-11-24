package packageParser

import (
	"go/ast"
	//"go/parser"
	"strconv"
	"container/vector"
	//"path"
	"st"
	//"os"
	//"utils"
)
import "fmt"
//import "path"

const (
	INITIAL_STATE = iota
	IMPORTS_MODE
	TYPES_MODE
	TYPES_FIXING_MODE
	GLOBALS_MODE
	GLOBALS_FIXING_MODE
	METHODS_MODE
	LOCALS_MODE
)


//Represents a builder object, which provides a st in a few passes
type packageParser struct {
	Package *st.Package

	RootSymbolTable    *st.SymbolTable // fast link to Package.Symbols	
	CurrentSymbolTable *st.SymbolTable // A symbol table which is currently being filled

	TypesParser   *typesVisitor      // An ast.Visitor to register all global types
	GlobalsParser *globalsVisitor    // An ast.Visitor to register all global variables
	GlobalsFixer  *globalsFixVisitor // An ast.Visitor to register all global variables
	MethodsParser *methodsVisitor    // An ast.Visitor to register all methods
	//	LocalsParser       *localsVisitor 	 // An ast.Visitor to register inner scopes of methods

	ExprParser *exprParser

	Mode              int  // Current builder mode (TYPE_MODE, GLOBALS_MODE, FUNCTIONS_MODE or LOCALS_MODE)
	RegisterPositions bool // If true, information about positions is gathered

	CurrentFileName string //Used for resolving packages local names

	visited map[string]bool
}

func ParsePackage(rootPack *st.Package) (*st.SymbolTable, *vector.Vector) {

	pp := newPackageParser(rootPack)

	pp.Package.Communication <- 0
	pp.Mode = TYPES_MODE
	for fName, atree := range rootPack.AstPackage.Files {

		pp.CurrentFileName = fName
		ast.Walk(pp.TypesParser, atree.Decls)
	}

	pp.Package.Communication <- 0

	<-pp.Package.Communication

	fmt.Printf("+++++++++++ package %s started fixing; fn=%s \n", pp.Package.AstPackage.Name, pp.CurrentFileName)

	pp.Mode = TYPES_FIXING_MODE

	pp.fixRootTypes()

	pp.Package.Communication <- 0

	//methods
	<-pp.Package.Communication

	pp.Mode = METHODS_MODE
	for fName, atree := range rootPack.AstPackage.Files {
		pp.CurrentFileName = fName
		ast.Walk(pp.MethodsParser, atree.Decls)
	}

	pp.fixMethodsAndFields()

	pp.Package.Communication <- 0

	// globals
	<-pp.Package.Communication

	pp.Mode = GLOBALS_MODE
	pp.RegisterPositions = false
	for fName, atree := range rootPack.AstPackage.Files {
		pp.CurrentFileName = fName
		ast.Walk(pp.GlobalsParser, atree.Decls)
	}
	pp.RegisterPositions = true
	pp.Package.Communication <- 0

	// fix globals
	<-pp.Package.Communication

	pp.Mode = GLOBALS_FIXING_MODE

	for fName, atree := range rootPack.AstPackage.Files {
		pp.CurrentFileName = fName
		ast.Walk(pp.GlobalsFixer, atree.Decls)
	}
	pp.Package.Communication <- 0

	/*
		//Locals
		for filename, atree := range pack.Files {
			if IsGoFile(filename) {
				ast.Walk(pp.LocalsBuilder, atree.Decls)
		}
		}
	*/
	return pp.RootSymbolTable, pp.Package.SymbolTablePool

}


func (pp *packageParser) registerNewSymbolTable() (table *st.SymbolTable) {
	table = st.NewSymbolTable(pp.Package)
	pp.Package.SymbolTablePool.Push(table)
	return
}

func (pp *packageParser) findSymbolByPosition(fileName string, line int, column int) (st.Symbol, *st.SymbolTable, bool) {
	for _, s := range *pp.Package.SymbolTablePool {
		table := s.(*st.SymbolTable)
		if sym, found := table.FindSymbolByPosition(fileName, line, column); found {
			return sym, table, true
		}
	}
	return nil, nil, false
}


/*^^SymbolTableBuilder Methods^^*/
func newPackageParser(p *st.Package) *packageParser {

	pp := &packageParser{p, p.Symbols, p.Symbols, nil, nil, nil, nil, new(exprParser), INITIAL_STATE, true, "", nil}

	pp.TypesParser = &typesVisitor{pp}
	pp.MethodsParser = &methodsVisitor{pp}
	pp.GlobalsParser = &globalsVisitor{pp, nil}
	pp.GlobalsFixer = &globalsFixVisitor{pp}

	//imv := &localsVisitor{pp}

	return pp
}

func getIntValue(t *ast.BasicLit) int {
	l, _ := strconv.Atoi(string(t.Value))
	return l
}
//Builds a type symbol according to given ast.Expression
func (pp *packageParser) parseTypeSymbol(typ ast.Expr) (result st.ITypeSymbol) {
	switch t := typ.(type) {
	case *ast.StarExpr:
		result = pp.tParseStarExpr(t)
	case *ast.ArrayType:
		result = pp.tParseArrayType(t)
	case *ast.ChanType:
		result = pp.tParseChanType(t)
	case *ast.InterfaceType:
		result = pp.tParseInterfaceType(t)
	case *ast.MapType:
		result = pp.tParseMapType(t)
	case *ast.Ident:
		result = pp.tParseIdent(t)
	case *ast.FuncType:
		result = pp.tParseFuncType(t)
	case *ast.StructType:
		result = pp.tParseStructType(t)
	case *ast.SelectorExpr:
		result = pp.tParseSelector(t)
	case *ast.Ellipsis: //parameters
		result = pp.tParseEllipsis(t)
	}

	return
}

func (pp *packageParser) getOrAddPointer(base st.ITypeSymbol) (result *st.PointerTypeSymbol) {

	nameToFind := base.Name()

	//anonymous type
	if base.Name() == "" {
		result = &st.PointerTypeSymbol{TypeSymbol: &st.TypeSymbol{}, BaseType: base}
		return
	}

	pd, found := 0, false
	if p, ok := base.(*st.PointerTypeSymbol); ok {
		pd = p.Depth()
		nameToFind = p.BaseName()
	}

	if result, found = pp.CurrentSymbolTable.LookUpPointerType(nameToFind, pd+1); found {
		fmt.Printf("Searched Pointer Type %s,%d\n", nameToFind, pd+1)
		return
	}

	result = &st.PointerTypeSymbol{&st.TypeSymbol{Obj: base.Object(), Meths: nil, Posits: new(vector.Vector), PackFrom: pp.Package}, base, nil}
	fmt.Printf(" %s Adding Pointer Type %s\n", pp.Package.AstPackage.Name, result.Name())

	//Solve where to place new pointer
	if _, found := pp.RootSymbolTable.LookUp(nameToFind, ""); found {
		pp.RootSymbolTable.AddSymbol(result)
	} else {
		pp.CurrentSymbolTable.AddSymbol(result)
	}
	return
}

// parse functions for specific nodes
func (pp *packageParser) tParseStarExpr(t *ast.StarExpr) (result *st.PointerTypeSymbol) {
	base := pp.parseTypeSymbol(t.X)
	result = pp.getOrAddPointer(base)
	return result;
}

func (pp *packageParser) tParseArrayType(t *ast.ArrayType) (result *st.ArrayTypeSymbol) {
	result = &st.ArrayTypeSymbol{TypeSymbol: &st.TypeSymbol{Posits: new(vector.Vector), PackFrom: pp.Package}, ElemType: pp.parseTypeSymbol(t.Elt)}
	if t.Len == nil {
		result.Len = st.SLICE
	} else {
		switch tt := t.Len.(type) {
		case *ast.Ellipsis:
			result.Len = st.ELLIPSIS
		case *ast.BasicLit:
			result.Len = getIntValue(tt)
		}
	}
	return
}
func (pp *packageParser) tParseChanType(t *ast.ChanType) (result *st.ChanTypeSymbol) {
	result = &st.ChanTypeSymbol{&st.TypeSymbol{Posits: new(vector.Vector), PackFrom: pp.Package}, pp.parseTypeSymbol(t.Value)}
	return
}
func (pp *packageParser) tParseInterfaceType(t *ast.InterfaceType) (result *st.InterfaceTypeSymbol) {
	result = &st.InterfaceTypeSymbol{&st.TypeSymbol{Posits: new(vector.Vector), PackFrom: pp.Package}}
	if len(t.Methods.List) > 0 {
		result.Meths = pp.registerNewSymbolTable()
		for _, method := range t.Methods.List {
			if len(method.Names) == 0 { //Embeded interface
				ft := pp.parseTypeSymbol(method.Type)
				result.AddMethod(ft)
			}
			for _, name := range method.Names {
				ft := pp.parseTypeSymbol(method.Type).(*st.FunctionTypeSymbol)

				name.Obj = &ast.Object{Kind: ast.Var, Name: name.Name}

				toAdd := &st.FunctionSymbol{Obj: name.Obj, FunctionType: ft, Locals: nil, Posits: new(vector.Vector), PackFrom: pp.Package}
				if pp.RegisterPositions {
					toAdd.AddPosition(st.NewOccurence(method.Pos()))
				}
				result.AddMethod(toAdd)
			}
		}
	}
	return
}
func (pp *packageParser) tParseMapType(t *ast.MapType) (result *st.MapTypeSymbol) {
	result = &st.MapTypeSymbol{&st.TypeSymbol{Posits: new(vector.Vector), PackFrom: pp.Package}, pp.parseTypeSymbol(t.Key), pp.parseTypeSymbol(t.Value)}
	return
}
func (pp *packageParser) tParseIdent(t *ast.Ident) (result st.ITypeSymbol) {

	if sym, found := pp.CurrentSymbolTable.LookUp(t.Name, pp.CurrentFileName); found {
		result = sym.(st.ITypeSymbol)
		t.Obj = sym.Object()
		
	} else {

		t.Obj = &ast.Object{Kind: ast.Var, Name: t.Name}
		result = &st.UnresolvedTypeSymbol{&st.TypeSymbol{Obj: t.Obj, Posits: new(vector.Vector), PackFrom: pp.Package}, t}
		
		if pp.Mode != TYPES_MODE {
			fmt.Printf("**************** %s\n", t.Name)
		}
	}
	if pp.RegisterPositions {
		fmt.Printf("AOAOAOAO %s %v\n",t.Name,t.Pos())
		result.AddPosition(st.NewOccurence(t.Pos()))
	}
	return
}
func (pp *packageParser) tParseFuncType(t *ast.FuncType) (result *st.FunctionTypeSymbol) {
	res := &st.FunctionTypeSymbol{&st.TypeSymbol{Posits: new(vector.Vector), PackFrom: pp.Package}, nil, nil, nil}
	if t.Params != nil {
		res.Parameters = pp.registerNewSymbolTable()
		e_count := 0
		for _, field := range t.Params.List {
			ftype := pp.parseTypeSymbol(field.Type)
			if len(field.Names) == 0 { // unnamed parameter. separate names are given to distinct parameters in symbolTable
				res.Parameters.AddSymbol(&st.VariableSymbol{Obj: &ast.Object{Kind: ast.Var, Name: "$unnamed" + strconv.Itoa(e_count)}, VariableType: ftype, Posits: new(vector.Vector), PackFrom: pp.Package})
				e_count += 1
			}
			for _, name := range field.Names {
				name.Obj = &ast.Object{Kind: ast.Var, Name: name.Name}

				toAdd := &st.VariableSymbol{Obj: name.Obj, VariableType: ftype, Posits: new(vector.Vector), PackFrom: pp.Package}
				if pp.RegisterPositions {
					toAdd.AddPosition(st.NewOccurence(name.Pos()))
				}
				res.Parameters.AddSymbol(toAdd)
			}
		}
	}
	if t.Results != nil {
		res.Results = pp.registerNewSymbolTable()
		e_count := 0
		for _, field := range t.Results.List {
			ftype := pp.parseTypeSymbol(field.Type)
			if len(field.Names) == 0 { //unnamed result
				res.Results.AddSymbol(&st.VariableSymbol{Obj: &ast.Object{Kind: ast.Var, Name: "$unnamed" + strconv.Itoa(e_count)}, VariableType: ftype, Posits: new(vector.Vector), PackFrom: pp.Package})
				e_count += 1
			}
			for _, name := range field.Names {
				name.Obj = &ast.Object{Kind: ast.Var, Name: name.Name}

				toAdd := &st.VariableSymbol{Obj: name.Obj, VariableType: ftype, Posits: new(vector.Vector), PackFrom: pp.Package}
				if pp.RegisterPositions {
					toAdd.AddPosition(st.NewOccurence(name.Pos()))
				}
				res.Results.AddSymbol(toAdd)
			}
		}
	}
	result = res
	return
}
func (pp *packageParser) tParseStructType(t *ast.StructType) (result *st.StructTypeSymbol) {

	res := &st.StructTypeSymbol{&st.TypeSymbol{Posits: new(vector.Vector), PackFrom: pp.Package}, pp.registerNewSymbolTable()}
	for _, field := range t.Fields.List {
		ftype := pp.parseTypeSymbol(field.Type)
		if len(field.Names) == 0 { //Embedded field
			res.Fields.AddSymbol(ftype)
		}
		for _, name := range field.Names {
			name.Obj = &ast.Object{Kind: ast.Var, Name: name.Name}

			toAdd := &st.VariableSymbol{Obj: name.Obj, VariableType: ftype, Posits: new(vector.Vector), PackFrom: pp.Package}
			if pp.RegisterPositions {
				toAdd.AddPosition(st.NewOccurence(name.Pos()))
			}
			res.Fields.AddSymbol(toAdd)
			if name.Name == "Rparen" {
				fmt.Printf(">>>>>>>>>>>>>>>Rparen %s\n", ftype.Name())
			}

		}
	}
	result = res
	return
}
// using type from imported package
func (pp *packageParser) tParseSelector(t *ast.SelectorExpr) (result st.ITypeSymbol) {
	if pp.Mode == TYPES_MODE {
		if sym, found := pp.CurrentSymbolTable.LookUp(t.X.(*ast.Ident).Name, pp.CurrentFileName); found {
			pack := sym.(*st.PackageSymbol)
			pack.AddPosition(st.NewOccurence(t.X.Pos()))
			name := t.X.(*ast.Ident).Name + "." + t.Sel.Name;
			var res st.Symbol;
			if res,found = pp.RootSymbolTable.LookUp(name,pp.CurrentFileName);!found{
				res = &st.UnresolvedTypeSymbol{&st.TypeSymbol{Obj: &ast.Object{Name: t.X.(*ast.Ident).Name + "." + t.Sel.Name}, Posits: new(vector.Vector), PackFrom: pack.Package}, t}
				pp.RootSymbolTable.AddSymbol(res);
			}
			if pp.RegisterPositions {
				res.AddPosition(st.NewOccurence(t.Sel.Pos()))
			}
			result = res.(st.ITypeSymbol);
		} else {
			panic("Can't find package " + pp.Package.QualifiedPath + " " + pp.CurrentFileName + " " + t.X.(*ast.Ident).Name + "." + t.Sel.Name + "\n")
		}

	} else {
		pref := pp.parseTypeSymbol(t.X)
		if pref.Name() == "vector" {
			fmt.Printf("GGG vector package at %v\n", t.X.Pos());
		}
		switch p := pref.(type) {
		case *st.PackageSymbol:
			if sym, found := p.Package.Symbols.LookUp(t.Sel.Name, ""); found {
				result = sym.(st.ITypeSymbol)
				if sym.Name() == "Vector" {
					fmt.Printf("GGG Vector symbol at %v\n", t.Sel.Pos());
				}
				t.Sel.Obj = sym.Object()
				if pp.RegisterPositions {
					sym.AddPosition(st.NewOccurence(t.X.Pos()))
				}
			} else {
				//result = &st.UnresolvedTypeSymbol{&st.TypeSymbol{Obj: &ast.Object{Name: pref.Name() + "." + t.Sel.Name}, Posits: new(vector.Vector)}, t}
				panic("Can't find symbol " + t.Sel.Name + " in package " + pp.Package.QualifiedPath + " " + pp.CurrentFileName + "\n")
			}
		case *st.UnresolvedTypeSymbol:
			panic("Can't find package " + pp.Package.QualifiedPath + " " + pp.CurrentFileName + " " + pref.Name() + "." + t.Sel.Name + "\n")
		}
	}
	return
}
// for function types
func (pp *packageParser) tParseEllipsis(t *ast.Ellipsis) (result *st.ArrayTypeSymbol) {
	elt := pp.parseTypeSymbol(t.Elt)
	result = &st.ArrayTypeSymbol{&st.TypeSymbol{Posits: new(vector.Vector)}, elt, st.SLICE}
	return
}
