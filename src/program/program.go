package program

import (
	"container/vector"
	"refactoring/st"
	"os"
	"refactoring/utils"
	"path"
	"refactoring/packageParser"
	"go/parser"
	"go/token"
	"go/printer"
	"go/ast"
	"strings"
	"refactoring/errors"
)
import "fmt"


var program *Program
var packages map[string]string //address, go pkg address (import address)
var goSrcDir string

func initialize() {

	for _, s := range st.PredeclaredTypes {
		program.BaseSymbolTable.AddSymbol(s)
		s.Scope_ = program.BaseSymbolTable
	}
	for _, s := range st.PredeclaredFunctions {
		program.BaseSymbolTable.AddSymbol(s)
		s.Scope_ = program.BaseSymbolTable
	}
	for _, s := range st.PredeclaredConsts {
		program.BaseSymbolTable.AddSymbol(s)
		s.Scope_ = program.BaseSymbolTable
	}

	goRoot := os.Getenv("GOROOT")
	if goRoot == "" {
		panic("please, set environment variable GOROOT (usualy $HOME/go)")
	}
	goSrcDir = path.Join(goRoot, "src", "pkg")

	packages = make(map[string]string)

}


type Program struct {
	BaseSymbolTable *st.SymbolTable        //Base sT for parsing any package. Contains basic language symbols
	Packages        map[string]*st.Package //map[qualifiedPath] package
	IdentMap        st.IdentifierMap
}

func isPackageDir(fileInIt *os.FileInfo) bool {
	return !fileInIt.IsDirectory() && utils.IsGoFile(fileInIt.Name)
}

func getFullNameFiles(files []string, srcDir string) []string {
	res := []string{}
	for _, fName := range files {
		res = append(res, path.Join(srcDir, fName))
	}
	return res
}
func getAstTree(srcDir string, specialFiles []string) (*token.FileSet, map[string]*ast.Package, os.Error) {
	fileSet := token.NewFileSet()
	if specialFiles != nil {
		pckgs, err := parser.ParseFiles(fileSet, getFullNameFiles(specialFiles, srcDir), parser.ParseComments)
		return fileSet, pckgs, err
	}
	pckgs, err := parser.ParseDir(fileSet, srcDir, utils.GoFilter, parser.ParseComments)
	return fileSet, pckgs, err

}
func parsePack(srcDir string, specialFiles []string) {

	fileSet, packs, err := getAstTree(srcDir, specialFiles)
	if err != nil {
		fmt.Printf("Warning: some errors occured during parsing package %s:\n %v\n", srcDir, err)
	}

	_, d := path.Split(srcDir)
	if packTree, ok := packs[d]; !ok {
		panic("Couldn't find a package " + d + " in directory \"" + srcDir + "\"")
	} else {
		pack := st.NewPackage(srcDir, packages[srcDir], fileSet, packTree)
		program.Packages[srcDir] = pack
	}
}

func locatePackage(dir string, specialFiles []string) {

	fd, err := os.Open(dir, os.O_RDONLY, 0)
	if err != nil {
		panic("Couldn't open directory \"" + dir + "\"")
	}
	defer fd.Close()

	list, err := fd.Readdir(-1)
	if err != nil {
		panic("Couldn't read from directory \"" + dir + "\"")
	}

	for i := 0; i < len(list); i++ {
		d := &list[i]
		if isPackageDir(d) { //current dir describes a package
			parsePack(dir, specialFiles)
			return
		}
	}
	panic("invalid .package field entity \"" + dir + "\"")
}

func ParseProgram(projectDir string, sources map[string]string, specialPackages map[string][]string) *Program {

	program = &Program{st.NewSymbolTable(nil), make(map[string]*st.Package), make(map[*ast.Ident]st.Symbol)}

	initialize()
	for fldr, goPath := range sources {
		packages[fldr] = goPath
	}

	for fldr, goPath := range sources {
		locatePackage(fldr, specialPackages[goPath])
	}

	packs := new(vector.Vector)
	for _, pack := range program.Packages {
		packs.Push(pack)
	}

	// Recursively fills program.Packages map.
	for _, ppack := range *packs {
		pack := ppack.(*st.Package)
		parseImports(pack, specialPackages)
	}

	for _, pack := range program.Packages {
		if IsGoSrcPackage(pack) {
			pack.IsGoPackage = true
		}
	}

	for _, pack := range program.Packages {

		pack.Symbols.AddOpenedScope(program.BaseSymbolTable)
		go packageParser.ParsePackage(pack, program.IdentMap)
	}
	for _, pack := range program.Packages {
		pack.Communication <- 0
		<-pack.Communication
	}
	// type resolving
	// 	for _, pack := range program.Packages {
	// 		
	// 	}
	for _, pack := range program.Packages {
		pack.Communication <- 0
		<-pack.Communication
	}
	// 	for _, pack := range program.Packages {
	// 		
	// 	}
	// 	fmt.Printf("===================All packages stopped fixing \n")

	for _, pack := range program.Packages {
		pack.Communication <- 0
		<-pack.Communication
	}

	// 	for _, pack := range program.Packages {
	// 		
	// 	}
	// 	fmt.Printf("===================All packages stopped opening \n")

	for _, pack := range program.Packages {
		pack.Communication <- 0
		<-pack.Communication
	}

	// 	for _, pack := range program.Packages {
	// 		
	// 	}
	// 	fmt.Printf("===================All packages stopped parsing globals \n")
	for _, pack := range program.Packages {
		pack.Communication <- 0
		<-pack.Communication
	}

	// 	for _, pack := range program.Packages {
	// 		
	// 	}
	// 	fmt.Printf("===================All packages stopped fixing globals \n")
	for _, pack := range program.Packages {
		pack.Communication <- 0
		<-pack.Communication
	}

	// 	for _, pack := range program.Packages {
	// 		
	// 	}
	// 	fmt.Printf("===================All packages stopped parsing locals \n")

	return program
}

func IsGoSrcPackage(p *st.Package) bool {
	//fmt.Printf("IS GO? %s %s\n", p.QualifiedPath,goSrcDir)
	return strings.HasPrefix(p.QualifiedPath, goSrcDir)
}

func (p *Program) FindPackageByGoPath(goPath string) (*st.Package, bool) {
	for _, pack := range p.Packages {
		if pack.GoPath == goPath {
			return pack, true
		}
	}
	return nil, false
}

func (p *Program) FindPackageAndFileByFilename(filename string) (*st.Package, *ast.File) {
	for _, pack := range p.Packages {
		for fName, file := range pack.AstPackage.Files {
			if filename == fName {
				return pack, file
			}
		}
	}
	return nil, nil
}

func (p *Program) FindSymbolByPosition(filename string, line int, column int) (symbol st.Symbol, error *errors.GoRefactorError) {
	packageIn, fileIn := p.FindPackageAndFileByFilename(filename)
	if packageIn == nil {
		return nil, errors.ArgumentError("filename", "Program packages don't contain file '"+filename+"'")
	}

	ident, found := findIdentByPos(packageIn, fileIn, token.Position{Filename: filename, Line: line, Column: column})
	if !found {
		return nil, errors.IdentifierNotFoundError(filename, line, column)
	}

	if sym, ok := p.IdentMap[ident]; ok {
		return sym, nil
	} else {
		panic("untracked ident " + ident.Name)
	}

	return nil, errors.IdentifierNotFoundError(filename, line, column)
}

func (p *Program) GetPointerType(t st.ITypeSymbol) *st.PointerTypeSymbol {
	pack := t.PackageFrom()
	res, ok := pack.Symbols.LookUpPointerType(t.Name(), 1)
	if ok {
		return res
	}
	return st.MakePointerType(t.Scope(), t)
}

func (p *Program) Save() {
	for _, pack := range p.Packages {
		if pack.IsGoPackage {
			continue
		}
		fmt.Printf("saving package: %s\n", pack.AstPackage.Name)
		for filename, _ := range pack.AstPackage.Files {
			p.SaveFile(filename)
		}
	}
}
func (p *Program) SaveFile(filename string) {
	pack, file := p.FindPackageAndFileByFilename(filename)
	fmt.Printf("saving file: %s\n", filename)
	fd, err := os.Open(filename, os.O_EXCL|os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		panic("couldn't open file " + filename + "for writing")
	}
	cfg := &printer.Config{printer.TabIndent, 8}
	_, err = cfg.Fprint(fd, pack.FileSet, file)
	if err != nil {
		panic("couldn't write to file " + filename)
	}
	fd.Close()
}

func (p *Program) SaveFileExplicit(filename string, fset *token.FileSet, file *ast.File) {
	fmt.Printf("saving file: %s\n", filename)
	fd, err := os.Open(filename, os.O_EXCL|os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		panic("couldn't open file " + filename + "for writing")
	}
	cfg := &printer.Config{printer.TabIndent, 8}
	_, err = cfg.Fprint(fd, fset, file)
	if err != nil {
		panic("couldn't write to file " + filename)
	}
	fd.Close()
}
