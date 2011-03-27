package refactoring

import (
	"go/ast"
	"refactoring/st"
	"go/token"
	"refactoring/utils"
	"refactoring/errors"
	"refactoring/program"
	"refactoring/printerUtil"
	"strconv"

	"fmt"
)

type getSelectedArgumentVisitor struct {
	FileSet    *token.FileSet
	argPos     token.Position
	result     *ast.Field
	nameNumber int
}

// only Decls as args (doesn't go in depth)
func (vis *getSelectedArgumentVisitor) Visit(node ast.Node) (w ast.Visitor) {
	switch t := node.(type) {
	case *ast.FuncDecl:
		if t.Type == nil {
			return nil
		}
		if t.Type.Params == nil {
			return nil
		}
		for _, f := range t.Type.Params.List {
			for i, n := range f.Names {
				if utils.ComparePosWithinFile(vis.FileSet.Position(n.Pos()), vis.argPos) == 0 {
					vis.result = f
					vis.nameNumber = i
					return nil
				}
			}
			if utils.ComparePosWithinFile(vis.FileSet.Position(f.Pos()), vis.argPos) == 0 {
				vis.result = f
				vis.nameNumber = -1
				return nil
			}
		}
	}
	return nil
}

func getSelectedArgument(file *ast.File, argPos token.Position, fileSet *token.FileSet) (*ast.Field, int, *ast.FuncDecl, bool) {
	vis := &getSelectedArgumentVisitor{fileSet, argPos, nil, -1}
	for _, d := range file.Decls {
		ast.Walk(vis, d)
		if vis.result != nil {
			return vis.result, vis.nameNumber, d.(*ast.FuncDecl), true
		}
	}
	return nil, 0, nil, false
}

type getUsedMethodsVisitor struct {
	FileSet  *token.FileSet
	identMap st.IdentifierMap
	varS     *st.VariableSymbol
	errs     map[*errors.GoRefactorError]bool
	result   map[*st.FunctionSymbol]bool
}

func (vis *getUsedMethodsVisitor) isDesiredMethodExpression(node ast.Node) bool {
	n := node
	stars := 0
	for {
		switch nt := n.(type) {
		case *ast.Ident:
			s := vis.identMap.GetSymbol(nt)
			switch vtt := vis.varS.VariableType.(type) {
			case *st.PointerTypeSymbol:
				return stars == vtt.Depth() && vtt.BaseType == s
			default:
				return stars == 0 && vtt == s
			}
		case *ast.ParenExpr:
			n = nt.X
		case *ast.StarExpr:
			stars++
			n = nt.X
		case *ast.SelectorExpr:
			n = nt.Sel
		default:
			return false
		}
	}
	return false
}

func (vis *getUsedMethodsVisitor) checkEditME(callExpr *ast.CallExpr, mId *ast.Ident) {
	m, ok := vis.identMap.GetSymbol(mId).(*st.FunctionSymbol)
	if !ok {
		panic("couldn't find method selector in method expression")
	}
	if id, ok := callExpr.Args[0].(*ast.Ident); ok {
		if vis.identMap.GetSymbol(id) == vis.varS {
			callExpr.Fun = &ast.SelectorExpr{id, mId}
			callExpr.Args = callExpr.Args[1:]
			vis.result[m] = true
		}
	}
}

func (vis *getUsedMethodsVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if node == nil {
		return nil
	}
	pos := vis.FileSet.Position(node.Pos())
	switch tt := node.(type) {
	case *ast.CallExpr:

		if t, ok := tt.Fun.(*ast.SelectorExpr); ok {

			switch x := t.X.(type) {
			case *ast.Ident:
				s := vis.identMap.GetSymbol(x)

				if vis.varS == s {
					for _, a := range tt.Args {
						ast.Walk(vis, a)
					}
					m, ok := vis.identMap.GetSymbol(t.Sel).(*st.FunctionSymbol)
					if !ok {

						vis.errs[&errors.GoRefactorError{ErrorType: "extract interface error", Message: "symbol used in selector expression that is not a call. at " + strconv.Itoa(pos.Line) + ":" + strconv.Itoa(pos.Column)}] = true
						return nil
					}
					vis.result[m] = true
					return nil
				}
				if vis.isDesiredMethodExpression(x) {
					for i, a := range tt.Args {
						if i > 0 {
							ast.Walk(vis, a)
						}
					}
					vis.checkEditME(tt, t.Sel)
					return nil
				}
			default:
				if vis.isDesiredMethodExpression(node) {
					for i, a := range tt.Args {
						if i > 0 {
							ast.Walk(vis, a)
						}
					}
					vis.checkEditME(tt, t.Sel)
					return nil
				}
			}
		}
		return nil
	case *ast.SelectorExpr:

		switch x := tt.X.(type) {
		case *ast.Ident:
			s := vis.identMap.GetSymbol(x)

			if vis.varS == s {
				_, ok := vis.identMap.GetSymbol(tt.Sel).(*st.FunctionSymbol)
				if !ok {

					vis.errs[&errors.GoRefactorError{ErrorType: "extract interface error", Message: "symbol used in selector expression that is not a call. at " + strconv.Itoa(pos.Line) + ":" + strconv.Itoa(pos.Column)}] = true
					return nil
				}
			}
		}

	case *ast.BinaryExpr:
		x, okX := tt.X.(*ast.Ident)
		y, okY := tt.Y.(*ast.Ident)
		if !okX || !okY {
			return vis
		}
		if (x.Name == "nil" && vis.identMap.GetSymbol(y) == vis.varS) ||
			(y.Name == "nil" && vis.identMap.GetSymbol(x) == vis.varS) {
			return nil
		}
	case *ast.Ident:
		if vis.identMap.GetSymbol(tt) == vis.varS {
			vis.errs[&errors.GoRefactorError{ErrorType: "extract interface error", Message: "symbol is used as standalone identifier, changing type can cause errors. at " + strconv.Itoa(pos.Line) + ":" + strconv.Itoa(pos.Column)}] = true
			return nil
		}
	}
	return vis
}
func getUsedMethods(programTree *program.Program, fset *token.FileSet, fdecl *ast.FuncDecl, varS *st.VariableSymbol) (map[*st.FunctionSymbol]bool, map[*errors.GoRefactorError]bool) {
	vis := &getUsedMethodsVisitor{fset, programTree.IdentMap, varS, make(map[*errors.GoRefactorError]bool), make(map[*st.FunctionSymbol]bool)}
	ast.Walk(vis, fdecl.Body)
	return vis.result, vis.errs
}

func CheckExtractInterfaceParameters(filename string, line int, column int, interfaceName string) (bool, *errors.GoRefactorError) {
	switch {
	case filename == "" || !utils.IsGoFile(filename):
		return false, errors.ArgumentError("filename", "It's not a valid go file name")
	case line < 1:
		return false, errors.ArgumentError("line", "Must be > 1")
	case column < 1:
		return false, errors.ArgumentError("column", "Must be > 1")
	case !IsGoIdent(interfaceName):
		return false, errors.ArgumentError("interfaceName", "It's not a valid go identifier")
	}
	return true, nil
}

func ExtractInterface(programTree *program.Program, filename string, line int, column int, interfaceName string) (bool, *errors.GoRefactorError) {

	if ok, err := CheckExtractInterfaceParameters(filename, line, column, interfaceName); !ok {
		return false, err
	}

	pack, file := programTree.FindPackageAndFileByFilename(filename)
	if pack == nil {
		return false, errors.ArgumentError("filename", "Program packages don't contain file '"+filename+"'")
	}
	fset := pack.FileSet
	mod := 20000
	fset, file = printerUtil.ReparseFile(file, filename, mod, programTree.IdentMap)
	tokFile := printerUtil.GetFileFromFileSet(fset, filename)
	lines := printerUtil.GetLines(tokFile)
	tokFile.SetLines(lines[:len(lines)-(mod)])

	if _, ok := pack.Symbols.LookUp(interfaceName, filename); ok {
		return false, errors.IdentifierAlreadyExistsError(interfaceName)
	}

	field, nameNum, fdecl, ok := getSelectedArgument(file, token.Position{filename, 0, line, column}, fset)
	if !ok {
		return false, &errors.GoRefactorError{ErrorType: "extract interface error", Message: "couldn't find function argument to extract interface"}
	}
	if field.Names == nil {
		return false, &errors.GoRefactorError{ErrorType: "extract interface error", Message: "argument has no name, so empty interface"}
	}

	varS, ok := programTree.IdentMap.GetSymbol(field.Names[nameNum]).(*st.VariableSymbol)
	if !ok {
		panic("symbol supposed to be a variable, but it's not")
	}
	meths, errs := getUsedMethods(programTree, fset, fdecl, varS)
	if len(errs) > 0 {
		println("some errors don't allow to extract interface:")
		for e, _ := range errs {
			println(e.String())
		}
		return false, &errors.GoRefactorError{ErrorType: "extract interface error", Message: "some errors occured, stopped extracting"}
	}
	methList, i := make([]*ast.Field, len(meths)), 0

	for m, _ := range meths {
		methList[i] = &ast.Field{nil, []*ast.Ident{ast.NewIdent(m.Name())}, m.FunctionType.ToAstExpr(pack, filename), nil, nil}
		i++
	}
	interfaceMethods := &ast.FieldList{token.NoPos, methList, token.NoPos}
	interfaceType := &ast.InterfaceType{token.NoPos, interfaceMethods, false}
	interfaceDecl := &ast.TypeSpec{nil, ast.NewIdent(interfaceName), interfaceType, nil}
	genDecl := &ast.GenDecl{nil, token.NoPos, token.TYPE, token.NoPos, []ast.Spec{interfaceDecl}, token.NoPos}

	fieldNum := 0
	for i := 0; i < len(fdecl.Type.Params.List); i++ {
		if field == fdecl.Type.Params.List[i] {
			fieldNum = i
			break
		}
	}

	oldFTypeLen := int(fdecl.Type.End() - fdecl.Type.Pos())

	var lField, mField, rField *ast.Field
	mField = &ast.Field{nil, []*ast.Ident{field.Names[nameNum]}, ast.NewIdent(interfaceName), nil, nil}
	if nameNum > 0 {
		lField = &ast.Field{nil, field.Names[:nameNum], field.Type, nil, nil}
	}
	if nameNum < len(field.Names)-1 {
		rField = &ast.Field{nil, field.Names[(nameNum + 1):], field.Type, nil, nil}
	}
	newList := make([]*ast.Field, fieldNum)
	copy(newList, fdecl.Type.Params.List[:fieldNum])

	if lField != nil {
		newList = append(newList, lField)
	}
	newList = append(newList, mField)
	if rField != nil {
		newList = append(newList, rField)
	}
	newList = append(newList, fdecl.Type.Params.List[(fieldNum+1):]...)
	fdecl.Type.Params.List = newList

	newFTypeLen, _ := utils.GetNodeLength(fdecl.Type)
	newFTypeLen += len(fdecl.Name.Name) + 1

	fmt.Printf("old %d new %d\n", oldFTypeLen, newFTypeLen)
	allMod := newFTypeLen - oldFTypeLen
	if allMod != 0 {
		printerUtil.FixPositions(fdecl.Type.End(), allMod, file, true)

		lines = printerUtil.GetLines(tokFile)
		fmt.Printf("before last (mod = %d) : %v\n", allMod, lines)
		for i, offset := range lines {
			if offset > tokFile.Offset(fdecl.Type.Pos()) {
				for j := i; j < len(lines); j++ {
					lines[j] += allMod
				}
				break
			}
		}
		fmt.Printf("after last (mod = %d) : %v\n", allMod, lines)
		if !tokFile.SetLines(lines) {
			println("FUUUUUUUUUWUWUWUWUWUWU")
		}
	}

	file.Decls = append(file.Decls, genDecl)
	programTree.SaveFileExplicit(filename, fset, file)
	return true, nil

}
