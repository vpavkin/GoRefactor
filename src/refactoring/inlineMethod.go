package refactoring

import (
	"program"
	"st"
	"utils"
	"errors"
	"go/ast"
	"go/token"
	"fmt"
)

type getCallVisitor struct {
	Package *st.Package

	callStart token.Position
	callEnd   token.Position

	CallNode ast.Node
}

func (vis *getCallVisitor) found() bool {
	return vis.CallNode != nil
}

func (vis *getCallVisitor) find(n ast.Node) bool {
	return utils.ComparePosWithinFile(vis.Package.FileSet.Position(n.Pos()), vis.callStart) == 0 &&
		utils.ComparePosWithinFile(vis.Package.FileSet.Position(n.End()), vis.callEnd) == 0
}

func (vis *getCallVisitor) Visit(node ast.Node) ast.Visitor {
	if vis.found() {
		return nil
	}
	switch t := node.(type) {
	case ast.Stmt, ast.Expr:
		if vis.find(t) {
			vis.CallNode = t
			return nil
		}
	}
	return vis
}

func getCall(pack *st.Package,file *ast.File, filename string, lineStart int,colStart int, lineEnd int, colEnd int) ast.Node{
	vis := &getCallVisitor{pack,token.Position{filename,0,lineStart,colStart},token.Position{filename,0,lineEnd,colEnd},nil}
	ast.Walk(vis,file);
	return vis.CallNode;
}

func getCallExpr(callNode ast.Node) (*ast.CallExpr,*errors.GoRefactorError){
	switch node := callNode.(type){
		case *ast.ExprStmt:
			if res,ok := node.X.(*ast.CallExpr); ok{
				return res,nil
			}
		case *ast.CallExpr:
			return node,nil
	}
	return nil,&errors.GoRefactorError{ErrorType: "inline method error", Message: "selected statement or expression is not a function call"}
}

func getMethodSymbol(programTree *program.Program, callExpr *ast.CallExpr) (*st.FunctionSymbol,*errors.GoRefactorError){
	var s st.Symbol
	switch funExpr := callExpr.Fun.(type){
		case *ast.Ident:
			s = programTree.IdentMap.GetSymbol(funExpr)
		case *ast.SelectorExpr:
			s = programTree.IdentMap.GetSymbol(funExpr.Sel)
		default:
			panic("function expression is not a selector or ident")
	}
	if res,ok := s.(*st.FunctionSymbol);ok{
		return res,nil;
	}
	panic("Call expr Function is not a st.FunctionSymbol")
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
	callNode := getCall(pack,file,filename,lineStart,colStart,lineEnd,colEnd);
	callExpr,err := getCallExpr(callNode);
	if err != nil{
		return false,err
	}
	funSym,err := getMethodSymbol(programTree,callExpr)
	if err != nil{
		return false,err
	}
	if funSym.PackageFrom() != pack{
		return false,&errors.GoRefactorError{ErrorType: "inline method error", Message: "can't inline method from other package"}
	}
	
	fmt.Println(funSym.Name());
	return false, errors.ArgumentError("blah", "blah blah blah")
}
