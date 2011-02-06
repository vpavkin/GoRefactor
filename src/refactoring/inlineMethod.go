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
	"os"
	"go/printer"
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
			for _, el := range *vis.Package.Imports[vis.destFile] {
				if p == el.(*st.PackageSymbol).Package {
					if el.(*st.PackageSymbol).Name() != ps.Name() {
						vis.Chan <- ast.NewIdent(el.(*st.PackageSymbol).Name())
					} else {
						vis.Chan <- nil
					}
					return nil
				}
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

type fixPositionsVisitor struct{
	sourceOrigin token.Pos
	destOrigin token.Pos
	list []ast.Stmt
}

func (vis *fixPositionsVisitor) newPos(pos token.Pos) token.Pos{
	return vis.destOrigin + (pos - vis.sourceOrigin)
}
func (vis *fixPositionsVisitor) Visit(node ast.Node) ast.Visitor{
	switch t := node.(type) {
		case *ast.Ident:
			t.NamePos = vis.newPos(t.NamePos)
	}
	return vis
}
func fixPositions(sourceOrigin token.Pos,destOrigin token.Pos,list []ast.Stmt) {
	vis := &fixPositionsVisitor{sourceOrigin,destOrigin,list}
	for _,stmt := range list{
		ast.Walk(vis,stmt)
	}
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

	if len(resList) > 0{
		sourcePos := resList[0].Pos()
		destPos := callExpr.Pos()
		fixPositions(sourcePos,destPos,resList)
	}
	fmt.Printf("%T %v %v\n", nodeFrom, CallAsExpression, len(destScope))
	for sym, expr := range newNames {
		fmt.Printf("%s -> ", sym.Name())
		printer.Fprint(os.Stdout, token.NewFileSet(), expr)
		print("; ")
	}
	println()
	for _, stmt := range decl.Body.List {
		printer.Fprint(os.Stdout, token.NewFileSet(), stmt)
		println()
	}
	
	println("\n=======================")
	for _, stmt := range resList {
		printer.Fprint(os.Stdout, token.NewFileSet(), stmt)
		println()
	}
	return false, errors.ArgumentError("blah", "blah blah blah")
}
