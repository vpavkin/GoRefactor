package refactoring

import (
	"go/ast"
	"container/vector"
	"st"
	"go/token"
	//"fmt"
	"utils"
	"errors"
	"program"
	"go/printer"
	"os"
	"packageParser"
)

type extractedSetVisitor struct {
	Package *st.Package

	resultBlock  *vector.Vector
	firstNodePos token.Position //in
	lastNodePos  token.Position //in
	firstNode    ast.Node
	lastNode     ast.Node

	listFrom []ast.Stmt
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
				vis.listFrom = list
				vis.resultBlock.Push(stmt)

				if utils.ComparePosWithinFile(vis.Package.FileSet.Position(stmt.Pos()), vis.lastNodePos) == 0 {
					//one stmt method
					vis.lastNode = stmt
				}
			}
		} else {
			if !vis.isValid() {
				vis.resultBlock.Push(stmt)
				if utils.ComparePosWithinFile(vis.Package.FileSet.Position(stmt.Pos()), vis.lastNodePos) == 0 {
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
	case *ast.SwitchStmt:
		for _, cc := range t.Body.List {
			caseClause := cc.(*ast.CaseClause)
			vis.checkStmtList(caseClause.Body)
		}
	case *ast.TypeSwitchStmt:
		for _, cc := range t.Body.List {
			caseClause := cc.(*ast.TypeCaseClause)
			vis.checkStmtList(caseClause.Body)
		}
	case *ast.SelectStmt:
		for _, cc := range t.Body.List {
			caseClause := cc.(*ast.CommClause)
			vis.checkStmtList(caseClause.Body)
		}
// 	case *ast.BasicLit:
// 		return nil
	case ast.Expr:
		if utils.ComparePosWithinFile(vis.Package.FileSet.Position(t.Pos()), vis.lastNodePos) == 0 {
			if utils.ComparePosWithinFile(vis.firstNodePos, vis.lastNodePos) == 0 {
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
	symbols        st.IdentifierMap
	globalIdentMap st.IdentifierMap
}

func (vis *getParametersVisitor) declare(ident *ast.Ident) {
	if ident.Name != "_" {
		s := vis.globalIdentMap.GetSymbol(ident)
		vis.declared.AddSymbol(s)
	}
}
func (vis *getParametersVisitor) getInnerVisitor() *getParametersVisitor {
	inVis := &getParametersVisitor{vis.Package, st.NewSymbolTable(vis.Package), vis.symbols, vis.globalIdentMap}
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

		if t.Tok == token.DEFINE {
			n := t.Lhs.(*ast.Ident)
			inVis.declare(n)
		} else {
			if t.Lhs != nil {
				ast.Walk(inVis, t.Lhs)
			}
		}
		if t.Rhs != nil {
			ast.Walk(inVis, t.Rhs)
		}
		for _, stmt := range t.Body {
			ast.Walk(vis, stmt)
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
			vis.symbols.AddIdent(t, vis.globalIdentMap.GetSymbol(t))
		}
	}
	return vis
}

func getParameters(pack *st.Package, stmtList *vector.Vector, globalIdentMap st.IdentifierMap) st.IdentifierMap {
	vis := &getParametersVisitor{pack, st.NewSymbolTable(pack), make(map[*ast.Ident]st.Symbol), globalIdentMap}
	vis.declared.AddOpenedScope(pack.Symbols)
	for _, stmt := range *stmtList {
		ast.Walk(vis, stmt.(ast.Node))
	}
	return vis.symbols
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

func makeFuncDecl(name string, stmtList []ast.Stmt, params *st.SymbolTable, result st.ITypeSymbol,pack *st.Package,filename string) *ast.FuncDecl {

	flist := params.ToAstFieldList(pack,filename)
	ftype := &ast.FuncType{token.NoPos, flist, nil}
	if result != nil {
		ftype.Results = &ast.FieldList{token.NoPos, []*ast.Field{ &ast.Field{nil,nil,result.ToAstExpr(pack,filename),nil,nil}}, token.NoPos}
	}
	fbody := &ast.BlockStmt{token.NoPos, stmtList, token.NoPos}
	return &ast.FuncDecl{nil, nil, ast.NewIdent(name), ftype, fbody}
}

func ExtractMethod(programTree *program.Program, filename string, lineStart int, colStart int, lineEnd int, colEnd int, methodName string, recieverVarLine int, recieverVarCol int) (bool, *errors.GoRefactorError) {

	if ok, err := CheckExtractMethodParameters(filename, lineStart, colStart, lineEnd, colEnd, methodName, recieverVarLine, recieverVarCol); !ok {
		return false, err
	}

	pack, file := programTree.FindPackageAndFileByFilename(filename)
	if pack == nil {
		return false, errors.ArgumentError("filename", "Program packages don't contain file '"+filename+"'")
	}

	vis := &extractedSetVisitor{pack, new(vector.Vector), token.Position{filename, 0, lineStart, colStart}, token.Position{filename, 0, lineEnd, colEnd}, nil, nil, nil}
	ast.Walk(vis, file)
	if !vis.isValid() {
		return false, &errors.GoRefactorError{ErrorType: "extract method error", Message: "can't extract such set of statements"}
	}

	parList := getParameters(pack, vis.resultBlock, programTree.IdentMap)
	paramSymbolsMap := make(map[st.Symbol]bool)
	params := st.NewSymbolTable(pack)
	for _, sym := range parList {
		if _, ok := paramSymbolsMap[sym]; !ok {
			paramSymbolsMap[sym] = true
			params.AddSymbol(sym)
		}
	}

	stmtList := makeStmtList(vis.resultBlock)

	var result st.ITypeSymbol
	if rs, ok := stmtList[0].(*ast.ReturnStmt); ok {
		result = packageParser.ParseExpr(rs.Results[0], pack, filename, programTree.IdentMap)
	}
	fdecl := makeFuncDecl(methodName, stmtList, params, result,pack,filename)
	printer.Fprint(os.Stdout, token.NewFileSet(), fdecl)
	return true, nil

}
