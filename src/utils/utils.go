package utils

import (
	"os"
	"path"
	"strings"
	"go/token"
	"go/ast"
)

func GoFilter(f *os.FileInfo) bool {
	return IsGoFile(f.Name)
}

func IsGoFile(fileName string) bool {
	return (path.Ext(fileName) == ".go") && !(strings.HasSuffix(fileName, "_test.go"))
}

func ComparePosWithinFile(pos1 token.Position, pos2 token.Position) int {
	switch {
	case pos1.Line > pos2.Line:
		fallthrough
	case pos1.Line == pos2.Line && pos1.Column > pos2.Column:
		return 1
	case pos1.Line == pos2.Line && pos1.Column == pos2.Column:
		return 0
	case pos1.Line == pos2.Line && pos1.Column < pos2.Column:
		fallthrough
	case pos1.Line < pos2.Line:
		return -1
	}
	panic("invalid positions")
}

func CopyExprList(list []ast.Expr) []ast.Expr {
	res := make([]ast.Expr, len(list))
	for i, stmt := range list {
		res[i] = CopyAstNode(stmt).(ast.Expr)
	}
	return res
}
func CopyStmtList(list []ast.Stmt) []ast.Stmt {
	res := make([]ast.Stmt, len(list))
	for i, stmt := range list {
		res[i] = CopyAstNode(stmt).(ast.Stmt)
	}
	return res
}
func CopyIdentList(list []*ast.Ident) []*ast.Ident {
	res := make([]*ast.Ident, len(list))
	for i, stmt := range list {
		res[i] = CopyAstNode(stmt).(*ast.Ident)
	}
	return res
}
func CopyFieldList(list []*ast.Field) []*ast.Field {
	res := make([]*ast.Field, len(list))
	for i, stmt := range list {
		res[i] = CopyAstNode(stmt).(*ast.Field)
	}
	return res
}
func CopySpecList(list []ast.Spec) []ast.Spec {
	res := make([]ast.Spec, len(list))
	for i, stmt := range list {
		res[i] = CopyAstNode(stmt).(ast.Spec)
	}
	return res
}
func CopyAstNode(node ast.Node) ast.Node {
	if node == nil {
		return nil
	}
	switch t := node.(type) {
	case *ast.ArrayType:
		Len, _ := CopyAstNode(t.Len).(ast.Expr)
		Elt, _ := CopyAstNode(t.Elt).(ast.Expr)
		return &ast.ArrayType{t.Lbrack, Len, Elt}
	case *ast.AssignStmt:
		return &ast.AssignStmt{CopyExprList(t.Lhs), t.TokPos, t.Tok, CopyExprList(t.Rhs)}
	case *ast.BasicLit:
		value := make([]byte, len(t.Value))
		copy(value, t.Value)
		return &ast.BasicLit{t.ValuePos, t.Kind, value}
	case *ast.BinaryExpr:
		return &ast.BinaryExpr{CopyAstNode(t.X).(ast.Expr), t.OpPos, t.Op, CopyAstNode(t.Y).(ast.Expr)}
	case *ast.BlockStmt:
		return &ast.BlockStmt{t.Lbrace, CopyStmtList(t.List), t.Rbrace}
	//case *ast.BranchStmt:
	case *ast.CallExpr:
		return &ast.CallExpr{CopyAstNode(t.Fun).(ast.Expr), t.Lparen, CopyExprList(t.Args), t.Ellipsis, t.Rparen}
	case *ast.CaseClause:
		return &ast.CaseClause{t.Case, CopyExprList(t.Values), t.Colon, CopyStmtList(t.Body)}
	case *ast.ChanType:
		return &ast.ChanType{t.Begin, t.Dir, CopyAstNode(t.Value).(ast.Expr)}
	case *ast.CommClause:
		return &ast.CommClause{t.Case, CopyAstNode(t.Comm).(ast.Stmt), t.Colon, CopyStmtList(t.Body)}
	case *ast.Comment:
		text := make([]byte, len(t.Text))
		copy(text, t.Text)
		return &ast.Comment{t.Slash, text}
	case *ast.CommentGroup:
		if t == nil {
			return nil
		}
		text := make([]*ast.Comment, len(t.List))
		copy(text, t.List)
		return &ast.CommentGroup{text}
	case *ast.CompositeLit:
		return &ast.CompositeLit{CopyAstNode(t.Type).(ast.Expr), t.Lbrace, CopyExprList(t.Elts), t.Rbrace}
	case *ast.DeclStmt:
		return &ast.DeclStmt{CopyAstNode(t.Decl).(ast.Decl)}
	case *ast.DeferStmt:
		return &ast.DeferStmt{t.Defer, CopyAstNode(t.Call).(*ast.CallExpr)}
	case *ast.Ellipsis:
		return &ast.Ellipsis{t.Ellipsis, CopyAstNode(t.Elt).(ast.Expr)}
	case *ast.EmptyStmt:
		return &ast.EmptyStmt{t.Semicolon}
	case *ast.ExprStmt:
		return &ast.ExprStmt{CopyAstNode(t.X).(ast.Expr)}
	case *ast.Field:
		return &ast.Field{CopyAstNode(t.Doc).(*ast.CommentGroup), CopyIdentList(t.Names), CopyAstNode(t.Type).(ast.Expr), CopyAstNode(t.Tag).(*ast.BasicLit), CopyAstNode(t.Comment).(*ast.CommentGroup)}
	case *ast.FieldList:
		return &ast.FieldList{t.Opening, CopyFieldList(t.List), t.Closing}
	//case *ast.File:
	case *ast.ForStmt:
		return &ast.ForStmt{t.For, CopyAstNode(t.Init).(ast.Stmt), CopyAstNode(t.Cond).(ast.Expr), CopyAstNode(t.Post).(ast.Stmt), CopyAstNode(t.Body).(*ast.BlockStmt)}
	case *ast.FuncDecl:
		return &ast.FuncDecl{CopyAstNode(t.Doc).(*ast.CommentGroup), CopyAstNode(t.Recv).(*ast.FieldList), CopyAstNode(t.Name).(*ast.Ident), CopyAstNode(t.Type).(*ast.FuncType), CopyAstNode(t.Body).(*ast.BlockStmt)}
	case *ast.FuncLit:
		return &ast.FuncLit{CopyAstNode(t.Type).(*ast.FuncType), CopyAstNode(t.Body).(*ast.BlockStmt)}
	case *ast.FuncType:
		return &ast.FuncType{t.Func, CopyAstNode(t.Params).(*ast.FieldList), CopyAstNode(t.Results).(*ast.FieldList)}
	case *ast.GenDecl:
		return &ast.GenDecl{CopyAstNode(t.Doc).(*ast.CommentGroup), t.TokPos, t.Tok, t.Lparen, CopySpecList(t.Specs), t.Rparen}
	case *ast.GoStmt:
		return &ast.GoStmt{t.Go, CopyAstNode(t.Call).(*ast.CallExpr)}
	case *ast.Ident:
		return &ast.Ident{t.NamePos, t.Name + "", nil}
	case *ast.IfStmt:
		return &ast.IfStmt{t.If, CopyAstNode(t.Init).(ast.Stmt), CopyAstNode(t.Cond).(ast.Expr), CopyAstNode(t.Body).(*ast.BlockStmt), CopyAstNode(t.Else).(*ast.BlockStmt)}
	case *ast.ImportSpec:
		return &ast.ImportSpec{CopyAstNode(t.Doc).(*ast.CommentGroup), CopyAstNode(t.Name).(*ast.Ident), CopyAstNode(t.Path).(*ast.BasicLit), CopyAstNode(t.Comment).(*ast.CommentGroup)}
	case *ast.IncDecStmt:
		return &ast.IncDecStmt{CopyAstNode(t.X).(ast.Expr), t.TokPos, t.Tok}
	case *ast.IndexExpr:
		return &ast.IndexExpr{CopyAstNode(t.X).(ast.Expr), t.Lbrack, CopyAstNode(t.Index).(ast.Expr), t.Rbrack}
	case *ast.InterfaceType:
		return &ast.InterfaceType{t.Interface, CopyAstNode(t.Methods).(*ast.FieldList), t.Incomplete}
	case *ast.KeyValueExpr:
		return &ast.KeyValueExpr{CopyAstNode(t.Key).(ast.Expr), t.Colon, CopyAstNode(t.Value).(ast.Expr)}
	//case *ast.LabeledStmt:
	case *ast.MapType:
		return &ast.MapType{t.Map, CopyAstNode(t.Key).(ast.Expr), CopyAstNode(t.Value).(ast.Expr)}
	//case *ast.Package:
	case *ast.ParenExpr:
		return &ast.ParenExpr{t.Lparen, CopyAstNode(t.X).(ast.Expr), t.Rparen}
	case *ast.RangeStmt:
		return &ast.RangeStmt{t.For, CopyAstNode(t.Key).(ast.Expr), CopyAstNode(t.Value).(ast.Expr), t.TokPos, t.Tok, CopyAstNode(t.X).(ast.Expr), CopyAstNode(t.Body).(*ast.BlockStmt)}
	case *ast.ReturnStmt:
		return &ast.ReturnStmt{t.Return, CopyExprList(t.Results)}
	case *ast.SelectStmt:
		return &ast.SelectStmt{t.Select, CopyAstNode(t.Body).(*ast.BlockStmt)}
	case *ast.SelectorExpr:
		return &ast.SelectorExpr{CopyAstNode(t.X).(ast.Expr), CopyAstNode(t.Sel).(*ast.Ident)}
	case *ast.SendStmt:
		return &ast.SendStmt{CopyAstNode(t.Chan).(ast.Expr), t.Arrow, CopyAstNode(t.Value).(ast.Expr)}
	case *ast.SliceExpr:
		return &ast.SliceExpr{CopyAstNode(t.X).(ast.Expr), t.Lbrack, CopyAstNode(t.Low).(ast.Expr), CopyAstNode(t.High).(ast.Expr), t.Rbrack}
	case *ast.StarExpr:
		return &ast.StarExpr{t.Star, CopyAstNode(t.X).(ast.Expr)}
	case *ast.StructType:
		return &ast.StructType{t.Struct, CopyAstNode(t.Fields).(*ast.FieldList), t.Incomplete}
	case *ast.SwitchStmt:
		return &ast.SwitchStmt{t.Switch, CopyAstNode(t.Init).(ast.Stmt), CopyAstNode(t.Tag).(ast.Expr), CopyAstNode(t.Body).(*ast.BlockStmt)}
	case *ast.TypeAssertExpr:
		return &ast.TypeAssertExpr{CopyAstNode(t.X).(ast.Expr), CopyAstNode(t.Type).(ast.Expr)}
	case *ast.TypeCaseClause:
		return &ast.TypeCaseClause{t.Case, CopyExprList(t.Types), t.Colon, CopyStmtList(t.Body)}
	case *ast.TypeSpec:
		return &ast.TypeSpec{CopyAstNode(t.Doc).(*ast.CommentGroup), CopyAstNode(t.Name).(*ast.Ident), CopyAstNode(t.Type).(ast.Expr), CopyAstNode(t.Comment).(*ast.CommentGroup)}
	case *ast.TypeSwitchStmt:
		return &ast.TypeSwitchStmt{t.Switch, CopyAstNode(t.Init).(ast.Stmt), CopyAstNode(t.Assign).(ast.Stmt), CopyAstNode(t.Body).(*ast.BlockStmt)}
	case *ast.UnaryExpr:
		return &ast.UnaryExpr{t.OpPos, t.Op, CopyAstNode(t.X).(ast.Expr)}
	case *ast.ValueSpec:
		return &ast.ValueSpec{CopyAstNode(t.Doc).(*ast.CommentGroup), CopyIdentList(t.Names), CopyAstNode(t.Type).(ast.Expr), CopyExprList(t.Values), CopyAstNode(t.Comment).(*ast.CommentGroup)}
	}
	panic("can't copy node")
}
