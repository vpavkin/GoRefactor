package printerUtil

import (
	"go/token"
	"go/ast"
	"refactoring/utils"
)

type replaceNodeVisitor struct {
	fileSet     *token.FileSet
	pos         token.Position
	end         token.Position
	replaceWith ast.Node
	replaced    ast.Node
	found       bool
}

func deleteNode(fset *token.FileSet, pos, end token.Position, where ast.Node) (ast.Node, bool) {
	vis := &replaceNodeVisitor{fset, pos, end, nil, nil, false}
	ast.Walk(vis, where)
	return vis.replaced, vis.found
}
func replaceNode(fset *token.FileSet, pos, end token.Position, where ast.Node, with ast.Node) (ast.Node, bool) {
	vis := &replaceNodeVisitor{fset, pos, end, with, nil, false}
	ast.Walk(vis, where)
	return vis.replaced, vis.found
}

func (vis *replaceNodeVisitor) find(node ast.Node) bool {
	if node == nil || utils.IsNullReally(node) {
		return false
	}

	if (utils.ComparePosWithinFile(vis.pos, vis.fileSet.Position(node.Pos())) == 0) &&
		(utils.ComparePosWithinFile(vis.end, vis.fileSet.Position(node.End())) == 0) {
		vis.found = true
		vis.replaced = node
	}
	return vis.found
}

func (vis *replaceNodeVisitor) findInExprSlice(nodes []ast.Expr) int {
	for i, e := range nodes {
		if vis.find(e) {
			return i
		}
	}
	return -1
}

func (vis *replaceNodeVisitor) replaceInExprSlice(i int, nodes []ast.Expr) []ast.Expr {
	if vis.replaceWith == nil {
		res := make([]ast.Expr, len(nodes)-1)
		for j := 0; j < i; j++ {
			res[j] = nodes[j]
		}
		for j := i + 1; j < len(nodes); j++ {
			res[j-1] = nodes[j]
		}
		return res
	}
	nodes[i] = vis.replaceWith.(ast.Expr)
	return nodes
}

func (vis *replaceNodeVisitor) findInStmtSlice(nodes []ast.Stmt) int {
	for i, e := range nodes {
		if vis.find(e) {
			return i
		}
	}
	return -1
}

func (vis *replaceNodeVisitor) replaceInStmtSlice(i int, nodes []ast.Stmt) []ast.Stmt {
	if vis.replaceWith == nil {
		res := make([]ast.Stmt, len(nodes)-1)
		for j := 0; j < i; j++ {
			res[j] = nodes[j]
		}
		for j := i + 1; j < len(nodes); j++ {
			res[j-1] = nodes[j]
		}
		return res
	}
	nodes[i] = vis.replaceWith.(ast.Stmt)
	return nodes
}

func (vis *replaceNodeVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	if vis.found {
		return nil
	}
	return vis.visitNode(node)
}

func (vis *replaceNodeVisitor) visitNode(node ast.Node) ast.Visitor {
	switch t := node.(type) {
	case *ast.ArrayType:
		if vis.find(t.Elt) {
			t.Elt = vis.replaceWith.(ast.Expr)
			return nil
		}
		if vis.find(t.Len) {
			t.Len = vis.replaceWith.(ast.Expr)
			return nil
		}
	case *ast.AssignStmt:
		if i := vis.findInExprSlice(t.Lhs); i >= 0 {
			t.Lhs = vis.replaceInExprSlice(i, t.Lhs)
			return nil
		}
		if i := vis.findInExprSlice(t.Rhs); i >= 0 {
			t.Rhs = vis.replaceInExprSlice(i, t.Rhs)
			return nil
		}
	case *ast.BinaryExpr:
		if vis.find(t.X) {
			t.X = vis.replaceWith.(ast.Expr)
			return nil
		}
		if vis.find(t.Y) {
			t.Y = vis.replaceWith.(ast.Expr)
			return nil
		}
	case *ast.BlockStmt:
		if i := vis.findInStmtSlice(t.List); i >= 0 {
			t.List = vis.replaceInStmtSlice(i, t.List)
			return nil
		}
	case *ast.BranchStmt:
		if vis.find(t.Label) {
			t.Label = vis.replaceWith.(*ast.Ident)
			return nil
		}
	case *ast.CallExpr:
		if vis.find(t.Fun) {
			t.Fun = vis.replaceWith.(ast.Expr)
			return nil
		}
		if i := vis.findInExprSlice(t.Args); i >= 0 {
			t.Args = vis.replaceInExprSlice(i, t.Args)
			return nil
		}
	case *ast.CaseClause:
		if i := vis.findInExprSlice(t.List); i >= 0 {
			t.List = vis.replaceInExprSlice(i, t.List)
			return nil
		}
		if i := vis.findInStmtSlice(t.Body); i >= 0 {
			t.Body = vis.replaceInStmtSlice(i, t.Body)
			return nil
		}
	case *ast.ChanType:
		if vis.find(t.Value) {
			t.Value = vis.replaceWith.(ast.Expr)
			return nil
		}
	case *ast.CommClause:
		if vis.find(t.Comm) {
			t.Comm = vis.replaceWith.(ast.Stmt)
			return nil
		}
		if i := vis.findInStmtSlice(t.Body); i >= 0 {
			t.Body = vis.replaceInStmtSlice(i, t.Body)
			return nil
		}
	case *ast.CommentGroup:
		for i, e := range t.List {
			if vis.find(e) {
				if vis.replaceWith == nil {
					res := make([]*ast.Comment, len(t.List)-1)
					for j := 0; j < i; j++ {
						res[j] = t.List[j]
					}
					for j := i + 1; j < len(t.List); j++ {
						res[j-1] = t.List[j]
					}
					t.List = res
					return nil
				}
				t.List[i] = vis.replaceWith.(*ast.Comment)
				return nil
			}
		}
	case *ast.CompositeLit:
		if vis.find(t.Type) {
			t.Type = vis.replaceWith.(ast.Expr)
			return nil
		}
		if i := vis.findInExprSlice(t.Elts); i >= 0 {
			t.Elts = vis.replaceInExprSlice(i, t.Elts)
			return nil
		}
	case *ast.DeferStmt:
		if vis.find(t.Call) {
			t.Call = vis.replaceWith.(*ast.CallExpr)
			return nil
		}
	case *ast.Ellipsis:
		if vis.find(t.Elt) {
			t.Elt = vis.replaceWith.(ast.Expr)
			return nil
		}
	case *ast.ExprStmt:
		if vis.find(t.X) {
			t.X = vis.replaceWith.(ast.Expr)
			return nil
		}
	case *ast.Field:
		if vis.find(t.Type) {
			t.Type = vis.replaceWith.(ast.Expr)
			return nil
		}
		if vis.find(t.Doc) {
			t.Doc = vis.replaceWith.(*ast.CommentGroup)
			return nil
		}
		if vis.find(t.Comment) {
			t.Comment = vis.replaceWith.(*ast.CommentGroup)
			return nil
		}
		if vis.find(t.Tag) {
			t.Tag = vis.replaceWith.(*ast.BasicLit)
			return nil
		}
		for i, e := range t.Names {
			if vis.find(e) {
				if vis.replaceWith == nil {
					if len(t.Names) == 1 {
						t.Names = nil
						return nil
					}
					res := make([]*ast.Ident, len(t.Names)-1)
					for j := 0; j < i; j++ {
						res[j] = t.Names[j]
					}
					for j := i + 1; j < len(t.Names); j++ {
						res[j-1] = t.Names[j]
					}
					t.Names = res
					return nil
				}
				t.Names[i] = vis.replaceWith.(*ast.Ident)
				return nil
			}
		}
	case *ast.FieldList:
		for i, e := range t.List {
			if vis.find(e) {
				if vis.replaceWith == nil {
					res := make([]*ast.Field, len(t.List)-1)
					for j := 0; j < i; j++ {
						res[j] = t.List[j]
					}
					for j := i + 1; j < len(t.List); j++ {
						res[j-1] = t.List[j]
					}
					t.List = res
					return nil
				}
				t.List[i] = vis.replaceWith.(*ast.Field)
				return nil
			}
		}
	case *ast.File:
		if vis.find(t.Doc) {
			t.Doc = vis.replaceWith.(*ast.CommentGroup)
			return nil
		}
		if vis.find(t.Name) {
			t.Name = vis.replaceWith.(*ast.Ident)
			return nil
		}
		for i, e := range t.Decls {
			if vis.find(e) {
				if vis.replaceWith == nil {
					res := make([]ast.Decl, len(t.Decls)-1)
					for j := 0; j < i; j++ {
						res[j] = t.Decls[j]
					}
					for j := i + 1; j < len(t.Decls); j++ {
						res[j-1] = t.Decls[j]
					}
					t.Decls = res
					return nil
				}
				t.Decls[i] = vis.replaceWith.(ast.Decl)
				return nil
			}
		}
		for i, e := range t.Comments {
			if vis.find(e) {
				if vis.replaceWith == nil {
					res := make([]*ast.CommentGroup, len(t.Comments)-1)
					for j := 0; j < i; j++ {
						res[j] = t.Comments[j]
					}
					for j := i + 1; j < len(t.Comments); j++ {
						res[j-1] = t.Comments[j]
					}
					t.Comments = res
					return nil
				}
				t.Comments[i] = vis.replaceWith.(*ast.CommentGroup)
				return nil
			}
		}
	case *ast.ForStmt:
		if vis.find(t.Init) {
			t.Init = vis.replaceWith.(ast.Stmt)
			return nil
		}
		if vis.find(t.Post) {
			t.Post = vis.replaceWith.(ast.Stmt)
			return nil
		}
		if vis.find(t.Body) {
			t.Body = vis.replaceWith.(*ast.BlockStmt)
			return nil
		}
		if vis.find(t.Cond) {
			t.Cond = vis.replaceWith.(ast.Expr)
			return nil
		}
	case *ast.FuncDecl:
		if vis.find(t.Doc) {
			t.Doc = vis.replaceWith.(*ast.CommentGroup)
			return nil
		}
		if vis.find(t.Recv) {
			t.Recv = vis.replaceWith.(*ast.FieldList)
			return nil
		}
		if vis.find(t.Name) {
			t.Name = vis.replaceWith.(*ast.Ident)
			return nil
		}
		if vis.find(t.Type) {
			t.Type = vis.replaceWith.(*ast.FuncType)
			return nil
		}
		if vis.find(t.Body) {
			t.Body = vis.replaceWith.(*ast.BlockStmt)
			return nil
		}
	case *ast.FuncLit:
		if vis.find(t.Type) {
			t.Type = vis.replaceWith.(*ast.FuncType)
			return nil
		}
		if vis.find(t.Body) {
			t.Body = vis.replaceWith.(*ast.BlockStmt)
			return nil
		}
	case *ast.FuncType:
		if vis.find(t.Params) {
			t.Params = vis.replaceWith.(*ast.FieldList)
			return nil
		}
		if vis.find(t.Results) {
			t.Results = vis.replaceWith.(*ast.FieldList)
			return nil
		}
	case *ast.GenDecl:
		if vis.find(t.Doc) {
			t.Doc = vis.replaceWith.(*ast.CommentGroup)
			return nil
		}
		for i, e := range t.Specs {
			if vis.find(e) {
				if vis.replaceWith == nil {
					res := make([]ast.Spec, len(t.Specs)-1)
					for j := 0; j < i; j++ {
						res[j] = t.Specs[j]
					}
					for j := i + 1; j < len(t.Specs); j++ {
						res[j-1] = t.Specs[j]
					}
					t.Specs = res
					return nil
				}
				t.Specs[i] = vis.replaceWith.(ast.Spec)
				return nil
			}
		}
	case *ast.GoStmt:
		if vis.find(t.Call) {
			t.Call = vis.replaceWith.(*ast.CallExpr)
			return nil
		}
	case *ast.IfStmt:
		if vis.find(t.Cond) {
			t.Cond = vis.replaceWith.(ast.Expr)
			return nil
		}
		if vis.find(t.Init) {
			t.Init = vis.replaceWith.(ast.Stmt)
			return nil
		}
		if vis.find(t.Body) {
			t.Body = vis.replaceWith.(*ast.BlockStmt)
			return nil
		}
		if vis.find(t.Else) {
			t.Else = vis.replaceWith.(ast.Stmt)
			return nil
		}
	case *ast.ImportSpec:
		if vis.find(t.Doc) {
			t.Doc = vis.replaceWith.(*ast.CommentGroup)
			return nil
		}
		if vis.find(t.Name) {
			t.Name = vis.replaceWith.(*ast.Ident)
			return nil
		}
		if vis.find(t.Path) {
			t.Path = vis.replaceWith.(*ast.BasicLit)
			return nil
		}
		if vis.find(t.Comment) {
			t.Comment = vis.replaceWith.(*ast.CommentGroup)
			return nil
		}
	case *ast.IncDecStmt:
		if vis.find(t.X) {
			t.X = vis.replaceWith.(ast.Expr)
			return nil
		}
	case *ast.IndexExpr:
		if vis.find(t.X) {
			t.X = vis.replaceWith.(ast.Expr)
			return nil
		}
		if vis.find(t.Index) {
			t.Index = vis.replaceWith.(ast.Expr)
			return nil
		}
	case *ast.InterfaceType:
		if vis.find(t.Methods) {
			t.Methods = vis.replaceWith.(*ast.FieldList)
			return nil
		}
	case *ast.KeyValueExpr:
		if vis.find(t.Key) {
			t.Key = vis.replaceWith.(ast.Expr)
			return nil
		}
		if vis.find(t.Value) {
			t.Value = vis.replaceWith.(ast.Expr)
			return nil
		}
	case *ast.LabeledStmt:
		if vis.find(t.Label) {
			t.Label = vis.replaceWith.(*ast.Ident)
			return nil
		}
		if vis.find(t.Stmt) {
			t.Stmt = vis.replaceWith.(ast.Stmt)
			return nil
		}
	case *ast.MapType:
		if vis.find(t.Key) {
			t.Key = vis.replaceWith.(ast.Expr)
			return nil
		}
		if vis.find(t.Value) {
			t.Value = vis.replaceWith.(ast.Expr)
			return nil
		}
	case *ast.ParenExpr:
		if vis.find(t.X) {
			t.X = vis.replaceWith.(ast.Expr)
			return nil
		}
	case *ast.RangeStmt:
		if vis.find(t.Key) {
			t.Key = vis.replaceWith.(ast.Expr)
			return nil
		}
		if vis.find(t.Value) {
			t.Value = vis.replaceWith.(ast.Expr)
			return nil
		}
		if vis.find(t.X) {
			t.X = vis.replaceWith.(ast.Expr)
			return nil
		}
		if vis.find(t.Body) {
			t.Body = vis.replaceWith.(*ast.BlockStmt)
			return nil
		}
	case *ast.ReturnStmt:
		if i := vis.findInExprSlice(t.Results); i >= 0 {
			t.Results = vis.replaceInExprSlice(i, t.Results)
			return nil
		}
	case *ast.SelectStmt:
		if vis.find(t.Body) {
			t.Body = vis.replaceWith.(*ast.BlockStmt)
			return nil
		}
	case *ast.SelectorExpr:
		if vis.find(t.X) {
			t.X = vis.replaceWith.(ast.Expr)
			return nil
		}
		if vis.find(t.Sel) {
			t.Sel = vis.replaceWith.(*ast.Ident)
			return nil
		}
	case *ast.SendStmt:
		if vis.find(t.Chan) {
			t.Chan = vis.replaceWith.(ast.Expr)
			return nil
		}
		if vis.find(t.Value) {
			t.Value = vis.replaceWith.(ast.Expr)
			return nil
		}
	case *ast.SliceExpr:
		if vis.find(t.X) {
			t.X = vis.replaceWith.(ast.Expr)
			return nil
		}
		if vis.find(t.Low) {
			t.Low = vis.replaceWith.(ast.Expr)
			return nil
		}
		if vis.find(t.High) {
			t.High = vis.replaceWith.(ast.Expr)
			return nil
		}
	case *ast.StarExpr:
		if vis.find(t.X) {
			t.X = vis.replaceWith.(ast.Expr)
			return nil
		}
	case *ast.StructType:
		if vis.find(t.Fields) {
			t.Fields = vis.replaceWith.(*ast.FieldList)
			return nil
		}
	case *ast.SwitchStmt:
		if vis.find(t.Tag) {
			t.Tag = vis.replaceWith.(ast.Expr)
			return nil
		}
		if vis.find(t.Init) {
			t.Init = vis.replaceWith.(ast.Stmt)
			return nil
		}
		if vis.find(t.Body) {
			t.Body = vis.replaceWith.(*ast.BlockStmt)
			return nil
		}
	case *ast.TypeAssertExpr:
		if vis.find(t.X) {
			t.X = vis.replaceWith.(ast.Expr)
			return nil
		}
		if vis.find(t.Type) {
			t.Type = vis.replaceWith.(ast.Expr)
			return nil
		}
	case *ast.TypeSpec:
		if vis.find(t.Doc) {
			t.Doc = vis.replaceWith.(*ast.CommentGroup)
			return nil
		}
		if vis.find(t.Name) {
			t.Name = vis.replaceWith.(*ast.Ident)
			return nil
		}
		if vis.find(t.Type) {
			t.Type = vis.replaceWith.(ast.Expr)
			return nil
		}
		if vis.find(t.Comment) {
			t.Comment = vis.replaceWith.(*ast.CommentGroup)
			return nil
		}
	case *ast.TypeSwitchStmt:
		if vis.find(t.Init) {
			t.Init = vis.replaceWith.(ast.Stmt)
			return nil
		}
		if vis.find(t.Assign) {
			t.Assign = vis.replaceWith.(ast.Stmt)
			return nil
		}
		if vis.find(t.Body) {
			t.Body = vis.replaceWith.(*ast.BlockStmt)
			return nil
		}
	case *ast.ValueSpec:
		if vis.find(t.Doc) {
			t.Doc = vis.replaceWith.(*ast.CommentGroup)
			return nil
		}
		if vis.find(t.Comment) {
			t.Comment = vis.replaceWith.(*ast.CommentGroup)
			return nil
		}
		for i, e := range t.Names {
			if vis.find(e) {
				if vis.replaceWith == nil {
					if len(t.Names) == 1 {
						t.Names = nil
						return nil
					}
					res := make([]*ast.Ident, len(t.Names)-1)
					for j := 0; j < i; j++ {
						res[j] = t.Names[j]
					}
					for j := i + 1; j < len(t.Names); j++ {
						res[j-1] = t.Names[j]
					}
					t.Names = res
					return nil
				}
				t.Names[i] = vis.replaceWith.(*ast.Ident)
				return nil
			}
		}
		if vis.find(t.Type) {
			t.Type = vis.replaceWith.(ast.Expr)
			return nil
		}
		if i := vis.findInExprSlice(t.Values); i >= 0 {
			t.Values = vis.replaceInExprSlice(i, t.Values)
			return nil
		}
	}
	return vis
}
