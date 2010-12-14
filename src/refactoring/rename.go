package refactoring

import (
	"unicode"
	"st"
	"utils"
	"errors"
	"program"
	"fmt"
	//"bufio"
	//"os"
)

func isGoIdent(name string) bool {

	if name == "_" || name == "nil" || name == "true" || name == "false" {
		return false
	}
	if st.IsPredeclaredIdentifier(name) {
		return false
	}

	if !(unicode.IsLetter(int(name[0])) || name[0] == '_') {
		return false
	}
	for i := 1; i < len(name); i++ {
		if !(unicode.IsLetter(int(name[i])) || unicode.IsDigit(int(name[i])) || name[0] == '_') {
			return false
		}
	}
	return true
}

// func Compile(directory string, pack *ast.Package) (bool, os.Error) {
// 
// 	goComp, err := exec.LookPath("8g")
// 	if err != nil {
// 		return false, err
// 	}
// 	v := make([]string, len(pack.Files)+1)
// 	v[0] = "-I."
// 	i := 1
// 	for n, _ := range pack.Files {
// 		if IsGoFile(n) {
// 			v[i] = n
// 			i++
// 		}
// 	}
// 
// 	cmd, err := exec.Run(goComp, v[0:i], nil, directory,
// 		exec.DevNull, exec.Pipe, exec.DevNull)
// 	if err != nil {
// 		return false, err
// 	}
// 	buf, err := ioutil.ReadAll(cmd.Stdout)
// 	if err != nil {
// 		return false, err
// 	}
// 	if string(buf) != "" {
// 		return false, nil
// 	}
// 	return true, nil
// }

func Rename(programTree *program.Program, filename string, line int, column int, newName string) (bool, *errors.GoRefactorError) {

	if filename == "" || !utils.IsGoFile(filename) {
		return false, errors.ArgumentError("filename", "It's not a valid go file name")
	}
	if line < 1 {
		return false, errors.ArgumentError("line", "Must be > 1")
	}
	if column < 1 {
		return false, errors.ArgumentError("column", "Must be > 1")
	}
	if !isGoIdent(newName) {
		return false, errors.ArgumentError("newName", "It's not a valid go identifier")
	}

	if sym, err := programTree.FindSymbolByPosition(filename, line, column); err == nil {

		if _, ok := sym.(*st.PointerTypeSymbol); ok {
			panic("find by position returned pointer type!!!")
		}
		if st.IsPredeclaredIdentifier(sym.Name()) {
			return false, errors.UnrenamableIdentifierError(sym.Name(), " It's a basic language symbol")
		}
		if sym.PackageFrom().IsGoPackage {
			return false, errors.UnrenamableIdentifierError(sym.Name(), " It's a symbol,imported from go library")
		}

		if _, ok := sym.Scope().LookUp(newName, filename); ok {
			return false, errors.IdentifierAlreadyExistsError(newName)
		}

		if meth, ok := sym.(*st.FunctionSymbol); ok {
			if meth.IsInterfaceMethod {
				return false, errors.UnrenamableIdentifierError(sym.Name(), " It's an interface method")
			}
		}
		count := renameSymbol(sym, newName)
		fmt.Printf("renamed %d occurences\n", count)
	} else {
		return false, err
	}
	return true, nil
}

func renameSymbol(sym st.Symbol, newName string) int {
	for ident, _ := range sym.Identifiers() {
		ident.Name = newName
	}
	return len(sym.Positions())
}
