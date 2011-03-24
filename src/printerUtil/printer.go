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

func getRangeLines(f *token.File, Pos, End token.Pos, fileSize int) (lines []int, firstLineNum int) {
	lines = []int{}
	firstLineNum = -1

	l := f.Line(Pos)
	for p := Pos; p <= End; p++ {
		if f.Line(p) > l {
			l = f.Line(p)
			if firstLineNum == -1 {
				firstLineNum = l
			}
			lines = append(lines, f.Offset(p))
		}
	}
	print(End)
	print(" -> ")
	println(fileSize + f.Base() - 1)
	if (int(End) == fileSize+f.Base()-1) || f.Line(End+1) > l {
		lines = append(lines, f.Offset(End+1))
		if firstLineNum == -1 {
			firstLineNum = f.Line(End + 1)
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

func removeLinesOfRange(Pos, End token.Pos, lines, rangeLines []int, firstLine int) []int {
	fmt.Printf("nodeLines %v\n", rangeLines)
	fmt.Printf("firstline %d\n", firstLine)
	fmt.Printf("Before: %v\n", lines)
	inc := -int(End - Pos)
	if len(rangeLines) > 0 {
		inc--
		newLines := make([]int, 0, len(lines)-len(rangeLines))
		newLines = append(newLines, lines[:firstLine-1]...)
		newLines = append(newLines, lines[firstLine+len(rangeLines)-1:]...)
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

func addLinesOfRange(Pos, End token.Pos, lines, rangeLines []int, firstLine int) []int {
	fmt.Printf("Before: %v\n", lines)
	inc := int(End - Pos)
	if len(rangeLines) > 0 {
		inc++
		newLines := make([]int, 0, len(lines)+len(rangeLines))
		if firstLine >= 0 {
			newLines = append(newLines, lines[:firstLine-1]...)
			newLines = append(newLines, rangeLines...)
			newLines = append(newLines, lines[firstLine-1:]...)
			for i := firstLine - 1 + len(rangeLines); i < len(newLines); i++ {
				newLines[i] += inc
			}
		} else {
			newLines = append(newLines, lines...)
			newLines = append(newLines, rangeLines...)
			for i := len(lines); i < len(newLines); i++ {
				newLines[i] += inc
			}
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
	nodeLines, firstLine := getRangeLines(tokFile, node.Pos(), node.End(), tokFile.Size())
	if _, ok := deleteNode(fset, posStart, posEnd, file); !ok {
		return false, errors.PrinterError("didn't find node to delete")
	}

	deleteCommentsInRange(file, node.Pos(), node.End())

	inc := -int(node.End() - node.Pos())
	fmt.Printf("\n%v, %v, mod = %v\n", nodeLines, firstLine, inc)
	FixPositions(node.Pos(), inc, file, true)

	tokFile.SetLines(removeLinesOfRange(node.Pos(), node.End(), lines, nodeLines, firstLine))

	return true, nil
}

func DeleteNodeList(fset *token.FileSet, filename string, file *ast.File, list interface{}) (bool, *errors.GoRefactorError) {
	tokFile := getFileFromFileSet(fset, filename)
	if tokFile == nil {
		return false, errors.PrinterError("couldn't find file " + filename + " in fileset")
	}
	lines := getLines(tokFile)
	for i, offset := range lines {
		fmt.Printf("%d -> %s(%d)\n", i+1, fset.Position(tokFile.Pos(offset)), offset)
	}
	var pos, end token.Pos

	switch t := list.(type) {
	case []ast.Stmt:
		pos, end = t[0].Pos(), t[len(t)-1].End()
	}

	rangeLines, firstLine := getRangeLines(tokFile, pos, end, tokFile.Size())
	switch t := list.(type) {
	case []ast.Stmt:
		for _, n := range t {
			if _, ok := deleteNode(fset, fset.Position(n.Pos()), fset.Position(n.End()), file); !ok {
				return false, errors.PrinterError("didn't find node to delete")
			}
		}
	}

	deleteCommentsInRange(file, pos, end)

	inc := -int(end - pos)

	fmt.Printf("\n%v, %v, mod = %v\n", rangeLines, firstLine, inc)
	FixPositions(pos, inc, file, true)

	tokFile.SetLines(removeLinesOfRange(pos, end, lines, rangeLines, firstLine))

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
	nodeLines, firstLine := getRangeLines(tokFile, node.Pos(), node.End(), oldSize)
	withNodeLines, _ := getRangeLines(withTokFile, withNode.Pos(), withNode.End(), withOldSize)

	fmt.Printf("withnodeLines: %v\n", withNodeLines)
	mod := wl - l
	withMod := int(tokFile.Offset(node.Pos()) - withTokFile.Offset(withNode.Pos()))

	fmt.Printf("withMod: %v\n", withMod)
	for i, _ := range withNodeLines {
		withNodeLines[i] += withMod
	}

	newLines := removeLinesOfRange(node.Pos(), node.End(), lines, nodeLines, firstLine)
	tokFile.SetLines(addLinesOfRange(withNode.Pos(), withNode.End(), newLines, withNodeLines, firstLine))

	printDecls(tokFile, file)

	fmt.Printf("node -------- %d %d --------- %d %d\n", node.Pos(), node.End(), tokFile.Offset(node.Pos()), tokFile.Offset(node.End()))
	fmt.Printf("with -------- %d %d --------- %d %d\n", withNode.Pos(), withNode.End(), withTokFile.Offset(withNode.Pos()), withTokFile.Offset(withNode.End()))
	FixPositions(0, withMod, withNode, true)
	fmt.Printf("node -------- %d %d --------- %d %d\n", node.Pos(), node.End(), tokFile.Offset(node.Pos()), tokFile.Offset(node.End()))
	fmt.Printf("with -------- %d %d --------- %d %d\n", withNode.Pos(), withNode.End(), withTokFile.Offset(withNode.Pos()), withTokFile.Offset(withNode.End()))
	printDecls(tokFile, file)
	if _, ok := replaceNode(fset, posStart, posEnd, file, withNode); !ok {
		return false, nil, nil, errors.PrinterError("didn't find node to replace")
	}

	except := map[ast.Node]bool{withNode: true}

	FixPositionsExcept(withNode.Pos(), mod, file, true, except)
	printDecls(tokFile, file)
	return true, fset, file, nil
}

func AddDecl(fset *token.FileSet, filename string, file *ast.File, withFileSet *token.FileSet, withFileName string, withFile *ast.File, withPosStart, withPosEnd token.Position, identMap st.IdentifierMap) (bool, *token.FileSet, *ast.File, *errors.GoRefactorError) {
	withNode := findNode(withFileSet, withFile, withPosStart, withPosEnd)
	if withNode == nil {
		return false, nil, nil, errors.PrinterError("couldn't find node with given positions")
	}
	if _, ok := withNode.(ast.Decl); !ok {
		return false, nil, nil, errors.PrinterError("node is not a declaration")
	}

	withTokFile := getFileFromFileSet(withFileSet, withFileName)
	withOldSize := withTokFile.Size()

	l := int(withNode.End() - withNode.Pos())
	fset, file = reparseFile(file, filename, l, identMap)
	if filename == withFileName {
		withFileSet, withFile = fset, file
		withNode = findNode(withFileSet, withFile, withPosStart, withPosEnd)
	}
	withNode = utils.CopyAstNode(withNode)

	tokFile := getFileFromFileSet(fset, filename)
	withTokFile = getFileFromFileSet(withFileSet, withFileName)
	if tokFile == nil || withTokFile == nil {
		return false, nil, nil, errors.PrinterError("couldn't find file " + filename + " in fileset")
	}

	lines := getLines(tokFile)
	fmt.Printf("linesCount = %d\n", len(lines))
	tokFile.SetLines(lines[:len(lines)-(l)])

	lines = getLines(tokFile)
	for i, offset := range lines {
		fmt.Printf("%d -> %s(%d)\n", i+1, fset.Position(tokFile.Pos(offset)), offset)
	}

	withNodeLines, _ := getRangeLines(withTokFile, withNode.Pos(), withNode.End(), withOldSize)
	withMod := int(tokFile.Offset(file.Decls[len(file.Decls)-1].End()) + 1 - withTokFile.Offset(withNode.Pos()))
	fmt.Printf("withMod: %v\n", withMod)
	for i, _ := range withNodeLines {
		withNodeLines[i] += withMod
	}

	tokFile.SetLines(addLinesOfRange(withNode.Pos(), withNode.End(), lines, withNodeLines, -1)) //to the end
	file.Decls = append(file.Decls, withNode.(ast.Decl))

	return true, fset, file, nil
}

func AddDeclExplicit(fset *token.FileSet, filename string, file *ast.File, withFileSet *token.FileSet, withFileName string, withFile *ast.File, withNode ast.Node, identMap st.IdentifierMap) (bool, *token.FileSet, *ast.File, *errors.GoRefactorError) {

	if _, ok := withNode.(ast.Decl); !ok {
		return false, nil, nil, errors.PrinterError("node is not a declaration")
	}

	withTokFile := getFileFromFileSet(withFileSet, withFileName)
	withOldSize := withTokFile.Size()

	l := int(withNode.End() - withNode.Pos())
	println("111")
	fset, file = reparseFile(file, filename, l, identMap)
	if filename == withFileName {
		FixPositions(0, -withTokFile.Base(), withNode, true)
		withFileSet, withFile = fset, file
	}

	tokFile := getFileFromFileSet(fset, filename)
	withTokFile = getFileFromFileSet(withFileSet, withFileName)
	if tokFile == nil || withTokFile == nil {
		return false, nil, nil, errors.PrinterError("couldn't find file " + filename + " in fileset")
	}

	lines := getLines(tokFile)
	fmt.Printf("linesCount = %d\n", len(lines))
	tokFile.SetLines(lines[:len(lines)-(l)])

	lines = getLines(tokFile)
	for i, offset := range lines {
		fmt.Printf("%d -> %s(%d)\n", i+1, fset.Position(tokFile.Pos(offset)), offset)
	}

	withNodeLines, _ := getRangeLines(withTokFile, withNode.Pos(), withNode.End(), withOldSize)
	withMod := int(tokFile.Offset(file.Decls[len(file.Decls)-1].End()) + 1 - withTokFile.Offset(withNode.Pos()))
	fmt.Printf("withMod: %v\n", withMod)
	for i, _ := range withNodeLines {
		withNodeLines[i] += withMod
	}

	tokFile.SetLines(addLinesOfRange(withNode.Pos(), withNode.End(), lines, withNodeLines, -1)) //to the end
	file.Decls = append(file.Decls, withNode.(ast.Decl))

	return true, fset, file, nil
}
func RenameIdents(fset *token.FileSet, identMap st.IdentifierMap, filename string, file *ast.File, positions []token.Position, name string) (bool, *token.FileSet, *ast.File, *errors.GoRefactorError) {
	l := len(positions)
	id, ok := FindIdentByPos(fset, file, positions[0])
	if !ok {
		return false, nil, nil, errors.PrinterError("couldn't find ident at " + positions[0].String())
	}
	delta := len(name) - len(id.Name)
	if delta > 0 {
		fset, file = reparseFile(file, filename, delta*l, identMap)
	}
	for _, p := range positions {
		if p.Filename != filename {
			return false, nil, nil, errors.PrinterError("positions array contain position with wrong filename")
		}
		if id, ok := FindIdentByPos(fset, file, p); ok {
			id.Name = name
		} else {
			return false, nil, nil, errors.PrinterError("couldn't find ident at " + p.String())
		}
	}
	return true, fset, file, nil
}

func AddLineForRange(fset *token.FileSet, filename string, Pos, End token.Pos) {
	tokFile := getFileFromFileSet(fset, filename)
	lines := getLines(tokFile)
	for i, offset := range lines {
		fmt.Printf("%d -> %s(%d)\n", i+1, fset.Position(tokFile.Pos(offset)), offset)
	}
	var li int
	for i, l := range lines {
		if l > tokFile.Offset(Pos) {
			li = i
			break
		}
	}
	mod := int(End-Pos) + tokFile.Offset(Pos) - lines[li-1]
	fmt.Printf("node --------- %d %d\n", tokFile.Offset(Pos), tokFile.Offset(End))
	//mod := int(End - Pos)
	fmt.Printf("mod = %d\n", mod)
	newLines := make([]int, len(lines)+1)
	copy(newLines, lines[0:li-1])
	newLines[li-1] = tokFile.Offset(Pos) - (tokFile.Offset(Pos) - lines[li-1])
	//newLines[li-1] = tokFile.Offset(Pos)
	for i := li - 1; i < len(lines); i++ {
		newLines[i+1] = lines[i] + mod
	}
	tokFile.SetLines(newLines)
	for i, offset := range newLines {
		fmt.Printf("%d -> %s(%d)\n", i+1, fset.Position(tokFile.Pos(offset)), offset)
	}
}
