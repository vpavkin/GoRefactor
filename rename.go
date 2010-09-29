package symbolTable

import (
	"go/parser"
	"os"
	"go/ast"
	"io/ioutil"
	"exec"
	"unicode"
)

const (
	INVALID_PARAMETER = -iota - 1
	INVALID_IDENTIFIER
	PARSE_ERROR
	DIRECTORY_OR_FILE_NOT_FOUND
	IDENTIFIER_ALREADY_EXISTS
	IDENTIFIER_NOT_FOUND_OR_UNRENAMABLE
	COMPILE_ERROR
)

func IsGoFileInfo(file *os.FileInfo) bool {
	return IsGoFile(file.Name)
}

func IsGoIdent(name string) bool {

	if name == "_" || name == "nil" || name == "true" || name == "false" {
		return false
	}
	if IsPredeclaredType(name) || IsBuiltInFunction(name) {
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

func Compile(directory string, pack *ast.Package) (bool, os.Error) {

	goComp, err := exec.LookPath("8g")
	if err != nil {
		return false, err
	}
	v := make([]string, len(pack.Files)+1)
	v[0] = "-I."
	i := 1
	for n, _ := range pack.Files {
		if IsGoFile(n) {
			v[i] = n
			i++
		}
	}

	cmd, err := exec.Run(goComp, v[0:i], nil, directory,
		exec.DevNull, exec.Pipe, exec.DevNull)
	if err != nil {
		return false, err
	}
	buf, err := ioutil.ReadAll(cmd.Stdout)
	if err != nil {
		return false, err
	}
	if string(buf) != "" {
		return false, nil
	}
	return true, nil
}

func RenameIdent(directory string, filename string, line int, column int, newName string) (*ast.Package, int) {
	if directory == "" || filename == "" || line < 0 || column < 0 || newName == "" {
		return nil, INVALID_PARAMETER
	}

	if !IsGoIdent(newName) {
		return nil, INVALID_IDENTIFIER
	}

	pack, err := parser.ParseDir(directory, IsGoFileInfo, parser.ParseComments)
	if err != nil {
		return nil, PARSE_ERROR //Parse Error
	}

	var p *ast.Package
	for _, pp := range pack {
		for FileName, _ := range pp.Files {
			if IsGoFile(FileName) && FileName == filename {
				p = pp
				break
			}
		}
		if p != nil {
			break
		}
	}

	if p == nil {
		return nil, DIRECTORY_OR_FILE_NOT_FOUND //file not found
	}

	if compiled, _ := Compile(directory, p); !compiled {
		return nil, COMPILE_ERROR //not compiled
	}

	STB.BuildSymbolTable(p)

	count := 0

	if sym, st, found := STB.FindSymbolByPosition(filename, line, column); found {
		if _, ok := st.FindSymbolByName(newName); ok {
			return nil, IDENTIFIER_ALREADY_EXISTS
		}
		if meth, ok := sym.(*FunctionSymbol); ok {
			if meth.Locals == nil {
				return nil, IDENTIFIER_NOT_FOUND_OR_UNRENAMABLE
			}
		}
		count = STB.RenameSymbol(sym, newName)
	} else {
		return nil, IDENTIFIER_NOT_FOUND_OR_UNRENAMABLE
	}
	return p, count
}
