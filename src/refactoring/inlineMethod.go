package refactoring

import (
	"program"
	"st"
	"utils"
	"errors"
	"go/ast"
	"go/token"
	"strconv"

	"fmt"
	// 	"os"
	// 	"go/printer"
)

//given start and end positions visitor finds call expression 
type getCallVisitor struct {
	Package *st.Package

	callStart token.Position
	callEnd   token.Position

	CallNode ast.Node

	nodeFrom ast.Node
}

func (vis *getCallVisitor) found() bool {
	return vis.CallNode != nil
}

func (vis *getCallVisitor) find(n ast.Node) bool {
	return utils.ComparePosWithinFile(vis.Package.FileSet.Position(n.Pos()), vis.callStart) == 0 &&
		utils.ComparePosWithinFile(vis.Package.FileSet.Position(n.End()), vis.callEnd) == 0
}

func (vis *getCallVisitor) checkStmtList(list []ast.Stmt) bool {

	for _, stmt := range list {
		if vis.find(stmt) {
			vis.CallNode = stmt
			return true
		}
	}
	return false
}
func (vis *getCallVisitor) Visit(node ast.Node) ast.Visitor {
	if vis.found() {
		return nil
	}
	switch t := node.(type) {
	case *ast.BlockStmt:
		if vis.checkStmtList(t.List) {
			vis.nodeFrom = t
			return nil
		}
	case *ast.SwitchStmt:
		for _, cc := range t.Body.List {
			caseClause := cc.(*ast.CaseClause)
			if vis.checkStmtList(caseClause.Body) {
				vis.nodeFrom = caseClause
				return nil
			}
		}
	case *ast.TypeSwitchStmt:
		for _, cc := range t.Body.List {
			caseClause := cc.(*ast.TypeCaseClause)
			if vis.checkStmtList(caseClause.Body) {
				vis.nodeFrom = caseClause
				return nil
			}
		}
	case *ast.SelectStmt:
		for _, cc := range t.Body.List {
			caseClause := cc.(*ast.CommClause)
			if vis.checkStmtList(caseClause.Body) {
				vis.nodeFrom = caseClause
				return nil
			}
		}
	case ast.Stmt, ast.Expr:
		if vis.find(t) {
			vis.CallNode = t
			return nil
		}
	}
	return vis
}

func getCall(pack *st.Package, file *ast.File, filename string, lineStart int, colStart int, lineEnd int, colEnd int) (ast.Node, ast.Node) {
	vis := &getCallVisitor{pack, token.Position{filename, 0, lineStart, colStart}, token.Position{filename, 0, lineEnd, colEnd}, nil, nil}
	ast.Walk(vis, file)
	return vis.CallNode, vis.nodeFrom
}

// support functions
func getCallExpr(callNode ast.Node) (*ast.CallExpr, *errors.GoRefactorError) {
	switch node := callNode.(type) {
	case *ast.ExprStmt:
		if res, ok := node.X.(*ast.CallExpr); ok {
			return res, nil
		}
	case *ast.CallExpr:
		return node, nil
	}
	return nil, &errors.GoRefactorError{ErrorType: "inline method error", Message: "selected statement or expression is not a function call"}
}

func getMethodSymbol(programTree *program.Program, callExpr *ast.CallExpr) (*st.FunctionSymbol, *errors.GoRefactorError) {
	var s st.Symbol
	switch funExpr := callExpr.Fun.(type) {
	case *ast.Ident:
		s = programTree.IdentMap.GetSymbol(funExpr)
	case *ast.SelectorExpr:
		s = programTree.IdentMap.GetSymbol(funExpr.Sel)
	default:
		panic("function expression is not a selector or ident")
	}
	if res, ok := s.(*st.FunctionSymbol); ok {
		return res, nil
	}
	panic("Call expr Function is not a st.FunctionSymbol")
}

func getDeclarationInFile(programTree *program.Program, pack *st.Package, sym *st.FunctionSymbol) (decl *ast.FuncDecl, filename string, err *errors.GoRefactorError) {
	for name, f := range pack.AstPackage.Files {
		for _, d := range f.Decls {
			if fdecl, ok := d.(*ast.FuncDecl); ok {
				if programTree.IdentMap.GetSymbol(fdecl.Name) == sym {
					return fdecl, name, nil
				}
			}
		}
	}
	return nil, "", &errors.GoRefactorError{ErrorType: "inline method error (critical)", Message: "couldn't find method declaration"}
}

func getDestScope(programTree *program.Program, pack *st.Package, nodeFrom ast.Node) map[st.Symbol]bool {
	res := make(map[st.Symbol]bool)
	if nodeFrom == nil { // method just uses it's parameters
		return res
	}
	list := getStmtList(nodeFrom)
	params, declared := getParametersAndDeclaredIn(pack, list, programTree)
	params.ForEach(func(sym st.Symbol) {
		res[sym] = true
	})
	declared.ForEach(func(sym st.Symbol) {
		res[sym] = true
	})
	return res
}

func getRecieverExpr(callExpr *ast.CallExpr) ast.Expr {
	switch t := callExpr.Fun.(type) {
	case *ast.Ident:
		panic("reciever expected")
	case *ast.SelectorExpr:
		return t.X
	case *ast.CallExpr:
		panic("can't inline call with reciever, returned by a method (possible side efects)")
	}
	panic("unknown reciever expression type")
}

func getNewNames(callExpr *ast.CallExpr, funSym *st.FunctionSymbol, destScope map[st.Symbol]bool) map[st.Symbol]ast.Expr {
	newNames := make(map[st.Symbol]ast.Expr)
	bt, _ := st.GetBaseType(funSym.FunctionType)
	ftype := bt.(*st.FunctionTypeSymbol)
	//Reciever
	if ftype.Reciever != nil {
		ftype.Reciever.ForEach(func(sym st.Symbol) {
			newNames[sym] = getRecieverExpr(callExpr)
		})
	}
	//Parameters
	i := 0
	ftype.Parameters.ForEach(func(sym st.Symbol) {
		newNames[sym] = callExpr.Args[i]
		i++
	})
	//Locals
	usedNames := make(map[string]bool)
	for dest, _ := range destScope {
		usedNames[dest.Name()] = true
	}
	funSym.Locals.ForEach(func(sym st.Symbol) {
		if _, ok := usedNames[sym.Name()]; ok {
			j := 0
			for {
				nn := sym.Name() + strconv.Itoa(j)
				if _, ok := usedNames[nn]; !ok {
					newNames[sym] = ast.NewIdent(nn)
					usedNames[nn] = true
					break
				}
			}
		}
	})
	return newNames
}

// convertStatementListVisitor prepares statement list of a method to be inlined in destination scope.
// It changes imported packages' aliases and adds import statements if needed.
// Names of method locals, that conflict with destination scope, are renamed.
type statementListConverter struct {
	IdentMap st.IdentifierMap
	Package  *st.Package

	newNames     map[st.Symbol]ast.Expr //for method parameters - passed expressions, for locals - their new names
	importsToAdd map[*st.PackageSymbol]bool

	sourceFile string
	destFile   string

	sourceList []ast.Stmt
	destList   []ast.Stmt

	Chan chan ast.Expr
}

func (slc *statementListConverter) source() {
	sv := &sourceVisitor{slc}
	for _, stmt := range slc.sourceList {
		ast.Walk(sv, stmt)
	}
}
func (slc *statementListConverter) destination(allCovered chan bool) {
	dv := &destinationVisitor{slc, nil}
	for _, stmt := range slc.destList {
		dv.rootNode = stmt
		ast.Walk(dv, stmt)
	}
	allCovered <- true
}

type sourceVisitor struct {
	*statementListConverter
}

func (vis sourceVisitor) Visit(node ast.Node) ast.Visitor {
	switch t := node.(type) {
	case *ast.SelectorExpr:
		ast.Walk(vis, t.X)
		return nil
	case *ast.Ident:
		sym := vis.IdentMap.GetSymbol(t)
		if ps, ok := sym.(*st.PackageSymbol); ok {
			p := ps.Package
			imp := vis.Package.GetImport(vis.destFile, p)
			if imp != nil {
				if imp.Name() != ps.Name() {
					vis.Chan <- ast.NewIdent(imp.Name())
				} else {
					vis.Chan <- nil
				}
				return nil
			}
			vis.importsToAdd[ps] = true
			vis.Chan <- ast.NewIdent(ps.Name())
			return nil
		}
		if nn, ok := vis.newNames[sym]; ok {
			vis.Chan <- nn
		} else {
			vis.Chan <- nil
		}
		return nil
	}
	return vis
}

type destinationVisitor struct {
	*statementListConverter
	rootNode ast.Node
}

func (vis destinationVisitor) Visit(node ast.Node) ast.Visitor {
	switch t := node.(type) {
	case *ast.SelectorExpr:
		ast.Walk(vis, t.X)
		return nil
	case *ast.Ident:
		newExpr := <-vis.Chan
		if newExpr != nil {
			replaceExpr(vis.Package.FileSet.Position(t.Pos()), vis.Package.FileSet.Position(t.End()), newExpr, vis.Package, vis.rootNode)
		}
		return nil
	}
	return vis
}

func getResultStmtList(IdentMap st.IdentifierMap, pack *st.Package, funSym *st.FunctionSymbol, newNames map[st.Symbol]ast.Expr, sourceFile string, destFile string, sourceList []ast.Stmt) []ast.Stmt {
	listCopy := utils.CopyStmtList(sourceList)
	converter := &statementListConverter{IdentMap, pack, newNames, make(map[*st.PackageSymbol]bool), sourceFile, destFile, sourceList, listCopy, make(chan ast.Expr)}
	allCovered := make(chan bool)
	go converter.source()
	go converter.destination(allCovered)
	<-allCovered
	return converter.destList
}

type fixPositionsVisitor struct {
	sourceOrigin token.Pos
	inc          int
}

func (vis *fixPositionsVisitor) newPos(pos token.Pos) token.Pos {
	if pos < vis.sourceOrigin {
		println(pos)
		return pos
	}
	print(pos)
	print(" -> ")
	println(token.Pos(int(pos) + vis.inc))
	return token.Pos(int(pos) + vis.inc)
}
func (vis *fixPositionsVisitor) Visit(node ast.Node) ast.Visitor {
	switch t := node.(type) {
	case *ast.ArrayType:
		t.Lbrack = vis.newPos(t.Lbrack)
	case *ast.AssignStmt:
		t.TokPos = vis.newPos(t.TokPos)
	case *ast.BasicLit:
		t.ValuePos = vis.newPos(t.ValuePos)
	case *ast.BinaryExpr:
		t.OpPos = vis.newPos(t.OpPos)
	case *ast.BlockStmt:
		t.Lbrace = vis.newPos(t.Lbrace)
		t.Rbrace = vis.newPos(t.Rbrace)
		//case *ast.BranchStmt:
	case *ast.CallExpr:
		t.Rparen = vis.newPos(t.Rparen)
		t.Lparen = vis.newPos(t.Lparen)
		t.Ellipsis = vis.newPos(t.Ellipsis)
	case *ast.CaseClause:
		t.Case = vis.newPos(t.Case)
		t.Colon = vis.newPos(t.Colon)
	case *ast.ChanType:
		t.Begin = vis.newPos(t.Begin)
	case *ast.CommClause:
		t.Case = vis.newPos(t.Case)
		t.Colon = vis.newPos(t.Colon)
	case *ast.Comment:
		t.Slash = vis.newPos(t.Slash)
	case *ast.CommentGroup:
	case *ast.CompositeLit:
		t.Lbrace = vis.newPos(t.Lbrace)
		t.Rbrace = vis.newPos(t.Rbrace)
	case *ast.DeclStmt:
	case *ast.DeferStmt:
		t.Defer = vis.newPos(t.Defer)
	case *ast.Ellipsis:
		t.Ellipsis = vis.newPos(t.Ellipsis)
	case *ast.EmptyStmt:
		t.Semicolon = vis.newPos(t.Semicolon)
	case *ast.ExprStmt:
	case *ast.Field:
	case *ast.FieldList:
		t.Opening = vis.newPos(t.Opening)
		t.Closing = vis.newPos(t.Closing)
	case *ast.File:
		t.Package = vis.newPos(t.Package)
	case *ast.ForStmt:
		t.For = vis.newPos(t.For)
	case *ast.FuncDecl:
	case *ast.FuncLit:
	case *ast.FuncType:
		t.Func = vis.newPos(t.Func)
	case *ast.GenDecl:
		t.TokPos = vis.newPos(t.TokPos)
		t.Lparen = vis.newPos(t.Lparen)
		t.Rparen = vis.newPos(t.Rparen)
	case *ast.GoStmt:
		t.Go = vis.newPos(t.Go)
	case *ast.Ident:
		t.NamePos = vis.newPos(t.NamePos)
	case *ast.IfStmt:
		t.If = vis.newPos(t.If)
	case *ast.ImportSpec:
	case *ast.IncDecStmt:
		t.TokPos = vis.newPos(t.TokPos)
	case *ast.IndexExpr:
		t.Lbrack = vis.newPos(t.Lbrack)
		t.Rbrack = vis.newPos(t.Rbrack)
	case *ast.InterfaceType:
		t.Interface = vis.newPos(t.Interface)
	case *ast.KeyValueExpr:
		t.Colon = vis.newPos(t.Colon)
		//case *ast.LabeledStmt:
	case *ast.MapType:
		t.Map = vis.newPos(t.Map)
		//case *ast.Package:
	case *ast.ParenExpr:
		t.Lparen = vis.newPos(t.Lparen)
		t.Rparen = vis.newPos(t.Rparen)
	case *ast.RangeStmt:
		t.For = vis.newPos(t.For)
		t.TokPos = vis.newPos(t.TokPos)
	case *ast.ReturnStmt:
		t.Return = vis.newPos(t.Return)
	case *ast.SelectStmt:
		t.Select = vis.newPos(t.Select)
	case *ast.SelectorExpr:
	case *ast.SendStmt:
		t.Arrow = vis.newPos(t.Arrow)
	case *ast.SliceExpr:
		t.Lbrack = vis.newPos(t.Lbrack)
		t.Rbrack = vis.newPos(t.Rbrack)
	case *ast.StarExpr:
		t.Star = vis.newPos(t.Star)
	case *ast.StructType:
		t.Struct = vis.newPos(t.Struct)
	case *ast.SwitchStmt:
		t.Switch = vis.newPos(t.Switch)
	case *ast.TypeAssertExpr:
	case *ast.TypeCaseClause:
		t.Case = vis.newPos(t.Case)
		t.Colon = vis.newPos(t.Colon)
	case *ast.TypeSpec:
	case *ast.TypeSwitchStmt:
		t.Switch = vis.newPos(t.Switch)
	case *ast.UnaryExpr:
		t.OpPos = vis.newPos(t.OpPos)
	case *ast.ValueSpec:
	}
	return vis
}
func fixPositions(sourceOrigin token.Pos, inc int, node ast.Node) {
	print("sourceOrigin = ")
	println(sourceOrigin)
	vis := &fixPositionsVisitor{sourceOrigin, inc}
	ast.Walk(vis, node)
}

func CheckInlineMethodParameters(filename string, lineStart int, colStart int, lineEnd int, colEnd int) (bool, *errors.GoRefactorError) {
	switch {
	case filename == "" || !utils.IsGoFile(filename):
		return false, errors.ArgumentError("filename", "It's not a valid go file name")
	case lineStart < 1:
		return false, errors.ArgumentError("lineStart", "Must be > 1")
	case lineEnd < 1 || lineEnd < lineStart:
		return false, errors.ArgumentError("lineEnd", "Must be > 1 and >= lineStart")
	case colStart < 1:
		return false, errors.ArgumentError("colStart", "Must be > 1")
	case colEnd < 1:
		return false, errors.ArgumentError("colEnd", "Must be > 1")
	}
	return true, nil
}

func InlineMethod(programTree *program.Program, filename string, lineStart int, colStart int, lineEnd int, colEnd int) (bool, *errors.GoRefactorError) {
	if ok, err := CheckInlineMethodParameters(filename, lineStart, colStart, lineEnd, colEnd); !ok {
		return false, err
	}
	pack, file := programTree.FindPackageAndFileByFilename(filename)
	if pack == nil {
		return false, errors.ArgumentError("filename", "Program packages don't contain file '"+filename+"'")
	}
	callNode, nodeFrom := getCall(pack, file, filename, lineStart, colStart, lineEnd, colEnd)
	if callNode == nil {
		return false, &errors.GoRefactorError{ErrorType: "inline method error", Message: "couldn't find call expression"}
	}
	_, CallAsExpression := callNode.(*ast.CallExpr)
	callExpr, err := getCallExpr(callNode)
	if err != nil {
		return false, err
	}

	funSym, err := getMethodSymbol(programTree, callExpr)
	if err != nil {
		return false, err
	}
	if funSym.PackageFrom() != pack {
		return false, &errors.GoRefactorError{ErrorType: "inline method error", Message: "can't inline method from other package"}
	}
	decl, sourceFile, err := getDeclarationInFile(programTree, pack, funSym)
	if err != nil {
		return false, err
	}

	destScope := getDestScope(programTree, pack, nodeFrom)
	newNames := getNewNames(callExpr, funSym, destScope)

	resList := getResultStmtList(programTree.IdentMap, pack, funSym, newNames, sourceFile, filename, decl.Body.List)

	if len(decl.Body.List) > 0 {
		sourcePos := decl.Body.List[0].Pos()
		destPos := callNode.Pos()
		inc := int(destPos) - int(sourcePos)
		for _, stmt := range resList {
			fixPositions(token.NoPos, inc, stmt)
		}
		inc = int(decl.Body.List[len(decl.Body.List)-1].End()-sourcePos) - int(callNode.End()-destPos)
		fixPositions(destPos, inc, file)
	}

	if CallAsExpression {
		newExpr := resList[0].(*ast.ReturnStmt).Results[0]
		replaceExpr(pack.FileSet.Position(callExpr.Pos()), pack.FileSet.Position(callExpr.End()), newExpr, pack, nodeFrom)
	} else {
		list := getStmtList(nodeFrom)
		i, ok := getIndexOfStmt(callNode.(ast.Stmt), list)
		if !ok {
			panic("couldn't find call statement during inline")
		}
		if len(resList) == 1 {
			list[i] = resList[0]
			return true, nil
		}
		fmt.Printf("%v\n", list)
		newList := append(list, list[len(list)-len(resList)+1:]...)
		fmt.Printf("%v\n", newList)
		for j := i + 1; j < len(list)-len(resList)+1; j++ {
			newList[j+len(resList)-1] = newList[j]
		}
		fmt.Printf("%v\n", newList)
		for j := i; j < len(resList)+i; j++ {
			newList[j] = resList[j-i]
		}
		fmt.Printf("%v\n", newList)
		setStmtList(nodeFrom, newList)
	}

	return true, nil
}
