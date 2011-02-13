package refactoring

import (
	"go/ast"
	"st"
	"go/token"
	"utils"
	"errors"
	"program"
	"strconv"
)

type getSelectedArgumentVisitor struct {
	Package *st.Package
	argPos  token.Position
	result  *ast.Field
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
			if utils.ComparePosWithinFile(vis.Package.FileSet.Position(f.Pos()), vis.argPos) == 0 {
				vis.result = f
				return nil
			}
		}
	}
	return nil
}

func getSelectedArgument(file *ast.File, argPos token.Position, pack *st.Package) (*ast.Field, *ast.FuncDecl, bool) {
	vis := &getSelectedArgumentVisitor{pack, argPos, nil}
	for _, d := range file.Decls {
		ast.Walk(vis, d)
		if vis.result != nil {
			return vis.result, d.(*ast.FuncDecl), true
		}
	}
	return nil, nil, false
}

type getUsedMethodsVisitor struct {
	Package  *st.Package
	identMap st.IdentifierMap
	varS     *st.VariableSymbol
	errs     map[*errors.GoRefactorError]bool
	result   map[*st.FunctionSymbol]bool
}

func (vis *getUsedMethodsVisitor) Visit(node ast.Node) (w ast.Visitor) {
	if node == nil {
		return nil
	}
	pos := vis.Package.FileSet.Position(node.Pos())
	switch t := node.(type) {
	case *ast.SelectorExpr:
		x, ok := t.X.(*ast.Ident)
		if !ok {
			return vis
		}
		s := vis.identMap.GetSymbol(x)
		if vis.varS == s {
			m, ok := vis.identMap.GetSymbol(t.Sel).(*st.FunctionSymbol)
			if !ok {

				vis.errs[&errors.GoRefactorError{ErrorType: "extract interface error", Message: "symbol used in selector expression that is not a call. at " + strconv.Itoa(pos.Line) + ":" + strconv.Itoa(pos.Column)}] = true
				return nil
			}
			vis.result[m] = true
			return nil
		}
		if vis.varS.VariableType == s {
			m, ok := vis.identMap.GetSymbol(t.Sel).(*st.FunctionSymbol)
			if !ok {
				panic("couldn't find method selector in method expression")
			}
			vis.result[m] = true
			return nil
		}

	case *ast.BinaryExpr:
		x, okX := t.X.(*ast.Ident)
		y, okY := t.Y.(*ast.Ident)
		if !okX || !okY {
			return vis
		}
		if (x.Name == "nil" && vis.identMap.GetSymbol(y) == vis.varS) ||
			(y.Name == "nil" && vis.identMap.GetSymbol(x) == vis.varS) {
			return nil
		}
	case *ast.Ident:
		if vis.identMap.GetSymbol(t) == vis.varS {
			vis.errs[&errors.GoRefactorError{ErrorType: "extract interface error", Message: "symbol is used as standalone identifier, changing type can cause errors. at " + strconv.Itoa(pos.Line) + ":" + strconv.Itoa(pos.Column)}] = true
			return nil
		}
	}
	return vis
}
func getUsedMethods(programTree *program.Program, pack *st.Package, fdecl *ast.FuncDecl, varS *st.VariableSymbol) (map[*st.FunctionSymbol]bool, map[*errors.GoRefactorError]bool) {
	vis := &getUsedMethodsVisitor{pack, programTree.IdentMap, varS, make(map[*errors.GoRefactorError]bool), make(map[*st.FunctionSymbol]bool)}
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

	if _, ok := pack.Symbols.LookUp(interfaceName, filename); ok {
		return false, errors.IdentifierAlreadyExistsError(interfaceName)
	}

	field, fdecl, ok := getSelectedArgument(file, token.Position{filename, 0, line, column}, pack)
	if !ok {
		return false, &errors.GoRefactorError{ErrorType: "extract interface error", Message: "couldn't find function argument to extract interface"}
	}
	if field.Names == nil {
		return false, &errors.GoRefactorError{ErrorType: "extract interface error", Message: "argument has no name, so empty interface"}
	}
	if len(field.Names) > 1 {
		return false, &errors.GoRefactorError{ErrorType: "extract interface error", Message: "field has two arguments, unsupported case"}
	}

	varS, ok := programTree.IdentMap.GetSymbol(field.Names[0]).(*st.VariableSymbol)
	if !ok {
		panic("symbol supposed to be a variable, but it's not")
	}
	meths, errs := getUsedMethods(programTree, pack, fdecl, varS)
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
	file.Decls = append(file.Decls, genDecl)
	field.Type = ast.NewIdent(interfaceName)

	return true, nil

}
