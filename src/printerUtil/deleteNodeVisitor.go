package printerUtil

import (
	"go/token"
	"go/ast"
	"refactoring/utils"
)

type deleteNodeVisitor struct {
	fileSet *token.FileSet
	pos     token.Position
	end     token.Position
	result  ast.Node
	found   bool
}

func deleteNode(fset *token.FileSet, pos, end token.Position, node ast.Node) (ast.Node, bool) {
	vis := &deleteNodeVisitor{fset, pos, end, nil, false}
	ast.Walk(vis, node)
	return vis.result, vis.found
}

func (vis *deleteNodeVisitor) find(node ast.Node) bool {
	if node == nil {
		return false
	}
	switch t := node.(type) {
	case *ast.ArrayType:
		if t == nil {
			return false
		}
	case *ast.AssignStmt:
		if t == nil {
			return false
		}
	case *ast.BasicLit:
		if t == nil {
			return false
		}
	case *ast.BinaryExpr:
		if t == nil {
			return false
		}
	case *ast.BlockStmt:
		if t == nil {
			return false
		}
	case *ast.BranchStmt:
		if t == nil {
			return false
		}
	case *ast.CallExpr:
		if t == nil {
			return false
		}
	case *ast.CaseClause:
		if t == nil {
			return false
		}
	case *ast.ChanType:
		if t == nil {
			return false
		}
	case *ast.CommClause:
		if t == nil {
			return false
		}
	case *ast.Comment:
		if t == nil {
			return false
		}
	case *ast.CommentGroup:
		if t == nil {
			return false
		}
	case *ast.CompositeLit:
		if t == nil {
			return false
		}
	case *ast.DeferStmt:
		if t == nil {
			return false
		}
	case *ast.Ellipsis:
		if t == nil {
			return false
		}
	case *ast.ExprStmt:
		if t == nil {
			return false
		}
	case *ast.Field:
		if t == nil {
			return false
		}
	case *ast.FieldList:
		if t == nil {
			return false
		}
	case *ast.ForStmt:
		if t == nil {
			return false
		}
	case *ast.FuncDecl:
		if t == nil {
			return false
		}
	case *ast.FuncLit:
		if t == nil {
			return false
		}
	case *ast.FuncType:
		if t == nil {
			return false
		}
	case *ast.GenDecl:
		if t == nil {
			return false
		}
	case *ast.GoStmt:
		if t == nil {
			return false
		}
	case *ast.Ident:
		if t == nil {
			return false
		}
	case *ast.IfStmt:
		if t == nil {
			return false
		}
	case *ast.ImportSpec:
		if t == nil {
			return false
		}
	case *ast.IncDecStmt:
		if t == nil {
			return false
		}
	case *ast.IndexExpr:
		if t == nil {
			return false
		}
	case *ast.InterfaceType:
		if t == nil {
			return false
		}
	case *ast.KeyValueExpr:
		if t == nil {
			return false
		}
	case *ast.LabeledStmt:
		if t == nil {
			return false
		}
	case *ast.MapType:
		if t == nil {
			return false
		}
	case *ast.ParenExpr:
		if t == nil {
			return false
		}
	case *ast.RangeStmt:
		if t == nil {
			return false
		}
	case *ast.ReturnStmt:
		if t == nil {
			return false
		}
	case *ast.SelectStmt:
		if t == nil {
			return false
		}
	case *ast.SelectorExpr:
		if t == nil {
			return false
		}
	case *ast.SendStmt:
		if t == nil {
			return false
		}
	case *ast.SliceExpr:
		if t == nil {
			return false
		}
	case *ast.StarExpr:
		if t == nil {
			return false
		}
	case *ast.StructType:
		if t == nil {
			return false
		}
	case *ast.SwitchStmt:
		if t == nil {
			return false
		}
	case *ast.TypeAssertExpr:
		if t == nil {
			return false
		}
	case *ast.TypeCaseClause:
		if t == nil {
			return false
		}
	case *ast.TypeSpec:
		if t == nil {
			return false
		}
	case *ast.TypeSwitchStmt:
		if t == nil {
			return false
		}
	case *ast.ValueSpec:
		if t == nil {
			return false
		}
	}
	if (utils.ComparePosWithinFile(vis.pos, vis.fileSet.Position(node.Pos())) == 0) &&
		(utils.ComparePosWithinFile(vis.end, vis.fileSet.Position(node.End())) == 0) {
		vis.found = true
		vis.result = node
	}
	return vis.found
}

func (vis *deleteNodeVisitor) findInExprSlice(nodes []ast.Expr) int {
	for i, e := range nodes {
		if vis.find(e) {
			return i
		}
	}
	return -1
}

func (vis *deleteNodeVisitor) removeFromExprSlice(i int, nodes []ast.Expr) []ast.Expr {
	res := make([]ast.Expr, len(nodes)-1)
	for j := 0; j < i; j++ {
		res[j] = nodes[j]
	}
	for j := i + 1; j < len(nodes); j++ {
		res[j-1] = nodes[j]
	}
	return res
}

func (vis *deleteNodeVisitor) findInStmtSlice(nodes []ast.Stmt) int {
	for i, e := range nodes {
		if vis.find(e) {
			return i
		}
	}
	return -1
}

func (vis *deleteNodeVisitor) removeFromStmtSlice(i int, nodes []ast.Stmt) []ast.Stmt {
	res := make([]ast.Stmt, len(nodes)-1)
	for j := 0; j < i; j++ {
		res[j] = nodes[j]
	}
	for j := i + 1; j < len(nodes); j++ {
		res[j-1] = nodes[j]
	}
	return res
}

func (vis *deleteNodeVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	if vis.found {
		return nil
	}
	return vis.visitNode(node)
}

func (vis *deleteNodeVisitor) visitNode(node ast.Node) ast.Visitor {
	switch t := node.(type) {
	case *ast.ArrayType:
		if vis.find(t.Elt) {
			t.Elt = nil
			return nil
		}
		if vis.find(t.Len) {
			t.Len = nil
			return nil
		}
	case *ast.AssignStmt:
		if i := vis.findInExprSlice(t.Lhs); i >= 0 {
			t.Lhs = vis.removeFromExprSlice(i, t.Lhs)
			return nil
		}
		if i := vis.findInExprSlice(t.Rhs); i >= 0 {
			t.Rhs = vis.removeFromExprSlice(i, t.Rhs)
			return nil
		}
	case *ast.BinaryExpr:
		if vis.find(t.X) {
			t.X = nil
			return nil
		}
		if vis.find(t.Y) {
			t.Y = nil
			return nil
		}
	case *ast.BlockStmt:
		if i := vis.findInStmtSlice(t.List); i >= 0 {
			t.List = vis.removeFromStmtSlice(i, t.List)
			return nil
		}
	case *ast.BranchStmt:
		if vis.find(t.Label) {
			t.Label = nil
			return nil
		}
	case *ast.CallExpr:
		if vis.find(t.Fun) {
			t.Fun = nil
			return nil
		}
		if i := vis.findInExprSlice(t.Args); i >= 0 {
			t.Args = vis.removeFromExprSlice(i, t.Args)
			return nil
		}
	case *ast.CaseClause:
		if i := vis.findInExprSlice(t.Values); i >= 0 {
			t.Values = vis.removeFromExprSlice(i, t.Values)
			return nil
		}
		if i := vis.findInStmtSlice(t.Body); i >= 0 {
			t.Body = vis.removeFromStmtSlice(i, t.Body)
			return nil
		}
	case *ast.ChanType:
		if vis.find(t.Value) {
			t.Value = nil
			return nil
		}
	case *ast.CommClause:
		if vis.find(t.Comm) {
			t.Comm = nil
			return nil
		}
		if i := vis.findInStmtSlice(t.Body); i >= 0 {
			t.Body = vis.removeFromStmtSlice(i, t.Body)
			return nil
		}
	case *ast.CommentGroup:
		for i, e := range t.List {
			if vis.find(e) {
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
		}
	case *ast.CompositeLit:
		if vis.find(t.Type) {
			t.Type = nil
			return nil
		}
		if i := vis.findInExprSlice(t.Elts); i >= 0 {
			t.Elts = vis.removeFromExprSlice(i, t.Elts)
			return nil
		}
	case *ast.DeferStmt:
		if vis.find(t.Call) {
			t.Call = nil
			return nil
		}
	case *ast.Ellipsis:
		if vis.find(t.Elt) {
			t.Elt = nil
			return nil
		}
	case *ast.ExprStmt:
		if vis.find(t.X) {
			t.X = nil
			return nil
		}
	case *ast.Field:
		if vis.find(t.Type) {
			t.Type = nil
			return nil
		}
		if vis.find(t.Doc) {
			t.Doc = nil
			return nil
		}
		if vis.find(t.Comment) {
			t.Comment = nil
			return nil
		}
		if vis.find(t.Tag) {
			t.Tag = nil
			return nil
		}
		for i, e := range t.Names {
			if vis.find(e) {
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
		}
	case *ast.FieldList:
		for i, e := range t.List {
			if vis.find(e) {
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
		}
	case *ast.File:
		if vis.find(t.Doc) {
			t.Doc = nil
			return nil
		}
		if vis.find(t.Name) {
			t.Name = nil
			return nil
		}
		for i, e := range t.Decls {
			if vis.find(e) {
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
		}
		for i, e := range t.Comments {
			if vis.find(e) {
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
		}
	case *ast.ForStmt:
		if vis.find(t.Init) {
			t.Init = nil
			return nil
		}
		if vis.find(t.Post) {
			t.Post = nil
			return nil
		}
		if vis.find(t.Body) {
			t.Body = nil
			return nil
		}
		if vis.find(t.Cond) {
			t.Cond = nil
			return nil
		}
	case *ast.FuncDecl:
		if vis.find(t.Doc) {
			t.Doc = nil
			return nil
		}
		if vis.find(t.Recv) {
			t.Recv = nil
			return nil
		}
		if vis.find(t.Name) {
			t.Name = nil
			return nil
		}
		if vis.find(t.Type) {
			t.Type = nil
			return nil
		}
		if vis.find(t.Body) {
			t.Body = nil
			return nil
		}
	case *ast.FuncLit:
		if vis.find(t.Type) {
			t.Type = nil
			return nil
		}
		if vis.find(t.Body) {
			t.Body = nil
			return nil
		}
	case *ast.FuncType:
		if vis.find(t.Params) {
			t.Params = nil
			return nil
		}
		if vis.find(t.Results) {
			t.Results = nil
			return nil
		}
	case *ast.GenDecl:
		if vis.find(t.Doc) {
			t.Doc = nil
			return nil
		}
		for i, e := range t.Specs {
			if vis.find(e) {
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
		}
	case *ast.GoStmt:
		if vis.find(t.Call) {
			t.Call = nil
			return nil
		}
	case *ast.IfStmt:
		if vis.find(t.Cond) {
			t.Cond = nil
			return nil
		}
		if vis.find(t.Init) {
			t.Init = nil
			return nil
		}
		if vis.find(t.Body) {
			t.Body = nil
			return nil
		}
		if vis.find(t.Else) {
			t.Else = nil
			return nil
		}
	case *ast.ImportSpec:
		if vis.find(t.Doc) {
			t.Doc = nil
			return nil
		}
		if vis.find(t.Name) {
			t.Name = nil
			return nil
		}
		if vis.find(t.Path) {
			t.Path = nil
			return nil
		}
		if vis.find(t.Comment) {
			t.Comment = nil
			return nil
		}
	case *ast.IncDecStmt:
		if vis.find(t.X) {
			t.X = nil
			return nil
		}
	case *ast.IndexExpr:
		if vis.find(t.X) {
			t.X = nil
			return nil
		}
		if vis.find(t.Index) {
			t.Index = nil
			return nil
		}
	case *ast.InterfaceType:
		if vis.find(t.Methods) {
			t.Methods = nil
			return nil
		}
	case *ast.KeyValueExpr:
		if vis.find(t.Key) {
			t.Key = nil
			return nil
		}
		if vis.find(t.Value) {
			t.Value = nil
			return nil
		}
	case *ast.LabeledStmt:
		if vis.find(t.Label) {
			t.Label = nil
			return nil
		}
		if vis.find(t.Stmt) {
			t.Stmt = nil
			return nil
		}
	case *ast.MapType:
		if vis.find(t.Key) {
			t.Key = nil
			return nil
		}
		if vis.find(t.Value) {
			t.Value = nil
			return nil
		}
	case *ast.ParenExpr:
		if vis.find(t.X) {
			t.X = nil
			return nil
		}
	case *ast.RangeStmt:
		if vis.find(t.Key) {
			t.Key = nil
			return nil
		}
		if vis.find(t.Value) {
			t.Value = nil
			return nil
		}
		if vis.find(t.X) {
			t.X = nil
			return nil
		}
		if vis.find(t.Body) {
			t.Body = nil
			return nil
		}
	case *ast.ReturnStmt:
		if i := vis.findInExprSlice(t.Results); i >= 0 {
			t.Results = vis.removeFromExprSlice(i, t.Results)
			return nil
		}
	case *ast.SelectStmt:
		if vis.find(t.Body) {
			t.Body = nil
			return nil
		}
	case *ast.SelectorExpr:
		if vis.find(t.X) {
			t.X = nil
			return nil
		}
		if vis.find(t.Sel) {
			t.Sel = nil
			return nil
		}
	case *ast.SendStmt:
		if vis.find(t.Chan) {
			t.Chan = nil
			return nil
		}
		if vis.find(t.Value) {
			t.Value = nil
			return nil
		}
	case *ast.SliceExpr:
		if vis.find(t.X) {
			t.X = nil
			return nil
		}
		if vis.find(t.Low) {
			t.Low = nil
			return nil
		}
		if vis.find(t.High) {
			t.High = nil
			return nil
		}
	case *ast.StarExpr:
		if vis.find(t.X) {
			t.X = nil
			return nil
		}
	case *ast.StructType:
		if vis.find(t.Fields) {
			t.Fields = nil
			return nil
		}
	case *ast.SwitchStmt:
		if vis.find(t.Tag) {
			t.Tag = nil
			return nil
		}
		if vis.find(t.Init) {
			t.Init = nil
			return nil
		}
		if vis.find(t.Body) {
			t.Body = nil
			return nil
		}
	case *ast.TypeAssertExpr:
		if vis.find(t.X) {
			t.X = nil
			return nil
		}
		if vis.find(t.Type) {
			t.Type = nil
			return nil
		}
	case *ast.TypeCaseClause:
		if i := vis.findInExprSlice(t.Types); i >= 0 {
			t.Types = vis.removeFromExprSlice(i, t.Types)
			return nil
		}
		if i := vis.findInStmtSlice(t.Body); i >= 0 {
			t.Body = vis.removeFromStmtSlice(i, t.Body)
			return nil
		}
	case *ast.TypeSpec:
		if vis.find(t.Doc) {
			t.Doc = nil
			return nil
		}
		if vis.find(t.Name) {
			t.Name = nil
			return nil
		}
		if vis.find(t.Type) {
			t.Type = nil
			return nil
		}
		if vis.find(t.Comment) {
			t.Comment = nil
			return nil
		}
	case *ast.TypeSwitchStmt:
		if vis.find(t.Init) {
			t.Init = nil
			return nil
		}
		if vis.find(t.Assign) {
			t.Assign = nil
			return nil
		}
		if vis.find(t.Body) {
			t.Body = nil
			return nil
		}
	case *ast.ValueSpec:
		if vis.find(t.Doc) {
			t.Doc = nil
			return nil
		}
		if vis.find(t.Comment) {
			t.Comment = nil
			return nil
		}
		for i, e := range t.Names {
			if vis.find(e) {
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
		}
		if vis.find(t.Type) {
			t.Type = nil
			return nil
		}
		if i := vis.findInExprSlice(t.Values); i >= 0 {
			t.Values = vis.removeFromExprSlice(i, t.Values)
			return nil
		}
	}
	return vis
}
