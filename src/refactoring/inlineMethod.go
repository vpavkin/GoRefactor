package refactoring

import (
	"refactoring/program"
	"refactoring/st"
	"refactoring/utils"
	"refactoring/errors"
	"refactoring/printerUtil"
	"go/ast"
	"go/token"
	"strconv"

	"fmt"
	"os"
	"go/printer"
)

//given start and end positions visitor finds call expression
type getCallVisitor struct {
	FileSet *token.FileSet

	callStart token.Position
	callEnd   token.Position

	CallNode ast.Node

	nodeFrom ast.Node
}

func (vis *getCallVisitor) found() bool {
	return vis.CallNode != nil
}

func (vis *getCallVisitor) find(n ast.Node) bool {
	return utils.ComparePosWithinFile(vis.FileSet.Position(n.Pos()), vis.callStart) == 0 &&
		utils.ComparePosWithinFile(vis.FileSet.Position(n.End()), vis.callEnd) == 0
}

func (vis *getCallVisitor) checkStmtList(list []ast.Stmt) bool {

	for _, stmt := range list {
		if vis.find(stmt) {
			vis.CallNode = stmt
			return true
		}
	}
	return false
}
func (vis *getCallVisitor) Visit(node ast.Node) ast.Visitor {
	if vis.found() {
		return nil
	}
	switch t := node.(type) {
	case *ast.BlockStmt:
		if vis.checkStmtList(t.List) {
			vis.nodeFrom = t
			return nil
		}
	case *ast.SwitchStmt:
		for _, cc := range t.Body.List {
			caseClause := cc.(*ast.CaseClause)
			if vis.checkStmtList(caseClause.Body) {
				vis.nodeFrom = caseClause
				return nil
			}
		}
	case *ast.TypeSwitchStmt:
		for _, cc := range t.Body.List {
			caseClause := cc.(*ast.CaseClause)
			if vis.checkStmtList(caseClause.Body) {
				vis.nodeFrom = caseClause
				return nil
			}
		}
	case *ast.SelectStmt:
		for _, cc := range t.Body.List {
			caseClause := cc.(*ast.CommClause)
			if vis.checkStmtList(caseClause.Body) {
				vis.nodeFrom = caseClause
				return nil
			}
		}
	case ast.Stmt, ast.Expr:
		if vis.find(t) {
			vis.CallNode = t
			return nil
		}
	}
	return vis
}

func getCall(fileSet *token.FileSet, file *ast.File, filename string, lineStart int, colStart int, lineEnd int, colEnd int) (ast.Node, ast.Node) {
	vis := &getCallVisitor{fileSet, token.Position{filename, 0, lineStart, colStart}, token.Position{filename, 0, lineEnd, colEnd}, nil, nil}
	ast.Walk(vis, file)
	return vis.CallNode, vis.nodeFrom
}

// support functions
func getCallExpr(callNode ast.Node) (*ast.CallExpr, *errors.GoRefactorError) {
	switch node := callNode.(type) {
	case *ast.ExprStmt:
		if res, ok := node.X.(*ast.CallExpr); ok {
			return res, nil
		}
	case *ast.CallExpr:
		return node, nil
	}
	return nil, &errors.GoRefactorError{ErrorType: "inline method error", Message: "selected statement or expression is not a function call"}
}

func getMethodSymbol(programTree *program.Program, callExpr *ast.CallExpr) (*st.FunctionSymbol, *errors.GoRefactorError) {
	var s st.Symbol
	switch funExpr := callExpr.Fun.(type) {
	case *ast.Ident:
		s = programTree.IdentMap.GetSymbol(funExpr)
	case *ast.SelectorExpr:
		s = programTree.IdentMap.GetSymbol(funExpr.Sel)
	default:
		panic("function expression is not a selector or ident")
	}
	if res, ok := s.(*st.FunctionSymbol); ok {
		return res, nil
	}
	panic("Call expr Function is not a st.FunctionSymbol")
}

func getDeclarationInFile(programTree *program.Program, pack *st.Package, sym *st.FunctionSymbol) (decl *ast.FuncDecl, filename string, err *errors.GoRefactorError) {
	for name, f := range pack.AstPackage.Files {
		for _, d := range f.Decls {
			if fdecl, ok := d.(*ast.FuncDecl); ok {
				if programTree.IdentMap.GetSymbol(fdecl.Name) == sym {
					return fdecl, name, nil
				}
			}
		}
	}
	return nil, "", &errors.GoRefactorError{ErrorType: "inline method error (critical)", Message: "couldn't find method declaration"}
}

func getDestScope(programTree *program.Program, pack *st.Package, nodeFrom ast.Node) map[st.Symbol]bool {
	res := make(map[st.Symbol]bool)
	if nodeFrom == nil { // method just uses it's parameters
		return res
	}
	list := getStmtList(nodeFrom)
	params, declared := getParametersAndDeclaredIn(pack, list, programTree)
	params.ForEach(func(sym st.Symbol) {
		res[sym] = true
	})
	declared.ForEach(func(sym st.Symbol) {
		res[sym] = true
	})
	return res
}

func getRecieverExpr(callExpr *ast.CallExpr) ast.Expr {
	switch t := callExpr.Fun.(type) {
	case *ast.Ident:
		panic("reciever expected")
	case *ast.SelectorExpr:
		return t.X
	case *ast.CallExpr:
		panic("can't inline call with reciever, returned by a method (possible side efects)")
	}
	panic("unknown reciever expression type")
}

func getNewNames(callExpr *ast.CallExpr, funSym *st.FunctionSymbol, destScope map[st.Symbol]bool) map[st.Symbol]ast.Expr {
	newNames := make(map[st.Symbol]ast.Expr)
	bt, _ := st.GetBaseType(funSym.FunctionType)
	ftype := bt.(*st.FunctionTypeSymbol)
	//Reciever
	if ftype.Reciever != nil {
		ftype.Reciever.ForEach(func(sym st.Symbol) {
			newNames[sym] = getRecieverExpr(callExpr)
		})
	}
	//Parameters
	i := 0
	ftype.Parameters.ForEach(func(sym st.Symbol) {
		newNames[sym] = callExpr.Args[i]
		i++
	})
	//Locals
	usedNames := make(map[string]bool)
	for dest, _ := range destScope {
		usedNames[dest.Name()] = true
	}
	funSym.Locals.ForEach(func(sym st.Symbol) {
		if _, ok := usedNames[sym.Name()]; ok {
			j := 0
			for {
				nn := sym.Name() + strconv.Itoa(j)
				if _, ok := usedNames[nn]; !ok {
					newNames[sym] = ast.NewIdent(nn)
					usedNames[nn] = true
					break
				}
			}
		}
	})
	return newNames
}

// convertStatementListVisitor prepares statement list of a method to be inlined in destination scope.
// It changes imported packages' aliases and adds import statements if needed.
// Names of method locals, that conflict with destination scope, are renamed.
type statementListConverter struct {
	IdentMap st.IdentifierMap
	Package  *st.Package
	FileSet  *token.FileSet
	TokFile  *token.File

	newNames     map[st.Symbol]ast.Expr //for method parameters - passed expressions, for locals - their new names
	importsToAdd map[*st.PackageSymbol]bool

	sourceFile string
	destFile   string

	nodeLines []int

	sourceList []ast.Stmt
	destList   []ast.Stmt

	Chan chan ast.Expr
}

func (slc *statementListConverter) source() {
	sv := &sourceVisitor{slc}
	for _, stmt := range slc.sourceList {
		ast.Walk(sv, stmt)
	}
}
func (slc *statementListConverter) destination(allCovered chan int) {
	dv := &destinationVisitor{slc, nil}
	for _, stmt := range slc.destList {
		dv.rootNode = stmt
		ast.Walk(dv, stmt)
	}
	allCovered <- 0
}

type sourceVisitor struct {
	*statementListConverter
}

func (vis *sourceVisitor) Visit(node ast.Node) ast.Visitor {
	switch t := node.(type) {
	case *ast.SelectorExpr:
		ast.Walk(vis, t.X)
		return nil
	case *ast.Ident:
		sym := vis.IdentMap.GetSymbol(t)
		if ps, ok := sym.(*st.PackageSymbol); ok {
			p := ps.Package
			imp := vis.Package.GetImport(vis.destFile, p)
			if imp != nil {
				if imp.Name() != ps.Name() {
					vis.Chan <- &ast.Ident{t.NamePos, imp.Name(), nil}
				} else {
					vis.Chan <- nil
				}
				return nil
			}
			vis.importsToAdd[ps] = true
			vis.Chan <- &ast.Ident{t.NamePos, ps.Name(), nil}
			return nil
		}
		if nn, ok := vis.newNames[sym]; ok {
			vis.Chan <- utils.CopyAstNode(nn).(ast.Expr)
		} else {
			vis.Chan <- nil
		}
		return nil
	}
	return vis
}

type destinationVisitor struct {
	*statementListConverter
	rootNode ast.Node
}

func (vis *destinationVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	switch t := node.(type) {
	case *ast.SelectorExpr:
		ast.Walk(vis, t.X)
		return nil
	case *ast.Ident:
		newExpr := <-vis.Chan
		if newExpr != nil {
			printerUtil.FixPositions(0, int(t.Pos()-newExpr.Pos()), newExpr, true)
			replaceExpr(vis.FileSet.Position(t.Pos()), vis.FileSet.Position(t.End()), newExpr, vis.FileSet, vis.rootNode)
			le, _ := utils.GetNodeLength(newExpr)
			mod := le - int(t.End()-t.Pos())
			for _, stmt := range vis.destList {
				printerUtil.FixPositionsExcept(t.Pos(), mod, stmt, true, map[ast.Node]bool{newExpr: true})
			}
			for i, _ := range vis.nodeLines {
				if vis.nodeLines[i] > vis.TokFile.Offset(t.Pos()) {
					vis.nodeLines[i] += mod
				}
			}
		}
		return nil
	}
	return vis
}

func getResultStmtList(IdentMap st.IdentifierMap, pack *st.Package, funSym *st.FunctionSymbol, newNames map[st.Symbol]ast.Expr, sourceFile string, destFile string, sourceList []ast.Stmt, fset *token.FileSet, newPos token.Pos) ([]ast.Stmt, []int, map[*st.PackageSymbol]bool) {
	listCopy := utils.CopyStmtList(sourceList)

	tokFile := printerUtil.GetFileFromFileSet(fset, sourceFile)

	nodeLines, _ := getRangeLinesAtLeastOne(tokFile, sourceList[0].Pos(), sourceList[len(sourceList)-1].End(), tokFile.Size())
	converter := &statementListConverter{IdentMap, pack, fset, tokFile, newNames, make(map[*st.PackageSymbol]bool), sourceFile, destFile, nodeLines, sourceList, listCopy, make(chan ast.Expr)}

	allCovered := make(chan int)
	go converter.source()
	go converter.destination(allCovered)
	<-allCovered

	return converter.destList, converter.nodeLines, converter.importsToAdd
}

func makeImportDecl(impPos token.Pos, path string) *ast.GenDecl {
	return &ast.GenDecl{nil, impPos, token.IMPORT, token.NoPos, []ast.Spec{&ast.ImportSpec{nil, nil, &ast.BasicLit{impPos + token.Pos(len("import ")), token.STRING, "\"" + path + "\""}, nil}}, token.NoPos}
}

func printDecls(tf *token.File, f *ast.File) {
	for _, d := range f.Decls {
		fmt.Printf("> %d %d\n", tf.Offset(d.Pos()), tf.Offset(d.End()))
	}
}

func getRangeLinesAtLeastOne(f *token.File, Pos, End token.Pos, fileSize int) (lines []int, firstLineNum int) {
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
	if (int(End) == fileSize+f.Base()-1) || f.Line(End+1) > l {
		lines = append(lines, f.Offset(End+1))
		if firstLineNum == -1 {
			firstLineNum = f.Line(End + 1)
		}
	}
	if firstLineNum < 0 {
		for p := End; ; p++ {
			if f.Line(p) > l {
				firstLineNum = l
				lines = append(lines, f.Offset(p))
				break
			}
		}
	}
	return
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

func InlineMethod(filename string, lineStart int, colStart int, lineEnd int, colEnd int) (bool, *errors.GoRefactorError) {
	p := parseProgram(filename)
	ok, err := inlineMethod(p, filename, lineStart, colStart, lineEnd, colEnd)
	if !ok {
		p.SaveFile(filename)
	}
	return ok, err
}

func inlineMethod(programTree *program.Program, filename string, lineStart int, colStart int, lineEnd int, colEnd int) (bool, *errors.GoRefactorError) {
	if ok, err := CheckInlineMethodParameters(filename, lineStart, colStart, lineEnd, colEnd); !ok {
		return false, err
	}
	pack, file := programTree.FindPackageAndFileByFilename(filename)
	if pack == nil {
		return false, errors.ArgumentError("filename", "Program packages don't contain file '"+filename+"'")
	}
	fset := pack.FileSet
	callNode, nodeFrom := getCall(fset, file, filename, lineStart, colStart, lineEnd, colEnd)
	if callNode == nil {
		return false, &errors.GoRefactorError{ErrorType: "inline method error", Message: "couldn't find call expression"}
	}
	_, CallAsExpression := callNode.(*ast.CallExpr)
	callExpr, err := getCallExpr(callNode)
	if err != nil {
		return false, err
	}

	funSym, err := getMethodSymbol(programTree, callExpr)
	if err != nil {
		return false, err
	}
	if funSym.PackageFrom() != pack {
		return false, &errors.GoRefactorError{ErrorType: "inline method error", Message: "can't inline method from other package"}
	}
	decl, sourceFile, err := getDeclarationInFile(programTree, pack, funSym)
	if err != nil {
		return false, err
	}

	if len(decl.Body.List) == 0 {
		ok, err := printerUtil.DeleteNode(fset, filename, file, fset.Position(callExpr.Pos()), fset.Position(callExpr.End()))
		if !ok {
			return false, err
		}
		programTree.SaveFileExplicit(filename, fset, file)
		return true, nil
	}

	destScope := getDestScope(programTree, pack, nodeFrom)
	newNames := getNewNames(callExpr, funSym, destScope)

	for sym, expr := range newNames {
		fmt.Printf("%s -> ", sym.Name())
		printer.Fprint(os.Stdout, token.NewFileSet(), expr)
		print("; ")
	}

	tokFile := printerUtil.GetFileFromFileSet(fset, filename)
	sourceTokFile := printerUtil.GetFileFromFileSet(fset, sourceFile)

	lines := printerUtil.GetLines(tokFile)
	for i, offset := range lines {
		fmt.Printf("%d -> %s(%d)\n", i+1, fset.Position(tokFile.Pos(offset)), offset)
	}

	oldListLines, fline := getRangeLinesAtLeastOne(sourceTokFile, decl.Body.List[0].Pos(), decl.Body.List[len(decl.Body.List)-1].End(), sourceTokFile.Size())
	fmt.Printf("fline = %d\n", fline)

	fmt.Printf("CONVERTER before: %v\n", oldListLines)
	fmt.Printf("CONVERTER pos,end: %d,%d\n", decl.Body.List[0].Pos(), decl.Body.List[len(decl.Body.List)-1].End())

	resList, newListLines, importsToAdd := getResultStmtList(programTree.IdentMap, pack, funSym, newNames, sourceFile, filename, decl.Body.List, pack.FileSet, callExpr.Pos())

	sourceLines := printerUtil.GetLines(sourceTokFile)
	zeroSourceLine := newListLines[0] - sourceLines[fline-2]
	fmt.Printf("zeroSourceLine = %d\n", zeroSourceLine)

	fmt.Printf("CONVERTER after: %v\n", newListLines)
	fmt.Printf("CONVERTER pos,end: %d,%d\n", resList[0].Pos(), resList[len(resList)-1].End())

	callExprLen := int(callExpr.End() - callExpr.Pos())
	listLen := newListLines[len(newListLines)-1] - sourceLines[fline-2]
	importsLen := 0
	for ps, _ := range importsToAdd {
		fmt.Printf("import \"%s\"\n", ps.ShortPath)
		importsLen += len("import \"\"\n") + len(ps.ShortPath)
	}
	mod := listLen + importsLen - callExprLen
	fmt.Printf("mod = %d\n", mod)
	if mod > 0 {
		println("REPARSING *****************************************")
		oldSourceTokFileSize := sourceTokFile.Size()

		fset, file = printerUtil.ReparseFile(file, filename, mod, programTree.IdentMap)
		tokFile = printerUtil.GetFileFromFileSet(fset, filename)
		sourceTokFile = printerUtil.GetFileFromFileSet(pack.FileSet, sourceFile)
		lines = printerUtil.GetLines(tokFile)
		tokFile.SetLines(lines[:len(lines)-(mod)])

		callNode, nodeFrom = getCall(fset, file, filename, lineStart, colStart, lineEnd, colEnd)
		callExpr, _ = getCallExpr(callNode)
		funSym, _ = getMethodSymbol(programTree, callExpr)
		decl, sourceFile, _ = getDeclarationInFile(programTree, pack, funSym)
		destScope = getDestScope(programTree, pack, nodeFrom)
		newNames = getNewNames(callExpr, funSym, destScope)

		lines = printerUtil.GetLines(tokFile)
		for i, offset := range lines {
			fmt.Printf("%d -> %s(%d)\n", i+1, fset.Position(tokFile.Pos(offset)), offset)
		}

		oldListLines, fline = getRangeLinesAtLeastOne(sourceTokFile, decl.Body.List[0].Pos(), decl.Body.List[len(decl.Body.List)-1].End(), oldSourceTokFileSize)
		fmt.Printf("fline = %d\n", fline)

		fmt.Printf("CONVERTER before: %v\n", oldListLines)
		fmt.Printf("CONVERTER pos,end: %d,%d\n", decl.Body.List[0].Pos(), decl.Body.List[len(decl.Body.List)-1].End())

		resList, newListLines, importsToAdd = getResultStmtList(programTree.IdentMap, pack, funSym, newNames, sourceFile, filename, decl.Body.List, pack.FileSet, callExpr.Pos())

		sourceLines := printerUtil.GetLines(sourceTokFile)
		zeroSourceLine = newListLines[0] - sourceLines[fline-2]
		fmt.Printf("lines[fline - 1] = %d, zeroSourceLine = %d\n", sourceLines[fline-2], zeroSourceLine)

		fmt.Printf("CONVERTER after: %v\n", newListLines)
		fmt.Printf("CONVERTER pos,end: %d,%d\n", resList[0].Pos(), resList[len(resList)-1].End())
	}

	impPos := file.Decls[0].Pos()
	nextLineInd := 0
	for i, offs := range lines {
		if offs > tokFile.Offset(impPos) {
			nextLineInd = i
			break
		}
	}

	for ps, _ := range importsToAdd {
		printDecls(tokFile, file)

		file.Decls = append([]ast.Decl{makeImportDecl(impPos, ps.ShortPath)}, file.Decls...)
		mod := len("import \"\"\n") + len(ps.ShortPath)
		//positions
		printerUtil.FixPositionsExcept(impPos-token.Pos(1), mod, file, true, map[ast.Node]bool{file.Decls[0]: true})
		//lines
		lines = printerUtil.GetLines(tokFile)
		fmt.Printf("before import %s (%d): %v\n", ps.ShortPath, impPos, lines)
		newLines := make([]int, 0, len(lines)+1)
		newLines = append(newLines, lines[:nextLineInd]...)
		newLines = append(newLines, tokFile.Offset(impPos))
		newLines = append(newLines, lines[nextLineInd:]...)
		for i := nextLineInd; i < len(newLines); i++ {
			newLines[i] += mod
		}
		fmt.Printf("after import %s (nextLine = %d): %v\n", ps.ShortPath, nextLineInd, newLines)
		if !tokFile.SetLines(newLines) {
			panic("couldn't set lines for file " + tokFile.Name())
		}

		printDecls(tokFile, file)
	}

	resMod := int(callExpr.Pos() - resList[0].Pos())
	for _, stmt := range resList {
		printerUtil.FixPositions(0, resMod, stmt, true)
	}
	lines = printerUtil.GetLines(tokFile)

	resLinesMod := lines[tokFile.Line(callExpr.Pos())-1] - newListLines[0] + zeroSourceLine
	fmt.Printf("resLinesMod: %d\n", resLinesMod)

	for i, _ := range newListLines {
		newListLines[i] += resLinesMod
	}
	fmt.Printf("call pos: %d,%d; list pos: %d,%d\n", callExpr.Pos(), callExpr.End(), resList[0].Pos(), resList[len(resList)-1].End())
	fmt.Printf("newLines %v\n", newListLines)

	lines = printerUtil.GetLines(tokFile)
	for i, offset := range lines {
		fmt.Printf("%d -> %s(%d)\n", i+1, fset.Position(tokFile.Pos(offset)), offset)
	}

	if CallAsExpression {
		rs, ok := resList[0].(*ast.ReturnStmt)
		if !ok {
			return false, &errors.GoRefactorError{ErrorType: "inline method error", Message: "method, inlined as expression, must have only one statement - return statement"}
		}
		switch len(rs.Results) {
		case 0:
			panic("methos, inlined as expression, doesn't return anything")
		default:

			elist := rs.Results
			for _, e := range elist {
				printerUtil.FixPositions(0, -len("return "), e, true)
			}
			mod := int(elist[len(elist)-1].End()-elist[0].Pos()) - callExprLen
			lines = printerUtil.GetLines(tokFile)

			fmt.Printf("before last (mod = %d) : %v\n", mod, lines)
			for i, offset := range lines {
				if offset > tokFile.Offset(callExpr.Pos()) {
					for j := i; j < len(lines); j++ {
						lines[j] += mod
					}
					break
				}
			}
			fmt.Printf("after last (mod = %d) : %v\n", mod, lines)
			fmt.Printf("posits: %s,%s\n", fset.Position(callExpr.Pos()), fset.Position(callExpr.End()))
			if !tokFile.SetLines(lines) {
				panic("couldn't set lines for file " + tokFile.Name())
			}

			printerUtil.FixPositionsExcept(callExpr.Pos(), mod, file, true, map[ast.Node]bool{callExpr: true})

			errs := replaceExprList(fset.Position(callExpr.Pos()), fset.Position(callExpr.End()), elist, fset, file)
			if err, ok := errs[INLINE_METHOD]; ok {
				return false, err
			}
			programTree.SaveFileExplicit(filename, fset, file)
		}

	} else {

		list := getStmtList(nodeFrom)
		i, ok := getIndexOfStmt(callNode.(ast.Stmt), list)
		if !ok {
			panic("couldn't find call statement during inline")
		}

		callExprLine := tokFile.Line(callExpr.Pos()) - 1
		mod := int(resList[len(resList)-1].End()-resList[0].Pos()) - callExprLen

		printerUtil.FixPositions(callExpr.Pos(), mod, file, true)

		lines = printerUtil.GetLines(tokFile)
		fmt.Printf("before last (mod = %d) : %v\n", mod, lines)
		newLines := make([]int, 0, len(lines)+len(newListLines)-1)
		newLines = append(newLines, lines[:callExprLine+1]...)
		newLines = append(newLines, newListLines...)
		newLines = append(newLines, lines[callExprLine+2:]...)
		fmt.Printf("after last (lines[callExprLine] = %d): %v\n", lines[callExprLine], newLines)
		for i := callExprLine + 1 + len(newListLines); i < len(newLines); i++ {
			newLines[i] += mod
		}
		fmt.Printf("after last: %v\n", newLines)
		if !tokFile.SetLines(newLines) {
			panic("couldn't set lines for file " + tokFile.Name())
		}

		if len(resList) == 1 {
			list[i] = resList[0]
			return true, nil
		}
		fmt.Printf("len = %d\n", len(list)-1+len(resList))
		fmt.Printf("%v\n", list)
		newList := make([]ast.Stmt, len(list)-1+len(resList))
		for j := 0; j < i; j++ {
			newList[j] = list[j]
		}
		fmt.Printf("%v\n", newList)
		for j := 0; j < len(resList); j++ {
			newList[j+i] = resList[j]
		}
		fmt.Printf("%v\n", newList)
		for j := i + 1; j < len(list); j++ {
			newList[j+len(resList)-1] = list[j]
		}
		fmt.Printf("%v\n", newList)

		setStmtList(nodeFrom, newList)

		programTree.SaveFileExplicit(filename, fset, file)
	}

	return true, nil
}
