package refactoring

import (
	"refactoring/st"
	"refactoring/utils"
	"refactoring/errors"
	"refactoring/program"
	"refactoring/printerUtil"
	"go/ast"
	"go/token"
	"unicode"
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
func Rename(programTree *program.Program, filename string, line int, column int, newName string) (ok bool, fnames []string, fsets []*token.FileSet, files []*ast.File, err *errors.GoRefactorError) {

	if ok, err = CheckRenameParameters(filename, line, column, newName); !ok {
		return
	}
	var sym st.Symbol
	if sym, err = programTree.FindSymbolByPosition(filename, line, column); err == nil {

		if _, ok := sym.(*st.PointerTypeSymbol); ok {
			panic("find by position returned pointer type!!!")
		}
		if st.IsPredeclaredIdentifier(sym.Name()) {
			return false, nil, nil, nil, errors.UnrenamableIdentifierError(sym.Name(), " It's a basic language symbol")
		}
		if sym.PackageFrom().IsGoPackage {
			return false, nil, nil, nil, errors.UnrenamableIdentifierError(sym.Name(), " It's a symbol,imported from go library")
		}
		if unicode.IsUpper(int(sym.Name()[0])) && !unicode.IsUpper(int(newName[0])) ||
			unicode.IsLower(int(sym.Name()[0])) && !unicode.IsLower(int(newName[0])) {
			return false, nil, nil, nil, &errors.GoRefactorError{ErrorType: "Can't rename identifier", Message: "can't change access modifier. Changing first letter's case can cause errors."}
		}
		if _, ok := sym.Scope().LookUp(newName, filename); ok {
			return false, nil, nil, nil, errors.IdentifierAlreadyExistsError(newName)
		}

		if meth, ok := sym.(*st.FunctionSymbol); ok {
			if meth.IsInterfaceMethod {
				return false, nil, nil, nil, errors.UnrenamableIdentifierError(sym.Name(), " It's an interface method")
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
		fnames, fsets, files, err = renameSymbol(sym, newName, programTree)
		return err == nil, fnames, fsets, files, err
	} else {
		return false, nil, nil, nil, err
	}
	panic("unreachable code")
}

func renameSymbol(sym st.Symbol, newName string, programTree *program.Program) (fnames []string, fsets []*token.FileSet, files []*ast.File, err *errors.GoRefactorError) {
	filesMap := make(map[string]bool)
	for _, pos := range sym.Positions() {
		filesMap[pos.Filename] = true
	}
	l, i := len(filesMap), 0
	fnames, fsets, files = make([]string, l), make([]*token.FileSet, l), make([]*ast.File, l)
	for f, _ := range filesMap {
		fnames[i] = f
		i++
	}

	for i, f := range fnames {
		pack, file := programTree.FindPackageAndFileByFilename(f)
		positions := []token.Position{}
		for _, pos := range sym.Positions() {
			if pos.Filename == f {
				positions = append(positions, pos)
			}
		}
		if _, fsets[i], files[i], err = printerUtil.RenameIdents(pack.FileSet, programTree.IdentMap, f, file, positions, newName); err != nil {
			return nil, nil, nil, err
		}
	}
	for ident, _ := range sym.Identifiers() {
		ident.Name = newName
	}
	return fnames, fsets, files, nil
}
