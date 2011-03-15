package refactoring

import (
	"go/ast"
	"refactoring/st"
	"go/token"
	"refactoring/utils"
	"refactoring/errors"
	"unicode"
	"refactoring/program"
)

const (
	RENAME              string = "ren"
	EXTRACT_METHOD      = "exm"
	INLINE_METHOD       = "inm"
	EXTRACT_INTERFACE   = "exi"
	IMPLEMENT_INTERFACE = "imi"
	SORT                = "sort"
)

// get parameters

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

// replace expr

type replaceExprVisitor struct {
	Package     *st.Package
	newExpr     ast.Expr
	newExprList []ast.Expr
	exprPos     token.Position
	exprEnd     token.Position
	found       bool
	errors      map[string]*errors.GoRefactorError
	listMode    bool
}

func replaceExpr(exprPos token.Position, exprEnd token.Position, newExpr ast.Expr, Package *st.Package, node ast.Node) map[string]*errors.GoRefactorError {
	vis := &replaceExprVisitor{Package, newExpr, nil, exprPos, exprEnd, false, make(map[string]*errors.GoRefactorError), false}
	ast.Walk(vis, node)
	return vis.errors
}

func replaceExprList(listStart token.Position, listEnd token.Position, newList []ast.Expr, Package *st.Package, node ast.Node) map[string]*errors.GoRefactorError {
	vis := &replaceExprVisitor{Package, newList[0], newList, listStart, listEnd, false, make(map[string]*errors.GoRefactorError), true}
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

func (vis *replaceExprVisitor) findList(list []ast.Expr) bool {
	if len(list) == 0 {
		return false
	}
	vis.found = (utils.ComparePosWithinFile(vis.exprPos, vis.Package.FileSet.Position(list[0].Pos())) == 0) &&
		(utils.ComparePosWithinFile(vis.exprEnd, vis.Package.FileSet.Position(list[len(list)-1].End())) == 0)
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

	if vis.listMode {
		return vis.visitExprList(node)
	}
	return vis.visitExpr(node)
}

func (vis *replaceExprVisitor) visitExpr(node ast.Node) ast.Visitor {
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

func (vis *replaceExprVisitor) visitExprList(node ast.Node) ast.Visitor {
	switch t := node.(type) {
	case *ast.CallExpr:
		if vis.findList(t.Args) {
			t.Args = vis.newExprList
			return nil
		}
	case *ast.AssignStmt:
		if vis.findList(t.Rhs) {
			t.Rhs = vis.newExprList
			return nil
		}
	case *ast.ReturnStmt:
		if vis.findList(t.Results) {
			t.Results = vis.newExprList
			return nil
		}
	}
	if !vis.found {
		return vis.visitExpr(node)
	}
	return vis
}


// make *ast.FuncDecl

func makeFuncDecl(name string, stmtList []ast.Stmt, params *st.SymbolTable, results *st.SymbolTable, recvSym *st.VariableSymbol, pack *st.Package, filename string) *ast.FuncDecl {

	flist := params.ToAstFieldList(pack, filename)
	ftype := &ast.FuncType{token.NoPos, flist, nil}
	if results.Count() > 0 {
		ftype.Results = results.ToAstFieldList(pack, filename)
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

// get/set statement list

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

// is go ident

func IsGoIdent(name string) bool {

	if name == "_" || name == "nil" || name == "true" || name == "false" {
		return false
	}
	if st.IsPredeclaredIdentifier(name) {
		return false
	}

	if !(unicode.IsLetter(int(name[0])) || name[0] == '_') {
		return false
	}
	for i := 1; i < len(name); i++ {
		if !(unicode.IsLetter(int(name[i])) || unicode.IsDigit(int(name[i])) || name[i] == '_') {
			return false
		}
	}
	return true
}
