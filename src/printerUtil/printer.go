package printerUtil

import (
	"go/ast"
	//"go/printer"
	"go/token"
	"refactoring/errors"
	"refactoring/utils"
	"fmt"
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

func getFileFromFileSet(fset *token.FileSet, filename string) *token.File {
	for f := range fset.Files() {
		if f.Name() == filename {
			return f
		}
	}
	return nil
}

func getNodeLines(file *token.File, node ast.Node) []token.Pos {
	result := []token.Pos{}
	if int(node.Pos()) == file.Base() {
		result = append(result, node.Pos())
	}
	l := file.Line(node.Pos())
	for p := node.Pos(); p < node.End(); p++ {
		if file.Line(p) > l {
			l = file.Line(p)
			result = append(result, p)
		}
	}
	return result
}

func DeleteNode(fset *token.FileSet, filename string, file *ast.File, posStart, posEnd token.Position) (bool, *errors.GoRefactorError) {
	tokFile := getFileFromFileSet(fset, filename)
	if tokFile == nil {
		return false, errors.PrinterError("couldn't find file " + filename + " in fileset")
	}
	node := findNode(fset, file, posStart, posEnd)
	if node == nil {
		return false, errors.PrinterError("couldn't find node with given positions")
	}
	nodeLines := getNodeLines(tokFile, node)
	fmt.Printf("%v\n", nodeLines)
	return false, errors.PrinterError("blah blah blah")
}
