package printerUtil

import (
	"go/token"
	"go/ast"
)

type fixPositionsVisitor struct {
	sourceOrigin   token.Pos
	inc            int
	affectComments bool
	visitedNodes   map[ast.Node]bool
}

func (vis *fixPositionsVisitor) newPos(pos token.Pos) token.Pos {
	if pos <= vis.sourceOrigin {
		return pos
	}
	return token.Pos(int(pos) + vis.inc)
}
func (vis *fixPositionsVisitor) Visit(node ast.Node) ast.Visitor {

	if _, ok := vis.visitedNodes[node]; ok {
		return nil
	}
	vis.visitedNodes[node] = true

	if vis.affectComments {
		switch t := node.(type) {
		case *ast.Comment:
			t.Slash = vis.newPos(t.Slash)
			println("changed comment pos " + string(t.Text))
			return vis
		}
	}
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
	case *ast.BranchStmt:
		t.TokPos = vis.newPos(t.TokPos)
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
	case *ast.LabeledStmt:
		t.Colon = vis.newPos(t.Colon)
	case *ast.MapType:
		t.Map = vis.newPos(t.Map)
	case *ast.Package:
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
func fixPositions(sourceOrigin token.Pos, inc int, node ast.Node, affectComments bool) {

	vis := &fixPositionsVisitor{sourceOrigin, inc, affectComments, make(map[ast.Node]bool)}
	ast.Walk(vis, node)
}

func fixPositionsExcept(sourceOrigin token.Pos, inc int, node ast.Node, affectComments bool, except map[ast.Node]bool) {

	vis := &fixPositionsVisitor{sourceOrigin, inc, affectComments, except}
	ast.Walk(vis, node)
}
