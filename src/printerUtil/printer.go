package printerUtil

import (
	"go/ast"
	//"go/printer"
	"go/token"
	"refactoring/errors"
	"fmt"
)

func getFileFromFileSet(fset *token.FileSet, filename string) *token.File {
	for f := range fset.Files() {
		if f.Name() == filename {
			return f
		}
	}
	return nil
}

func getLines(f *token.File) []int {
	lines := make([]int, 0, 20)
	l := -1
	for i := f.Base(); i < f.Size(); i++ {
		if f.Line(token.Pos(i)) > l {
			l = f.Line(token.Pos(i))
			lines = append(lines, f.Offset(token.Pos(i)))
		}
	}
	return lines
}

func getNodeLines(f *token.File, node ast.Node) (lines []int, firstLineNum int) {
	lines = []int{}
	firstLineNum = -1

	l := f.Line(node.Pos())
	for p := node.Pos(); p <= node.End(); p++ {
		if f.Line(p) > l {
			l = f.Line(p)
			if firstLineNum == -1 {
				firstLineNum = l
			}
			lines = append(lines, f.Offset(p))
		}
	}
	if (int(node.End()) < f.Size()-1) && f.Line(node.End() + 1) > l{
		lines = append(lines, f.Offset(node.End() + 1))
		if firstLineNum == -1 {
			firstLineNum = f.Line(node.End() + 1)
		}
	}
	return
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
	lines := getLines(tokFile)
	for i, offset := range lines {
		fmt.Printf("%d -> %s(%d)\n", i+1, fset.Position(tokFile.Pos(offset)), offset)
	}
	nodeLines, firstLine := getNodeLines(tokFile, node)
	if _, ok := deleteNode(fset, posStart, posEnd, file); !ok {
		return false, errors.PrinterError("didn't find node to delete")
	}
	idxs := make(map[int]bool)
	for i,cg := range file.Comments{
		if cg.Pos() >= node.Pos() && cg.End() <= node.End(){
			idxs[i] = true
		}
	}
	if len(idxs) > 0{
		newComments := make([]*ast.CommentGroup,len(file.Comments) - len(idxs))
		i,j:=0,0
		for i < len(file.Comments){
			if _,ok:= idxs[i];!ok{
				newComments[j] = file.Comments[i]
				j++
			}
			i++
		}
		file.Comments = newComments
	}
	
	inc := -int(node.End() - node.Pos())
	fmt.Printf("\n%v, %v, mod = %v\n", nodeLines, firstLine, inc)
	fixPositions(node.Pos(), inc, file, true)
	if len(nodeLines) > 0{
		fmt.Printf("Before: %v\n", lines)
		newLines := make([]int, 0, len(lines)-len(nodeLines))
		newLines = append(newLines, lines[:firstLine-1]...)
		newLines = append(newLines, lines[firstLine+len(nodeLines)-1:]...)
		for i:= firstLine - 1;i < len(newLines);i++{
			newLines[i] += inc
		}
		fmt.Printf("After: %v\n", newLines)
		tokFile.SetLines(newLines)
	}
	return true, nil
}
