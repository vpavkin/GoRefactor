package packageParser

import (
	"go/ast"
	//"go/parser"
	"strconv"
	"container/vector"
	//"path"
	"st"
	"go/token"
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
	LocalsParser  *localsVisitor     // An ast.Visitor to register inner scopes of methods

	ExprParser *exprParser

	Mode int // Current builder mode (TYPE_MODE, GLOBALS_MODE, FUNCTIONS_MODE or LOCALS_MODE)
	//	RegisterPositions bool // If true, information about positions is gathered

	CurrentFileName string //Used for resolving packages local names

	visited map[string]bool
}

func ParsePackage(rootPack *st.Package) (*st.SymbolTable, *vector.Vector) {

	pp := newPackageParser(rootPack)

	<-pp.Package.Communication
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
	fmt.Printf("gggggggg parsing globals at pack %s\n", rootPack.AstPackage.Name)
	for fName, atree := range rootPack.AstPackage.Files {
		pp.CurrentFileName = fName
		ast.Walk(pp.GlobalsParser, atree.Decls)
	}

	pp.Package.Communication <- 0

	// fix globals
	<-pp.Package.Communication

	pp.Mode = GLOBALS_FIXING_MODE

	for fName, atree := range rootPack.AstPackage.Files {
		pp.CurrentFileName = fName
		ast.Walk(pp.GlobalsFixer, atree.Decls)
	}
	pp.Package.Communication <- 0

	//Locals
	<-pp.Package.Communication

	if !pp.Package.IsGoPackage {
		fmt.Printf("@@@@@@@@ parsing locals at pack %s\n", rootPack.AstPackage.Name)
		for fName, atree := range rootPack.AstPackage.Files {
			pp.CurrentFileName = fName
			ast.Walk(pp.LocalsParser, atree.Decls)
		}
	}

	pp.Package.Communication <- 0
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

	pp := &packageParser{p, p.Symbols, p.Symbols, nil, nil, nil, nil, nil, new(exprParser), INITIAL_STATE, "", nil}

	pp.TypesParser = &typesVisitor{pp}
	pp.MethodsParser = &methodsVisitor{pp}
	pp.GlobalsParser = &globalsVisitor{pp, nil}
	pp.GlobalsFixer = &globalsFixVisitor{pp}
	pp.LocalsParser = &localsVisitor{pp}

	return pp
}

func getIntValue(t *ast.BasicLit) int {
	l, _ := strconv.Atoi(string(t.Value))
	return l
}
//Builds a type symbol according to given ast.Expression
func (pp *packageParser) parseTypeSymbol(typ ast.Expr) (result st.ITypeSymbol) {

	if typ == nil {
		return nil
	}

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

	var toLookUp *st.SymbolTable
	if base.PackageFrom() == pp.Package || base.PackageFrom() == nil {
		toLookUp = pp.Package.Symbols
	} else {
		toLookUp = base.PackageFrom().Symbols
	}
	if base.Name() == "typesVisitor" {
		fmt.Printf("goap %v and pp is in %v.  toLookUp is from %s\n", base.PackageFrom().AstPackage.Name, pp.Package.AstPackage.Name, toLookUp.Package.AstPackage.Name)
	}

	if result, found = toLookUp.LookUpPointerType(nameToFind, pd+1); found {
		fmt.Printf("Searched Pointer Type %s (%p) ,%d, from package %s\n", nameToFind, result, pd+1, func(p *st.Package) string {
			if p == nil {
				return "nil"
			}
			return p.AstPackage.Name
		}(result.PackageFrom()))
		return
	}

	result = &st.PointerTypeSymbol{&st.TypeSymbol{Obj: base.Object(), Meths: nil, Posits: make(map[string]token.Position), PackFrom: toLookUp.Package}, base, nil}
	fmt.Printf("Adding Pointer Type %s to %s at file of %s\n", result.Name(), func(p *st.Package) string {
		if p == nil {
			return "nil"
		}
		return p.AstPackage.Name
	}(base.PackageFrom()), pp.Package.AstPackage.Name)

	//Solve where to place new pointer
	// 	if _, found := pp.RootSymbolTable.LookUp(nameToFind, ""); found {
	// 		pp.RootSymbolTable.AddSymbol(result)
	// 	} else {
	// 		pp.CurrentSymbolTable.AddSymbol(result)
	// 	}
	toLookUp.AddSymbol(result)
	return
}

// parse functions for specific nodes
func (pp *packageParser) tParseStarExpr(t *ast.StarExpr) (result *st.PointerTypeSymbol) {
	base := pp.parseTypeSymbol(t.X)
	result = pp.getOrAddPointer(base)
	return result
}

func (pp *packageParser) tParseArrayType(t *ast.ArrayType) (result *st.ArrayTypeSymbol) {
	result = &st.ArrayTypeSymbol{TypeSymbol: &st.TypeSymbol{Posits: make(map[string]token.Position), PackFrom: pp.Package}, ElemType: pp.parseTypeSymbol(t.Elt)}
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
	result = &st.ChanTypeSymbol{&st.TypeSymbol{Posits: make(map[string]token.Position), PackFrom: pp.Package}, pp.parseTypeSymbol(t.Value)}
	return
}
func (pp *packageParser) tParseInterfaceType(t *ast.InterfaceType) (result *st.InterfaceTypeSymbol) {
	result = &st.InterfaceTypeSymbol{&st.TypeSymbol{Posits: make(map[string]token.Position), PackFrom: pp.Package}}
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

				toAdd := &st.FunctionSymbol{Obj: name.Obj, FunctionType: ft, Locals: nil, Posits: make(map[string]token.Position), PackFrom: pp.Package}

				toAdd.AddPosition(method.Pos())

				result.AddMethod(toAdd)
			}
		}
	}
	return
}
func (pp *packageParser) tParseMapType(t *ast.MapType) (result *st.MapTypeSymbol) {
	result = &st.MapTypeSymbol{&st.TypeSymbol{Posits: make(map[string]token.Position), PackFrom: pp.Package}, pp.parseTypeSymbol(t.Key), pp.parseTypeSymbol(t.Value)}
	return
}
func (pp *packageParser) tParseIdent(t *ast.Ident) (result st.ITypeSymbol) {

	if sym, found := pp.CurrentSymbolTable.LookUp(t.Name, pp.CurrentFileName); found {
		result = sym.(st.ITypeSymbol)
		t.Obj = sym.Object()

	} else {

		t.Obj = &ast.Object{Kind: ast.Var, Name: t.Name}
		result = &st.UnresolvedTypeSymbol{&st.TypeSymbol{Obj: t.Obj, Posits: make(map[string]token.Position), PackFrom: pp.Package}, t}

		if pp.Mode != TYPES_MODE {
			fmt.Printf("**************** %s\n", t.Name)
		}
	}

	//fmt.Printf("AOAOAOAO %s %v\n", t.Name, t.Pos())
	result.AddPosition(t.Pos())

	return
}
func (pp *packageParser) tParseFuncType(t *ast.FuncType) (result *st.FunctionTypeSymbol) {
	res := &st.FunctionTypeSymbol{&st.TypeSymbol{Posits: make(map[string]token.Position), PackFrom: pp.Package}, nil, nil, nil}
	if t.Params != nil {
		res.Parameters = pp.registerNewSymbolTable()
		e_count := 0
		for _, field := range t.Params.List {
			ftype := pp.parseTypeSymbol(field.Type)
			if len(field.Names) == 0 { // unnamed parameter. separate names are given to distinct parameters in symbolTable
				res.Parameters.AddSymbol(&st.VariableSymbol{Obj: &ast.Object{Kind: ast.Var, Name: "$unnamed" + strconv.Itoa(e_count)}, VariableType: ftype, Posits: make(map[string]token.Position), PackFrom: pp.Package})
				e_count += 1
			}
			for _, name := range field.Names {
				name.Obj = &ast.Object{Kind: ast.Var, Name: name.Name}

				toAdd := &st.VariableSymbol{Obj: name.Obj, VariableType: ftype, Posits: make(map[string]token.Position), PackFrom: pp.Package}

				toAdd.AddPosition(name.Pos())
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
				res.Results.AddSymbol(&st.VariableSymbol{Obj: &ast.Object{Kind: ast.Var, Name: "$unnamed" + strconv.Itoa(e_count)}, VariableType: ftype, Posits: make(map[string]token.Position), PackFrom: pp.Package})
				e_count += 1
			}
			for _, name := range field.Names {
				name.Obj = &ast.Object{Kind: ast.Var, Name: name.Name}

				toAdd := &st.VariableSymbol{Obj: name.Obj, VariableType: ftype, Posits: make(map[string]token.Position), PackFrom: pp.Package}

				toAdd.AddPosition(name.Pos())

				res.Results.AddSymbol(toAdd)
			}
		}
	}
	result = res
	return
}
func (pp *packageParser) tParseStructType(t *ast.StructType) (result *st.StructTypeSymbol) {

	res := &st.StructTypeSymbol{&st.TypeSymbol{Posits: make(map[string]token.Position), PackFrom: pp.Package}, pp.registerNewSymbolTable()}
	for _, field := range t.Fields.List {
		ftype := pp.parseTypeSymbol(field.Type)
		if len(field.Names) == 0 { //Embedded field
			res.Fields.AddSymbol(ftype)
		}
		for _, name := range field.Names {
			name.Obj = &ast.Object{Kind: ast.Var, Name: name.Name}

			toAdd := &st.VariableSymbol{Obj: name.Obj, VariableType: ftype, Posits: make(map[string]token.Position), PackFrom: pp.Package}

			toAdd.AddPosition(name.Pos())

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
			pack.AddPosition(t.X.Pos())
			name := t.Sel.Name
			var res st.Symbol
			if res, found = pack.Package.Symbols.LookUp(name, ""); !found {
				res = &st.UnresolvedTypeSymbol{&st.TypeSymbol{Obj: &ast.Object{Name: t.Sel.Name}, Posits: make(map[string]token.Position), PackFrom: pack.Package}, t}
				pack.Package.Symbols.AddSymbol(res)
			}

			res.AddPosition(t.Sel.Pos())

			result = res.(st.ITypeSymbol)
		} else {
			panic("Can't find package " + pp.Package.QualifiedPath + " " + pp.CurrentFileName + " " + t.X.(*ast.Ident).Name + "." + t.Sel.Name + "\n")
		}

	} else {
		pref := pp.parseTypeSymbol(t.X)
		if pref.Name() == "vector" {
			fmt.Printf("GGG vector package at %v\n", t.X.Pos())
		}
		switch p := pref.(type) {
		case *st.PackageSymbol:
			if sym, found := p.Package.Symbols.LookUp(t.Sel.Name, ""); found {
				result = sym.(st.ITypeSymbol)
				if sym.Name() == "Vector" {
					fmt.Printf("GGG Vector symbol at %v\n", t.Sel.Pos())
				}
				t.Sel.Obj = sym.Object()

				sym.AddPosition(t.X.Pos())

			} else {
				//result = &st.UnresolvedTypeSymbol{&st.TypeSymbol{Obj: &ast.Object{Name: pref.Name() + "." + t.Sel.Name}, Posits: make(map[string]token.Position)}, t}
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
	result = &st.ArrayTypeSymbol{&st.TypeSymbol{Posits: make(map[string]token.Position)}, elt, st.SLICE}
	return
}
