package packageParser

import (
	"go/ast"
	//"go/parser"
	"strconv"
	"container/vector"
	//"path"
	"st"
	//"go/token"
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
	REFACTORING_MODE
)


//Represents a builder object, which provides a st in a few passes
type packageParser struct {
	Package *st.Package

	IdentMap           st.IdentifierMap
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

	visited map[st.Symbol]bool
}

func ParsePackage(rootPack *st.Package, identMap st.IdentifierMap) (*st.SymbolTable, *vector.Vector) {

	pp := newPackageParser(rootPack, identMap)

	<-pp.Package.Communication
	pp.Mode = TYPES_MODE
	for fName, atree := range rootPack.AstPackage.Files {

		pp.CurrentFileName = fName
		for _, decl := range atree.Decls {
			ast.Walk(pp.TypesParser, decl)
		}
	}

	pp.Package.Communication <- 0

	<-pp.Package.Communication

	//fmt.Printf("+++++++++++ package %s started fixing; fn=%s \n", pp.Package.AstPackage.Name, pp.CurrentFileName)

	pp.Mode = TYPES_FIXING_MODE

	pp.fixRootTypes()

	pp.Package.Communication <- 0

	//methods
	<-pp.Package.Communication

	pp.Mode = METHODS_MODE
	for fName, atree := range rootPack.AstPackage.Files {
		pp.CurrentFileName = fName
		for _, decl := range atree.Decls {
			ast.Walk(pp.MethodsParser, decl)
		}
	}

	pp.fixMethodsAndFields()

	pp.Package.Communication <- 0

	// globals
	<-pp.Package.Communication

	pp.Mode = GLOBALS_MODE
	//fmt.Printf("gggggggg parsing globals at pack %s\n", rootPack.AstPackage.Name)
	for fName, atree := range rootPack.AstPackage.Files {
		pp.CurrentFileName = fName
		for _, decl := range atree.Decls {
			ast.Walk(pp.GlobalsParser, decl)
		}
	}

	pp.Package.Communication <- 0

	// fix globals
	<-pp.Package.Communication

	pp.Mode = GLOBALS_FIXING_MODE

	for fName, atree := range rootPack.AstPackage.Files {
		pp.CurrentFileName = fName
		for _, decl := range atree.Decls {
			ast.Walk(pp.GlobalsFixer, decl)
		}
	}

	// 	if pp.Package.AstPackage.Name == "st" {
	// 		fmt.Printf("$$$$$$$$$ Symbols of st:\n")
	// 		pp.Package.Symbols.ForEach(func(sym st.Symbol) {
	// 			fmt.Printf("%s", sym.Name())
	// 			switch symt := sym.(type) {
	// 			case *st.PointerTypeSymbol:
	// 				if symt.Fields != nil {
	// 					fmt.Printf(":\nfields:\n %s\n", *symt.Fields.String())
	// 				}
	// 				if symt.Methods() != nil {
	// 					fmt.Printf("methods:\n %s\n", *symt.Methods().String())
	// 				}
	// 			case *st.StructTypeSymbol:
	// 				if symt.Fields != nil {
	// 					fmt.Printf(":\nfields:\n %s\n", *symt.Fields.String())
	// 				}
	// 				if symt.Methods() != nil {
	// 					fmt.Printf("methods:\n %s\n", *symt.Methods().String())
	// 				}
	// 			default:
	// 				if tt, ok := sym.(st.ITypeSymbol); ok {
	// 					if tt.Methods() != nil {
	// 						fmt.Printf(":\nmethods:\n %s\n", *tt.Methods().String())
	// 					}
	// 				} else if vt, ok := sym.(*st.VariableSymbol); ok {
	// 					fmt.Printf(" %s", vt.VariableType.Name())
	// 				} else if vt, ok := sym.(*st.FunctionSymbol); ok {
	// 					fmt.Printf(" %s", vt.FunctionType.Name())
	// 				}
	// 			}
	// 		})
	// 	}
	pp.Package.Communication <- 0
	//Locals
	<-pp.Package.Communication

	if !pp.Package.IsGoPackage {
		// 		fmt.Printf("@@@@@@@@ parsing locals at pack %s\n", rootPack.AstPackage.Name)
		for fName, atree := range rootPack.AstPackage.Files {
			pp.CurrentFileName = fName
			for _, decl := range atree.Decls {
				ast.Walk(pp.LocalsParser, decl)
			}
		}
	}

	pp.Package.Communication <- 0
	return pp.RootSymbolTable, pp.Package.SymbolTablePool

}

func ParseExpr(expr ast.Expr, pack *st.Package, filename string, identMap st.IdentifierMap) st.ITypeSymbol {
	pp := newPackageParser(pack, identMap)
	pp.CurrentFileName = filename
	pp.Mode = REFACTORING_MODE
	v := pp.parseExpr(expr)
	if len(*v) == 0 {
		return st.MakeInterfaceType(st.NO_NAME, pack.Symbols)
	}
	if t, ok := v.At(0).(st.ITypeSymbol); ok {
		return t
	}
	return st.MakeInterfaceType(st.NO_NAME, pack.Symbols)
}

/*^^SymbolTableBuilder Methods^^*/
func newPackageParser(p *st.Package, identMap st.IdentifierMap) *packageParser {

	pp := &packageParser{p, identMap, p.Symbols, p.Symbols, nil, nil, nil, nil, nil, new(exprParser), INITIAL_STATE, "", nil}

	pp.TypesParser = &typesVisitor{pp}
	pp.MethodsParser = &methodsVisitor{pp}
	pp.GlobalsParser = &globalsVisitor{pp, nil}
	pp.GlobalsFixer = &globalsFixVisitor{pp}
	pp.LocalsParser = &localsVisitor{pp}

	return pp
}

func (pp *packageParser) registerIdent(sym st.Symbol, ident *ast.Ident) {
	sym.AddIdent(ident)
	// 	fmt.Printf("%p goes to map as %s %T %p %s\n", ident, sym.Name(), sym, sym, pp.CurrentFileName)
	pp.IdentMap.AddIdent(ident, sym)
	sym.AddPosition(pp.Package.FileSet.Position(ident.Pos()))
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
	if pp.Mode == TYPES_MODE {
		if result.Name() != st.NO_NAME && result.PackageFrom() == pp.Package {
			pp.CurrentSymbolTable.AddSymbol(result)
		}
	}
	return
}

func (pp *packageParser) getOrAddPointer(base st.ITypeSymbol) (result *st.PointerTypeSymbol) {

	nameToFind := base.Name()

	//anonymous type
	if base.Name() == st.NO_NAME {
		result = st.MakePointerType(base.Scope(), base)
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
	// 	if base.Name() == "typesVisitor" {
	// 		fmt.Printf("goap %v and pp is in %v.  toLookUp is from %s\n", base.PackageFrom().AstPackage.Name, pp.Package.AstPackage.Name, toLookUp.Package.AstPackage.Name)
	// 	}

	if result, found = toLookUp.LookUpPointerType(nameToFind, pd+1); found {
		// 		fmt.Printf("Searched Pointer Type %s (%p) ,%d, from package %s\n", nameToFind, result, pd+1, func(p *st.Package) string {
		// 			if p == nil {
		// 				return "nil"
		// 			}
		// 			return p.AstPackage.Name
		// 		}(result.PackageFrom()))
		return
	}

	result = st.MakePointerType(toLookUp, base)
	// 	fmt.Printf("Adding Pointer Type %s to %s at file of %s\n", result.Name(), func(p *st.Package) string {
	// 		if p == nil {
	// 			return "nil"
	// 		}
	// 		return p.AstPackage.Name
	// 	}(base.PackageFrom()),
	// 		pp.Package.AstPackage.Name)

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
	result = st.MakeArrayType(st.NO_NAME, pp.CurrentSymbolTable, pp.parseTypeSymbol(t.Elt), 0)
	if t.Len == nil {
		result.Len = st.SLICE
	} else {
		switch tt := t.Len.(type) {
		case *ast.Ellipsis:
			result.Len = st.ELLIPSIS
		case *ast.BasicLit:
			l, _ := strconv.Atoi(string(tt.Value))
			result.Len = l
		}
	}
	return
}
func (pp *packageParser) tParseChanType(t *ast.ChanType) (result *st.ChanTypeSymbol) {
	result = st.MakeChannelType(st.NO_NAME, pp.CurrentSymbolTable, pp.parseTypeSymbol(t.Value), t.Dir)
	return
}
func (pp *packageParser) tParseInterfaceType(t *ast.InterfaceType) (result *st.InterfaceTypeSymbol) {
	result = st.MakeInterfaceType(st.NO_NAME, pp.CurrentSymbolTable)
	if len(t.Methods.List) > 0 {
		for _, method := range t.Methods.List {
			if len(method.Names) == 0 { //Embeded interface
				ft := pp.parseTypeSymbol(method.Type)
				result.AddMethod(ft)
			}
			for _, name := range method.Names {
				ft := pp.parseTypeSymbol(method.Type).(*st.FunctionTypeSymbol)

				toAdd := st.MakeFunction(name.Name, result.Methods(), ft)
				toAdd.IsInterfaceMethod = true

				pp.registerIdent(toAdd, name)

				result.AddMethod(toAdd)
			}
		}
	}
	return
}
func (pp *packageParser) tParseMapType(t *ast.MapType) (result *st.MapTypeSymbol) {
	result = st.MakeMapType(st.NO_NAME, pp.CurrentSymbolTable, pp.parseTypeSymbol(t.Key), pp.parseTypeSymbol(t.Value))
	return
}
func (pp *packageParser) tParseIdent(t *ast.Ident) (result st.ITypeSymbol) {

	if sym, found := pp.CurrentSymbolTable.LookUp(t.Name, pp.CurrentFileName); found {
		result = sym.(st.ITypeSymbol)
	} else {
		result = st.MakeUnresolvedType(t.Name, pp.CurrentSymbolTable, t)
		// 		fmt.Printf("%p it's Uunres Ident %s %p %s\n", t, result.Name(), result, pp.CurrentFileName)
		if pp.Mode != TYPES_MODE {
			fmt.Printf("**************** %s\n", t.Name)
		}
	}
	pp.registerIdent(result, t)
	return
}
func (pp *packageParser) tParseFuncType(t *ast.FuncType) (result *st.FunctionTypeSymbol) {
	res := st.MakeFunctionType(st.NO_NAME, pp.CurrentSymbolTable)
	if t.Params != nil {
		e_count := 0
		for _, field := range t.Params.List {
			ftype := pp.parseTypeSymbol(field.Type)
			if len(field.Names) == 0 { // unnamed parameter. separate names are given to distinct parameters in symbolTable
				res.Parameters.AddSymbol(st.MakeVariable("$unnamed"+strconv.Itoa(e_count), pp.CurrentSymbolTable, ftype))
				e_count += 1
			}
			for _, name := range field.Names {

				toAdd := st.MakeVariable(name.Name, res.Parameters, ftype)
				pp.registerIdent(toAdd, name)
				res.Parameters.AddSymbol(toAdd)
			}
		}
	}
	if t.Results != nil {
		e_count := 0
		for _, field := range t.Results.List {
			ftype := pp.parseTypeSymbol(field.Type)
			if len(field.Names) == 0 { //unnamed result
				res.Results.AddSymbol(st.MakeVariable("$unnamed"+strconv.Itoa(e_count), pp.CurrentSymbolTable, ftype))
				e_count += 1
			}
			for _, name := range field.Names {

				toAdd := st.MakeVariable(name.Name, res.Results, ftype)
				pp.registerIdent(toAdd, name)
				res.Results.AddSymbol(toAdd)
			}
		}
	}
	result = res
	return
}
func (pp *packageParser) tParseStructType(t *ast.StructType) (result *st.StructTypeSymbol) {

	res := st.MakeStructType(st.NO_NAME, pp.CurrentSymbolTable)
	for _, field := range t.Fields.List {
		ftype := pp.parseTypeSymbol(field.Type)
		if len(field.Names) == 0 { //Embedded field
			res.Fields.AddSymbol(ftype)
		}
		for _, name := range field.Names {

			toAdd := st.MakeVariable(name.Name, res.Fields, ftype)
			pp.registerIdent(toAdd, name)
			res.Fields.AddSymbol(toAdd)

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
			pp.registerIdent(pack, t.X.(*ast.Ident))

			name := t.Sel.Name
			var res st.Symbol
			if res, found = pack.Package.Symbols.LookUp(name, ""); !found {
				res = st.MakeUnresolvedType(name, pack.Package.Symbols, t)
				// 				fmt.Printf("%p it's Uunres %s %p %s\n", t.Sel, res.Name(), res, pp.CurrentFileName)
				pack.Package.Symbols.AddSymbol(res)
			}

			pp.registerIdent(res, t.Sel)

			result = res.(st.ITypeSymbol)
		} else {
			panic("Can't find package " + pp.Package.QualifiedPath + " " + pp.CurrentFileName + " " + t.X.(*ast.Ident).Name + "." + t.Sel.Name + "\n")
		}

	} else {
		pref := pp.parseTypeSymbol(t.X)

		switch p := pref.(type) {
		case *st.PackageSymbol:
			if sym, found := p.Package.Symbols.LookUp(t.Sel.Name, ""); found {

				result = sym.(st.ITypeSymbol)
				pp.registerIdent(sym, t.Sel)
			} else {
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
	result = st.MakeArrayType(st.NO_NAME, pp.CurrentSymbolTable, elt, st.SLICE)
	return
}
