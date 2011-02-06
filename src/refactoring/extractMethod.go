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

const (
	RENAME              string = "ren"
	EXTRACT_METHOD      = "exm"
	INLINE_METHOD       = "inm"
	EXTRACT_INTERFACE   = "exi"
	IMPLEMENT_INTERFACE = "imi"
	SORT                = "sort"
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

type getParametersVisitor struct {
	Package        *st.Package
	declared       *st.SymbolTable
	params         st.IdentifierMap
	globalIdentMap st.IdentifierMap
}

func (vis *getParametersVisitor) declare(ident *ast.Ident) {
	if ident.Name != "_" {
		s := vis.globalIdentMap.GetSymbol(ident)
		vis.declared.AddSymbol(s)
	}
}
func (vis *getParametersVisitor) getInnerVisitor() *getParametersVisitor {
	inVis := &getParametersVisitor{vis.Package, st.NewSymbolTable(vis.Package), vis.params, vis.globalIdentMap}
	inVis.declared.AddOpenedScope(vis.declared)
	return inVis
}
func (vis *getParametersVisitor) Visit(node ast.Node) ast.Visitor {

	switch t := node.(type) {

	case *ast.AssignStmt:
		if t.Tok == token.DEFINE {
			for _, nn := range t.Lhs {
				n := nn.(*ast.Ident)
				vis.declare(n)
			}
		} else {
			for _, exp := range t.Lhs {
				ast.Walk(vis, exp)
			}
		}
		for _, exp := range t.Rhs {
			ast.Walk(vis, exp)
		}
		return nil
	case *ast.SelectorExpr:
		if id, ok := t.X.(*ast.Ident); ok {

			s := vis.globalIdentMap.GetSymbol(id)

			switch sym := s.(type) {
			case *st.PackageSymbol:
			case st.ITypeSymbol:
			case *st.VariableSymbol:
				if sym.Scope() != vis.Package.Symbols {
					ast.Walk(vis, id)
				}
			}
		} else {
			ast.Walk(vis, t.X)
		}
		return nil
	case *ast.ValueSpec:
		for _, n := range t.Names {

			vis.declare(n)
		}
		return nil
	//case *ast.TypeSpec:
	case *ast.RangeStmt:

		inVis := vis.getInnerVisitor()

		if t.Tok == token.DEFINE {
			key := t.Key.(*ast.Ident)
			inVis.declare(key)
			if t.Value != nil {
				value := t.Value.(*ast.Ident)
				inVis.declare(value)
			}
		} else {
			ast.Walk(inVis, t.Key)
			ast.Walk(inVis, t.Value)
		}
		ast.Walk(inVis, t.X)
		ast.Walk(inVis, t.Body)
		return nil
	case *ast.CommClause:

		inVis := vis.getInnerVisitor()

		ast.Walk(inVis, t.Comm)
		for _, stmt := range t.Body {
			ast.Walk(inVis, stmt)
		}
		return nil
	case *ast.IfStmt:

		inVis := vis.getInnerVisitor()
		if t.Init != nil {
			ast.Walk(inVis, t.Init)
		}
		if t.Cond != nil {
			ast.Walk(inVis, t.Cond)
		}
		ww1 := inVis.getInnerVisitor()
		ww2 := inVis.getInnerVisitor()
		ast.Walk(ww1, t.Body)
		if t.Else != nil {
			ast.Walk(ww2, t.Else)
		}
		return nil
	case *ast.FuncLit:
		inVis := vis.getInnerVisitor()
		if t.Type.Params != nil {
			for _, f := range t.Type.Params.List {
				for _, n := range f.Names {
					inVis.declare(n)
				}
			}
		}
		if t.Type.Results != nil {
			for _, f := range t.Type.Results.List {
				for _, n := range f.Names {
					inVis.declare(n)
				}
			}
		}
		return inVis
	case *ast.SwitchStmt, *ast.ForStmt, *ast.SelectStmt, *ast.TypeSwitchStmt, *ast.CaseClause, *ast.TypeCaseClause:
		return vis.getInnerVisitor()
	case *ast.Ident:
		if _, found := vis.declared.LookUp(t.Name, ""); !found {
			vis.params.AddIdent(t, vis.globalIdentMap.GetSymbol(t))
		}
	}
	return vis
}

func getParameters(pack *st.Package, stmtList []ast.Stmt, globalIdentMap st.IdentifierMap) (st.IdentifierMap, *st.SymbolTable) {
	vis := &getParametersVisitor{pack, st.NewSymbolTable(pack), make(map[*ast.Ident]st.Symbol), globalIdentMap}
	vis.declared.AddOpenedScope(pack.Symbols)
	for _, stmt := range stmtList {
		ast.Walk(vis, stmt)
	}
	return vis.params, vis.declared
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

type replaceExprVisitor struct {
	Package *st.Package
	newExpr ast.Expr
	exprPos token.Position
	exprEnd token.Position
	found   bool
	errors  map[string]*errors.GoRefactorError
}

func replaceExpr(exprPos token.Position, exprEnd token.Position, newExpr ast.Expr, Package *st.Package, node ast.Node) map[string]*errors.GoRefactorError {
	vis := &replaceExprVisitor{Package, newExpr, exprPos, exprEnd, false, make(map[string]*errors.GoRefactorError)}
	ast.Walk(vis, node)
	return vis.errors
}

func (vis *replaceExprVisitor) find(expr ast.Expr) bool {
	if expr == nil {
		return false
	}
	vis.found = (utils.ComparePosWithinFile(vis.exprPos, vis.Package.FileSet.Position(expr.Pos())) == 0) &&
		(utils.ComparePosWithinFile(vis.exprEnd, vis.Package.FileSet.Position(expr.End())) == 0)
	return vis.found
}

func (vis *replaceExprVisitor) replaceInSlice(exprs []ast.Expr) bool {
	for i, e := range exprs {
		if vis.find(e) {
			exprs[i] = vis.newExpr
			return true
		}
	}
	return false
}

func (vis *replaceExprVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	if vis.found {
		return nil
	}

	switch t := node.(type) {
	case *ast.ArrayType:
		if vis.find(t.Elt) {
			t.Elt = vis.newExpr
		}
		if vis.find(t.Len) {
			t.Len = vis.newExpr
		}
		if vis.found {
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "calls not allowed in type definitions"}
			return nil
		}
	case *ast.AssignStmt:
		for i, e := range t.Lhs {
			if vis.find(e) {
				t.Lhs[i] = vis.newExpr
				vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "calls not allowed in left side of assignment"}
				return nil
			}
		}
		if vis.replaceInSlice(t.Rhs) {
			return nil
		}
	case *ast.BinaryExpr:
		if vis.find(t.X) {
			t.X = vis.newExpr
			return nil
		}
		if vis.find(t.Y) {
			t.Y = vis.newExpr
			return nil
		}
	case *ast.CallExpr:
		if vis.find(t.Fun) {
			t.Fun = vis.newExpr
			return nil
		}
		if vis.replaceInSlice(t.Args) {
			return nil
		}
	case *ast.CaseClause:
		if vis.replaceInSlice(t.Values) {
			return nil
		}
	case *ast.ChanType:
		if vis.find(t.Value) {
			t.Value = vis.newExpr
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "calls not allowed in type definitions"}
			return nil
		}
	case *ast.CommClause:
	case *ast.CompositeLit:
		if vis.find(t.Type) {
			t.Type = vis.newExpr
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "method can't return a type"}
			return nil
		}
		for i, e := range t.Elts {
			if vis.find(e) {
				t.Elts[i] = vis.newExpr
				vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "can't extract a key-value pair"}
				return nil
			}
		}
	case *ast.DeferStmt:
		if vis.find(t.Call) {
			t.Call = vis.newExpr.(*ast.CallExpr)
			return nil
		}
	case *ast.Ellipsis:
		if vis.find(t.Elt) {
			t.Elt = vis.newExpr
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "method can't return a type"}
			return nil
		}
	case *ast.ExprStmt:
		if vis.find(t.X) {
			t.X = vis.newExpr
			return nil
		}
	case *ast.Field:
		if vis.find(t.Type) {
			t.Type = vis.newExpr
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "method can't return a type"}
			return nil
		}
		// 		if vis.find(t.Tag) {
		// 			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "calls not allowed in field tags"}
		// 			return nil
		// 		}
		for i, e := range t.Names {
			if vis.find(e) {
				if id, ok := vis.newExpr.(*ast.Ident); ok {
					t.Names[i] = id
				}
				vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "calls not allowed in field names"}
				return nil
			}
		}
	case *ast.ForStmt:
		if vis.find(t.Cond) {
			t.Cond = vis.newExpr
			return nil
		}
	case *ast.FuncLit:
		if vis.find(t.Type) {
			if ft, ok := vis.newExpr.(*ast.FuncType); ok {
				t.Type = ft
			}
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "method can't return a type"}
			return nil
		}
	case *ast.GoStmt:
		if vis.find(t.Call) {
			t.Call = vis.newExpr.(*ast.CallExpr)
			return nil
		}
	case *ast.IfStmt:
		if vis.find(t.Cond) {
			t.Cond = vis.newExpr
			return nil
		}
	case *ast.IncDecStmt:
		if vis.find(t.X) {
			t.X = vis.newExpr
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "incrementing method result makes no sense"}
			return nil
		}
	case *ast.IndexExpr:
		if vis.find(t.X) {
			t.X = vis.newExpr
			return nil
		}
		if vis.find(t.Index) {
			t.Index = vis.newExpr
			return nil
		}
	case *ast.KeyValueExpr:
		if vis.find(t.Key) {
			t.Key = vis.newExpr
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "can't extract key-value pair key"}
			return nil
		}
		if vis.find(t.Value) {
			t.Value = vis.newExpr
			return nil
		}
	case *ast.MapType:
		if vis.find(t.Key) {
			t.Key = vis.newExpr
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "calls not allowed in type definitions"}
			return nil
		}
		if vis.find(t.Value) {
			t.Value = vis.newExpr
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "calls not allowed in type definitions"}
			return nil
		}
	case *ast.ParenExpr:
		if vis.find(t.X) {
			t.X = vis.newExpr
			return nil
		}
	case *ast.RangeStmt:
		if vis.find(t.Key) {
			t.Key = vis.newExpr
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "calls not allowed in left side of assignment"}
			return nil
		}
		if vis.find(t.Value) {
			t.Value = vis.newExpr
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "calls not allowed in left side of assignment"}
			return nil
		}
		if vis.find(t.X) {
			t.X = vis.newExpr
			return nil
		}
	case *ast.ReturnStmt:
		if vis.replaceInSlice(t.Results) {
			return nil
		}
	case *ast.SelectorExpr:
		if vis.find(t.X) {
			t.X = vis.newExpr
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "can't extract a part of selector expression"}
			return nil
		}
		if vis.find(t.Sel) {
			if id, ok := vis.newExpr.(*ast.Ident); ok {
				t.Sel = id
			}
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "can't extract a part of selector expression"}
			return nil
		}
	case *ast.SliceExpr:
		if vis.find(t.X) {
			t.X = vis.newExpr
			return nil
		}
		if vis.find(t.Low) {
			t.Low = vis.newExpr
			return nil
		}
		if vis.find(t.High) {
			t.High = vis.newExpr
			return nil
		}
	case *ast.StarExpr:
		if vis.find(t.X) {
			t.X = vis.newExpr
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "can't extract indirected code"}
			return nil
		}
	case *ast.SwitchStmt:
		if vis.find(t.Tag) {
			t.Tag = vis.newExpr
			return nil
		}
	case *ast.TypeAssertExpr:
		if vis.find(t.X) {
			t.X = vis.newExpr
			return nil
		}
		if vis.find(t.Type) {
			t.Type = vis.newExpr
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "method can't return a type"}
			return nil
		}
	case *ast.TypeCaseClause:
		for i, e := range t.Types {
			if vis.find(e) {
				t.Types[i] = vis.newExpr
				vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "method can't return a type"}
				return nil
			}
		}
	case *ast.TypeSpec:
		if vis.find(t.Name) {
			if id, ok := vis.newExpr.(*ast.Ident); ok {
				t.Name = id
			}
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "calls not allowed in type definitions"}
			return nil
		}
		if vis.find(t.Type) {
			t.Type = vis.newExpr
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "calls not allowed in type definitions"}
			return nil
		}
	case *ast.ValueSpec:
		for i, e := range t.Names {
			if vis.find(e) {
				if id, ok := vis.newExpr.(*ast.Ident); ok {
					t.Names[i] = id
				}
				vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "calls not allowed in variable definitions"}
				return nil
			}
		}
		if vis.find(t.Type) {
			t.Type = vis.newExpr
			vis.errors[EXTRACT_METHOD] = &errors.GoRefactorError{ErrorType: "extract method error", Message: "calls not allowed in type definitions"}
			return nil
		}
		if vis.replaceInSlice(t.Values) {
			return nil
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

func makeFuncDecl(name string, stmtList []ast.Stmt, params *st.SymbolTable, result st.ITypeSymbol, recvSym *st.VariableSymbol, pack *st.Package, filename string) *ast.FuncDecl {

	flist := params.ToAstFieldList(pack, filename)
	ftype := &ast.FuncType{token.NoPos, flist, nil}
	if result != nil {
		ftype.Results = &ast.FieldList{token.NoPos, []*ast.Field{&ast.Field{nil, nil, result.ToAstExpr(pack, filename), nil, nil}}, token.NoPos}
	}
	fbody := &ast.BlockStmt{token.NoPos, stmtList, token.NoPos}

	var recvList *ast.FieldList
	if recvSym != nil {
		recvSt := st.NewSymbolTable(pack)
		recvSt.AddSymbol(recvSym)
		recvList = recvSt.ToAstFieldList(pack, filename)
	}

	return &ast.FuncDecl{nil, recvList, ast.NewIdent(name), ftype, fbody}
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
func getStmtList(node ast.Node) []ast.Stmt {
	switch t := node.(type) {
	case *ast.BlockStmt:
		return t.List
	case *ast.CaseClause:
		return t.Body
	case *ast.TypeCaseClause:
		return t.Body
	case *ast.CommClause:
		return t.Body
	}
	panic("wrong node type")
}

func setStmtList(node ast.Node, stmtList []ast.Stmt) {
	switch t := node.(type) {
	case *ast.BlockStmt:
		t.List = stmtList
	case *ast.CaseClause:
		t.Body = stmtList
	case *ast.TypeCaseClause:
		t.Body = stmtList
	case *ast.CommClause:
		t.Body = stmtList
	default:
		panic("wrong node type")
	}
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

	fdecl := makeFuncDecl(methodName, stmtList, params, result, recvSym, pack, filename)
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
