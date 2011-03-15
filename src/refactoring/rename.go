package refactoring

import (
	"refactoring/st"
	"refactoring/utils"
	"refactoring/errors"
	"refactoring/program"
	"go/ast"
)

type findImportVisitor struct {
	Package *st.Package
	ps      *st.PackageSymbol
	result  *ast.ImportSpec
}

func (v *findImportVisitor) found() bool {
	return v.result != nil
}

func (v *findImportVisitor) find(i *ast.ImportSpec) bool {
	if ("\"" + v.ps.ShortPath + "\"") == string(i.Path.Value) {
		v.result = i
		return true
	}
	return false
}

func (v *findImportVisitor) Visit(node ast.Node) ast.Visitor {
	if v.found() {
		return nil
	}
	if is, ok := node.(*ast.ImportSpec); ok {
		v.find(is)
		return nil
	}
	return v
}

func findImportDecl(pack *st.Package, file *ast.File, ps *st.PackageSymbol) *ast.ImportSpec {
	v := &findImportVisitor{pack, ps, nil}
	ast.Walk(v, file)
	return v.result
}

func CheckRenameParameters(filename string, line int, column int, newName string) (bool, *errors.GoRefactorError) {
	switch {
	case filename == "" || !utils.IsGoFile(filename):
		return false, errors.ArgumentError("filename", "It's not a valid go file name")
	case line < 1:
		return false, errors.ArgumentError("line", "Must be > 1")
	case column < 1:
		return false, errors.ArgumentError("column", "Must be > 1")
	case !IsGoIdent(newName):
		return false, errors.ArgumentError("newName", "It's not a valid go identifier")
	}
	return true, nil
}
func Rename(programTree *program.Program, filename string, line int, column int, newName string) (bool, int, *errors.GoRefactorError) {

	if ok, err := CheckRenameParameters(filename, line, column, newName); !ok {
		return false, 0, err
	}

	var count int
	if sym, err := programTree.FindSymbolByPosition(filename, line, column); err == nil {

		if _, ok := sym.(*st.PointerTypeSymbol); ok {
			panic("find by position returned pointer type!!!")
		}
		if st.IsPredeclaredIdentifier(sym.Name()) {
			return false, 0, errors.UnrenamableIdentifierError(sym.Name(), " It's a basic language symbol")
		}
		if sym.PackageFrom().IsGoPackage {
			return false, 0, errors.UnrenamableIdentifierError(sym.Name(), " It's a symbol,imported from go library")
		}

		if _, ok := sym.Scope().LookUp(newName, filename); ok {
			return false, 0, errors.IdentifierAlreadyExistsError(newName)
		}

		if meth, ok := sym.(*st.FunctionSymbol); ok {
			if meth.IsInterfaceMethod {
				return false, 0, errors.UnrenamableIdentifierError(sym.Name(), " It's an interface method")
			}
		}
		if ps, ok := sym.(*st.PackageSymbol); ok {
			pack, file := programTree.FindPackageAndFileByFilename(filename)
			impDecl := findImportDecl(pack, file, ps)
			if impDecl == nil {
				panic("couldn't find import decl")
			}
			if impDecl.Name == nil || impDecl.Name.Name == "" {
				impDecl.Name = ast.NewIdent(newName)
			}
		}
		count = renameSymbol(sym, newName)
	} else {
		return false, 0, err
	}
	return true, count, nil
}

func renameSymbol(sym st.Symbol, newName string) int {
	for ident, _ := range sym.Identifiers() {
		ident.Name = newName
	}
	return len(sym.Positions())
}
