package printerUtil

import (
	"go/token"
	"go/ast"
	"refactoring/utils"
)

type findNodeVisitor struct {
	fset   *token.FileSet
	stPos  token.Position
	endPos token.Position
	result ast.Node
}

func (vis *findNodeVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if node == nil || vis.result != nil {
		return nil
	}
	if utils.ComparePosWithinFile(vis.stPos, vis.fset.Position(node.Pos())) == 0 &&
		utils.ComparePosWithinFile(vis.endPos, vis.fset.Position(node.End())) == 0 {
		vis.result = node
		return nil
	}
	return vis
}

func findNode(fset *token.FileSet, file *ast.File, posStart, posEnd token.Position) ast.Node {
	vis := &findNodeVisitor{fset, posStart, posEnd, nil}
	ast.Walk(vis, file)
	return vis.result
}
