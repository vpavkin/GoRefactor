package refactoring

import (
	"refactoring/program"
	"refactoring/st"
	"refactoring/utils"
	"refactoring/errors"
	"go/ast"
	"go/token"
	"go/printer"
	"os"
)

func CheckImplementInterfaceParameters(filename string, line int, column int, varFile string, varLine int, varColumn int) (bool, *errors.GoRefactorError) {
	switch {
	case filename == "" || !utils.IsGoFile(filename):
		return false, errors.ArgumentError("filename", "It's not a valid go file name")
	case varFile == "" || !utils.IsGoFile(varFile):
		return false, errors.ArgumentError("varFile", "It's not a valid go file name")
	case line < 1:
		return false, errors.ArgumentError("line", "Must be > 1")
	case varLine < 1:
		return false, errors.ArgumentError("varLine", "Must be > 1")
	case column < 1:
		return false, errors.ArgumentError("column", "Must be > 1")
	case varColumn < 1:
		return false, errors.ArgumentError("varColumn", "Must be > 1")
	}
	return true, nil
}
func containsMethod(m *st.FunctionSymbol, sym st.ITypeSymbol) (bool, *errors.GoRefactorError) {
	candidate, ok := sym.Methods().LookUp(m.Name(), "")
	if !ok {
		return false, nil
	}
	cand, ok := candidate.(*st.FunctionSymbol)
	if !ok {
		panic("non-method symbol " + candidate.Name() + " in type " + sym.Name() + " methods()")
	}

	if !st.EqualsMethods(cand, m) {
		return true, &errors.GoRefactorError{ErrorType: "implement interface error", Message: "type has a method, named " + cand.Name() + "but it has signature, different from the necessary one"}
	}

	return true, nil
}

func ImplementInterface(filename string, line int, column int, varFile string, varLine int, varColumn int, asPointer bool) (bool, *errors.GoRefactorError) {
	p := parseProgram(filename)
	ok, err := implementInterface(p, filename, line, column, varFile, varLine, varColumn, asPointer)
	p.SaveFile(varFile)
	return ok, err
}

func implementInterface(programTree *program.Program, filename string, line int, column int, varFile string, varLine int, varColumn int, asPointer bool) (bool, *errors.GoRefactorError) {
	if ok, err := CheckImplementInterfaceParameters(filename, line, column, varFile, varLine, varColumn); !ok {
		return false, err
	}
	packInt, _ := programTree.FindPackageAndFileByFilename(filename)
	if packInt == nil {
		return false, errors.ArgumentError("filename", "Program packages don't contain file '"+filename+"'")
	}
	packType, fileType := programTree.FindPackageAndFileByFilename(varFile)
	if packType == nil {
		return false, errors.ArgumentError("filename", "Program packages don't contain file '"+varFile+"'")
	}
	symInt, err := programTree.FindSymbolByPosition(filename, line, column)
	if err != nil {
		return false, err
	}
	symType, err := programTree.FindSymbolByPosition(varFile, varLine, varColumn)
	if err != nil {
		return false, err
	}

	sI, ok := symInt.(*st.InterfaceTypeSymbol)
	if !ok {
		return false, &errors.GoRefactorError{ErrorType: "implement interface error", Message: "symbol, pointed as interface, is actually not a one"}
	}
	if _, ok := symType.(*st.InterfaceTypeSymbol); ok {
		return false, &errors.GoRefactorError{ErrorType: "implement interface error", Message: "interface can't implement methods"}
	}
	sT, ok := symType.(st.ITypeSymbol)
	if !ok {
		return false, &errors.GoRefactorError{ErrorType: "implement interface error", Message: "interface can be implemented only by a type"}
	}
	if asPointer {
		sT = programTree.GetPointerType(sT)
	}

	errors := make([]*errors.GoRefactorError, 0, 10)
	missedMethods := make(map[*st.FunctionSymbol]bool)
	checker := func(sym st.Symbol) {
		f := sym.(*st.FunctionSymbol)
		ok, err := containsMethod(f, sT)
		if err != nil {
			errors = append(errors, err)
		} else if !ok {
			missedMethods[f] = true
		}
	}
	sI.Methods().ForEachNoLock(checker)
	sI.Methods().ForEachOpenedScope(func(table *st.SymbolTable) {
		table.ForEachNoLock(checker)
	})

	println("missedMethods:")
	for s, _ := range missedMethods {
		list := make([]ast.Stmt, 1)
		list[0] = &ast.ExprStmt{&ast.CallExpr{ast.NewIdent("panic"), token.NoPos, []ast.Expr{&ast.BasicLit{token.NoPos, token.STRING, []byte("\"not implemented yet\"")}}, token.NoPos, token.NoPos}}
		fdecl := makeFuncDecl(s.Name(), list, s.FunctionType.(*st.FunctionTypeSymbol).Parameters, nil, s.FunctionType.(*st.FunctionTypeSymbol).Results, st.MakeVariable(st.NO_NAME, sT.Scope(), sT), packType, varFile)
		printer.Fprint(os.Stdout, token.NewFileSet(), fdecl)
		println()
		fileType.Decls = append(fileType.Decls, fdecl)
	}
	println()
	println("errors:")
	for _, err := range errors {
		println(err.String())
	}

	return true, nil
}
