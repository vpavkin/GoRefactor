package refactoring

import (
	"go/ast"
	"container/vector"
	"st"
	"go/token"
	"utils"
	"errors"
	"program"
	"packageParser"
)

type extractedSetVisitor struct {
	Package *st.Package

	resultBlock  *vector.Vector
	firstNodePos token.Position //in
	lastNodePos  token.Position //in
	firstNode    ast.Node
	lastNode     ast.Node

	nodeFrom ast.Node
}

func (vis *extractedSetVisitor) foundFirst() bool {
	return vis.firstNode != nil
}

func (vis *extractedSetVisitor) isValid() bool {
	return (vis.firstNode != nil) && (vis.lastNode != nil)
}

func (vis *extractedSetVisitor) checkStmtList(list []ast.Stmt) {

	for i, stmt := range list {
		if !vis.foundFirst() {
			if utils.ComparePosWithinFile(vis.Package.FileSet.Position(stmt.Pos()), vis.firstNodePos) == 0 {
				vis.firstNode = stmt
				vis.resultBlock.Push(stmt)

				if utils.ComparePosWithinFile(vis.Package.FileSet.Position(stmt.End()), vis.lastNodePos) == 0 {
					//one stmt method
					vis.lastNode = stmt
				}
			}
		} else {
			if !vis.isValid() {
				vis.resultBlock.Push(stmt)
				if utils.ComparePosWithinFile(vis.Package.FileSet.Position(stmt.End()), vis.lastNodePos) == 0 {
					vis.lastNode = list[i]
				}
			}
		}
	}
}
func (vis *extractedSetVisitor) Visit(node ast.Node) ast.Visitor {

	if vis.foundFirst() {
		return nil
	}
	switch t := node.(type) {
	case *ast.BlockStmt:
		vis.checkStmtList(t.List)
		if vis.isValid() {
			vis.nodeFrom = t
		}
	case *ast.SwitchStmt:
		for _, cc := range t.Body.List {
			caseClause := cc.(*ast.CaseClause)
			vis.checkStmtList(caseClause.Body)
			if vis.isValid() {
				vis.nodeFrom = caseClause
				break
			}
		}
	case *ast.TypeSwitchStmt:
		for _, cc := range t.Body.List {
			caseClause := cc.(*ast.TypeCaseClause)
			vis.checkStmtList(caseClause.Body)
			if vis.isValid() {
				vis.nodeFrom = caseClause
				break
			}
		}
	case *ast.SelectStmt:
		for _, cc := range t.Body.List {
			caseClause := cc.(*ast.CommClause)
			vis.checkStmtList(caseClause.Body)
			if vis.isValid() {
				vis.nodeFrom = caseClause
				break
			}
		}
	case ast.Expr:
		if utils.ComparePosWithinFile(vis.Package.FileSet.Position(t.Pos()), vis.firstNodePos) == 0 {
			if utils.ComparePosWithinFile(vis.Package.FileSet.Position(t.End()), vis.lastNodePos) == 0 {
				vis.firstNode = t
				vis.lastNode = t
				vis.resultBlock.Push(t)
			}
		}
	}
	return vis
}

type checkReturnVisitor struct {
	has bool
}

func (vis *checkReturnVisitor) Visit(node ast.Node) ast.Visitor {
	switch node.(type) {
	case *ast.ReturnStmt:
		vis.has = true
	}
	return vis
}

func checkForReturns(block *vector.Vector) bool {

	vis := &checkReturnVisitor{}
	for _, stmt := range *block {
		ast.Walk(vis, stmt.(ast.Node))
	}
	return vis.has
}

type checkScopingVisitor struct {
	declaredInExtracted *st.SymbolTable
	errs                *st.SymbolTable
	globalIdentMap      st.IdentifierMap
}

func checkScoping(block ast.Node, stmtList []ast.Stmt, declaredInExtracted *st.SymbolTable, globalIdentMap st.IdentifierMap) (bool, *st.SymbolTable) {
	source := getStmtList(block)
	i, found := getIndexOfStmt(stmtList[len(stmtList)-1], source)
	if !found {
		panic("didn't find extracted code's end")
	}
	if i == len(source)-1 {
		return true, nil
	}
	vis := &checkScopingVisitor{declaredInExtracted, st.NewSymbolTable(declaredInExtracted.Package), globalIdentMap}
	for j := i + 1; j < len(source); j++ {
		ast.Walk(vis, source[j])
	}
	if vis.errs.Count() > 0 {
		return false, vis.errs
	}
	return true, nil
}

func (vis *checkScopingVisitor) Visit(node ast.Node) ast.Visitor {
	switch t := node.(type) {
	case *ast.Ident:
		if t.Name == "_" {
			return vis
		}
		s := vis.globalIdentMap.GetSymbol(t)
		if vis.declaredInExtracted.Contains(s) {
			if !vis.errs.Contains(s) {
				vis.errs.AddSymbol(s)
			}
		}
	}
	return vis
}

func CheckExtractMethodParameters(filename string, lineStart int, colStart int, lineEnd int, colEnd int, methodName string, recieverVarLine int, recieverVarCol int) (bool, *errors.GoRefactorError) {
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
	case !IsGoIdent(methodName):
		return false, errors.ArgumentError("methodName", "It's not a valid go identifier")
	}
	return true, nil
}

func makeStmtList(block *vector.Vector) []ast.Stmt {
	stmtList := make([]ast.Stmt, len(*block))
	for i, stmt := range *block {
		switch el := stmt.(type) {
		case ast.Stmt:
			stmtList[i] = el
		case ast.Expr:
			stmtList[i] = &ast.ReturnStmt{token.NoPos, []ast.Expr{el}}
		}
	}
	return stmtList
}

func makeCallExpr(name string, params *st.SymbolTable, pos token.Pos, recvSym *st.VariableSymbol, pack *st.Package, filename string) *ast.CallExpr {
	var Fun ast.Expr
	if recvSym != nil {
		x := ast.NewIdent(recvSym.Name())
		x.NamePos = pos
		Fun = &ast.SelectorExpr{x, ast.NewIdent(name)}
	} else {
		x := ast.NewIdent(name)
		x.NamePos = pos
		Fun = x

	}
	//Fun.NamePos = pos
	Args := params.ToAstExprSlice(pack, filename)
	return &ast.CallExpr{Fun, token.NoPos, Args, token.NoPos, token.NoPos}
}

func getIndexOfStmt(entry ast.Stmt, in []ast.Stmt) (int, bool) {
	for i, stmt := range in {
		if entry == stmt {
			return i, true
		}
	}
	return -1, false
}

func replaceStmtsWithCall(origin []ast.Stmt, replace []ast.Stmt, with *ast.CallExpr) []ast.Stmt {

	ind, found := getIndexOfStmt(replace[0], origin)

	if !found {
		panic("didn't find replace origin")
	}
	result := make([]ast.Stmt, ind)
	copy(result, origin[0:ind])
	result = append(result, &ast.ExprStmt{with})
	result = append(result, origin[ind+len(replace):]...)
	return result
}

func getRecieverSymbol(programTree *program.Program, pack *st.Package, filename string, recieverVarLine int, recieverVarCol int) (*st.VariableSymbol, *errors.GoRefactorError) {
	if recieverVarLine >= 0 && recieverVarLine >= 0 {
		rr, err := programTree.FindSymbolByPosition(filename, recieverVarLine, recieverVarCol)
		if err != nil {
			return nil, &errors.GoRefactorError{ErrorType: "extract method error", Message: "'recieverVarLine' and 'recieverVarCol' arguments don't point on an identifier"}
		}
		recvSym, ok := rr.(*st.VariableSymbol)
		if !ok {
			return nil, &errors.GoRefactorError{ErrorType: "extract method error", Message: "symbol, desired to be reciever, is not a variable symbol"}
		}
		if recvSym.VariableType.PackageFrom() != pack {
			return nil, &errors.GoRefactorError{ErrorType: "extract method error", Message: "reciever type is not from the same package as extracted code (not allowed to define methods for a type from imported package)"}
		}
		return recvSym, nil
	}
	return nil, nil
}

func getExtractedStatementList(pack *st.Package, file *ast.File, filename string, lineStart int, colStart int, lineEnd int, colEnd int) ([]ast.Stmt, ast.Node, *errors.GoRefactorError) {
	vis := &extractedSetVisitor{pack, new(vector.Vector), token.Position{filename, 0, lineStart, colStart}, token.Position{filename, 0, lineEnd, colEnd}, nil, nil, nil}
	ast.Walk(vis, file)
	if !vis.isValid() {
		return nil, nil, &errors.GoRefactorError{ErrorType: "extract method error", Message: "can't extract such set of statements"}
	}

	if checkForReturns(vis.resultBlock) {
		return nil, nil, &errors.GoRefactorError{ErrorType: "extract method error", Message: "can't extract code witn return statements"}
	}

	return makeStmtList(vis.resultBlock), vis.nodeFrom, nil
}

func getParametersAndDeclaredIn(pack *st.Package, stmtList []ast.Stmt, programTree *program.Program) (*st.SymbolTable, *st.SymbolTable) {
	parList, declared := getParameters(pack, stmtList, programTree.IdentMap)
	paramSymbolsMap := make(map[st.Symbol]bool)
	params := st.NewSymbolTable(pack)
	for _, sym := range parList {
		if _, ok := paramSymbolsMap[sym]; !ok {
			paramSymbolsMap[sym] = true
			params.AddSymbol(sym)
		}
	}
	return params, declared
}

func getResultTypeIfAny(programTree *program.Program, pack *st.Package, filename string, stmtList []ast.Stmt) st.ITypeSymbol {
	if rs, ok := stmtList[0].(*ast.ReturnStmt); ok {
		return packageParser.ParseExpr(rs.Results[0], pack, filename, programTree.IdentMap)
	}
	return nil
}

// Extracts a set of statements or expression to a method;
// start position - where the first statement starts;
// end position - where the last statement ends.
func ExtractMethod(programTree *program.Program, filename string, lineStart int, colStart int, lineEnd int, colEnd int, methodName string, recieverVarLine int, recieverVarCol int) (bool, *errors.GoRefactorError) {

	if ok, err := CheckExtractMethodParameters(filename, lineStart, colStart, lineEnd, colEnd, methodName, recieverVarLine, recieverVarCol); !ok {
		return false, err
	}

	pack, file := programTree.FindPackageAndFileByFilename(filename)
	if pack == nil {
		return false, errors.ArgumentError("filename", "Program packages don't contain file '"+filename+"'")
	}

	stmtList, nodeFrom, err := getExtractedStatementList(pack, file, filename, lineStart, colStart, lineEnd, colEnd)
	if err != nil {
		return false, err
	}

	params, declared := getParametersAndDeclaredIn(pack, stmtList, programTree)

	recvSym, err := getRecieverSymbol(programTree, pack, filename, recieverVarLine, recieverVarCol)
	if err != nil {
		return false, err
	}

	if recvSym != nil {
		if _, found := params.LookUp(recvSym.Name(), ""); !found {
			return false, &errors.GoRefactorError{ErrorType: "extract method error", Message: "symbol, desired to be reciever, is not a parameter to extracted code"}
		}
		params.RemoveSymbol(recvSym.Name())
	}

	result := getResultTypeIfAny(programTree, pack, filename, stmtList)
	results := st.NewSymbolTable(pack)
	if result != nil {
		results.AddSymbol(st.MakeVariable(st.NO_NAME, results, result))
	}

	fdecl := makeFuncDecl(methodName, stmtList, params, results, recvSym, pack, filename)
	callExpr := makeCallExpr(methodName, params, stmtList[0].Pos(), recvSym, pack, filename)

	if nodeFrom != nil {
		if ok, errs := checkScoping(nodeFrom, stmtList, declared, programTree.IdentMap); !ok {
			s := ""
			errs.ForEach(func(sym st.Symbol) {
				s += sym.Name() + " "
			})
			return false, &errors.GoRefactorError{ErrorType: "extract method error", Message: "extracted code declares symbols that are used in not-extracted code: " + s}
		}
		list := getStmtList(nodeFrom)
		newList := replaceStmtsWithCall(list, stmtList, callExpr)
		setStmtList(nodeFrom, newList)
	} else {
		rs := stmtList[0].(*ast.ReturnStmt)
		errs := replaceExpr(pack.FileSet.Position(rs.Results[0].Pos()), pack.FileSet.Position(rs.Results[0].End()), callExpr, pack, file)
		if err, ok := errs[EXTRACT_METHOD]; ok {
			return false, err
		}
	}
	file.Decls = append(file.Decls, fdecl)
	return true, nil

}
