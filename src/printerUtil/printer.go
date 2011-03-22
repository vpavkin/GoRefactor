package printerUtil

import (
	"go/ast"
	//"go/printer"
	"go/token"
	"refactoring/errors"
	"refactoring/utils"
	"refactoring/st"
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
	for i := f.Base(); i < f.Base()+f.Size(); i++ {
		if f.Line(token.Pos(i)) > l {
			l = f.Line(token.Pos(i))
			lines = append(lines, f.Offset(token.Pos(i)))
		}
	}
	return lines
}

func getNodeLines(f *token.File, node ast.Node, fileSize int) (lines []int, firstLineNum int) {
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
	print(node.End())
	print(" -> ")
	println(fileSize + f.Base() - 1)
	if (int(node.End()) == fileSize+f.Base()-1) || f.Line(node.End()+1) > l {
		lines = append(lines, f.Offset(node.End()+1))
		if firstLineNum == -1 {
			firstLineNum = f.Line(node.End() + 1)
		}
	}
	return
}

func deleteCommentsInRange(file *ast.File, posStart, posEnd token.Pos) {
	idxs := make(map[int]bool)
	for i, cg := range file.Comments {
		if cg.Pos() >= posStart && cg.End() <= posEnd {
			idxs[i] = true
		}
	}
	if len(idxs) > 0 {
		newComments := make([]*ast.CommentGroup, len(file.Comments)-len(idxs))
		i, j := 0, 0
		for i < len(file.Comments) {
			if _, ok := idxs[i]; !ok {
				newComments[j] = file.Comments[i]
				j++
			}
			i++
		}
		file.Comments = newComments
	}
}

func removeLinesOfNode(node ast.Node, lines, nodeLines []int, firstLine int) []int {
	fmt.Printf("nodeLines %v\n", nodeLines)
	fmt.Printf("firstline %d\n", firstLine)
	fmt.Printf("Before: %v\n", lines)
	inc := -int(node.End() - node.Pos())
	if len(nodeLines) > 0 {
		inc--
		newLines := make([]int, 0, len(lines)-len(nodeLines))
		newLines = append(newLines, lines[:firstLine-1]...)
		newLines = append(newLines, lines[firstLine+len(nodeLines)-1:]...)
		for i := firstLine - 1; i < len(newLines); i++ {
			newLines[i] += inc
		}
		fmt.Printf("After: %v\n", newLines)
		return newLines
	}
	for i := firstLine - 1; i < len(lines); i++ {
		lines[i] += inc
	}
	fmt.Printf("After: %v\n", lines)
	return lines
}

func addLinesOfNode(node ast.Node, lines, nodeLines []int, firstLine int) []int {
	fmt.Printf("Before: %v\n", lines)
	inc := int(node.End() - node.Pos())
	if len(nodeLines) > 0 {
		inc++
		newLines := make([]int, 0, len(lines)+len(nodeLines))
		newLines = append(newLines, lines[:firstLine-1]...)
		newLines = append(newLines, nodeLines...)
		newLines = append(newLines, lines[firstLine-1:]...)
		for i := firstLine - 1 + len(nodeLines); i < len(newLines); i++ {
			newLines[i] += inc
		}
		fmt.Printf("After: %v\n", newLines)
		return newLines
	}
	for i := firstLine - 1; i < len(lines); i++ {
		lines[i] += inc
	}
	fmt.Printf("After: %v\n", lines)
	return lines
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
	nodeLines, firstLine := getNodeLines(tokFile, node, tokFile.Size())
	if _, ok := deleteNode(fset, posStart, posEnd, file); !ok {
		return false, errors.PrinterError("didn't find node to delete")
	}

	deleteCommentsInRange(file, node.Pos(), node.End())

	inc := -int(node.End() - node.Pos())
	fmt.Printf("\n%v, %v, mod = %v\n", nodeLines, firstLine, inc)
	fixPositions(node.Pos(), inc, file, true)

	tokFile.SetLines(removeLinesOfNode(node, lines, nodeLines, firstLine))

	return true, nil
}

func printDecls(tf *token.File, f *ast.File) {
	for _, d := range f.Decls {
		fmt.Printf("> %d %d\n", tf.Offset(d.Pos()), tf.Offset(d.End()))
	}
}

func ReplaceNode(fset *token.FileSet, filename string, file *ast.File, posStart, posEnd token.Position, withFileSet *token.FileSet, withFileName string, withFile *ast.File, withPosStart, withPosEnd token.Position, identMap st.IdentifierMap) (bool, *token.FileSet, *ast.File, *errors.GoRefactorError) {
	tokFile := getFileFromFileSet(fset, filename)
	withTokFile := getFileFromFileSet(withFileSet, withFileName)
	if tokFile == nil {
		return false, nil, nil, errors.PrinterError("couldn't find file " + filename + " in fileset")
	}
	if withTokFile == nil {
		return false, nil, nil, errors.PrinterError("couldn't find file " + withFileName + " in fileset")
	}
	node := findNode(fset, file, posStart, posEnd)
	withNode := findNode(withFileSet, withFile, withPosStart, withPosEnd)
	if node == nil || withNode == nil {
		return false, nil, nil, errors.PrinterError("couldn't find node with given positions")
	}

	printDecls(tokFile, file)
	oldSize := tokFile.Size()
	withOldSize := withTokFile.Size()
	l, wl := int(node.End()-node.Pos()), int(withNode.End()-withNode.Pos())
	if wl > l {
		fmt.Printf("length to add = %d\n", wl-l)
		inc := -tokFile.Base()
		fmt.Printf("inc = %d\n", inc)
		fset, file = reparseFile(file, filename, wl-l, identMap)
		node = findNode(fset, file, posStart, posEnd)
		tokFile = getFileFromFileSet(fset, filename)
		if filename == withFileName {
			withFileSet, withFile, withTokFile = fset, file, tokFile
			withNode = findNode(withFileSet, withFile, withPosStart, withPosEnd)
		}
		lines := getLines(tokFile)
		fmt.Printf("linesCount = %d\n", len(lines))
		tokFile.SetLines(lines[:len(lines)-(wl-l)])
	}

	withNode = utils.CopyAstNode(withNode)
	lines := getLines(tokFile)
	for i, offset := range lines {
		fmt.Printf("%d -> %s(%d)\n", i+1, fset.Position(tokFile.Pos(offset)), offset)
	}
	nodeLines, firstLine := getNodeLines(tokFile, node, oldSize)
	withNodeLines, _ := getNodeLines(withTokFile, withNode, withOldSize)

	fmt.Printf("withnodeLines: %v\n", withNodeLines)
	mod := wl - l
	withMod := int(tokFile.Offset(node.Pos()) - withTokFile.Offset(withNode.Pos()))

	fmt.Printf("withMod: %v\n", withMod)
	for i, _ := range withNodeLines {
		withNodeLines[i] += withMod
	}

	newLines := removeLinesOfNode(node, lines, nodeLines, firstLine)
	tokFile.SetLines(addLinesOfNode(withNode, newLines, withNodeLines, firstLine))

	printDecls(tokFile, file)

	fmt.Printf("node -------- %d %d --------- %d %d\n", node.Pos(), node.End(), tokFile.Offset(node.Pos()), tokFile.Offset(node.End()))
	fmt.Printf("with -------- %d %d --------- %d %d\n", withNode.Pos(), withNode.End(), withTokFile.Offset(withNode.Pos()), withTokFile.Offset(withNode.End()))
	fixPositions(0, withMod, withNode, true)
	fmt.Printf("node -------- %d %d --------- %d %d\n", node.Pos(), node.End(), tokFile.Offset(node.Pos()), tokFile.Offset(node.End()))
	fmt.Printf("with -------- %d %d --------- %d %d\n", withNode.Pos(), withNode.End(), withTokFile.Offset(withNode.Pos()), withTokFile.Offset(withNode.End()))
	printDecls(tokFile, file)
	if _, ok := replaceNode(fset, posStart, posEnd, file, withNode); !ok {
		return false, nil, nil, errors.PrinterError("didn't find node to replace")
	}

	except := map[ast.Node]bool{withNode: true}

	fixPositionsExcept(withNode.Pos(), mod, file, true, except)
	printDecls(tokFile, file)
	return true, fset, file, nil
}
