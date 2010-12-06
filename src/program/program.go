package program

import (
	"container/vector"
	"st"
	"os"
	"utils"
	"path"
	"packageParser"
	"go/parser"
	"go/token"
	"bufio"
	"go/ast"
	"strings"
	"errors"
)
import "fmt"


var program *Program
var externPackageTrees *vector.StringVector // [dir][packagename]package
var goSrcDir string
var specificFiles map[string]*vector.StringVector
var specificFilesPackages []string = []string{"syscall", "os","runtime"}

func initialize() {

	for _, s := range st.PredeclaredTypes {
		program.BaseSymbolTable.AddSymbol(s)
	}
	for _, s := range st.PredeclaredFunctions {
		program.BaseSymbolTable.AddSymbol(s)
	}
	for _, s := range st.PredeclaredConsts {
		program.BaseSymbolTable.AddSymbol(s)
	}

	goRoot := os.Getenv("GOROOT")
	goSrcDir = path.Join(goRoot, "src", "pkg")

	externPackageTrees = new(vector.StringVector)
	externPackageTrees.Push(goSrcDir)
	//externPackageTrees.Push("/home/rulerr/GoRefactor/src") // for tests on self

	specificFiles = make(map[string]*vector.StringVector)

}

type Program struct {
	BaseSymbolTable *st.SymbolTable        //Base sT for parsing any package. Contains basic language symbols
	Packages        map[string]*st.Package //map[qualifiedPath] package
}

func loadConfig(packageName string) *vector.StringVector {
	fd, err := os.Open(packageName+".cfg", os.O_RDONLY, 0)
	if err != nil {
		println(err.String())
		panic("Couldn't open " + packageName + " config")
	}
	defer fd.Close()

	res := new(vector.StringVector)

	reader := bufio.NewReader(fd)
	for {

		str, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		res.Push(str[:len(str)-1])

	}
	fmt.Printf("%s:\n%v\n", packageName, res)

	return res
}

func isPackageDir(fileInIt *os.FileInfo) bool {
	return !fileInIt.IsDirectory() && utils.IsGoFile(fileInIt.Name)
}

func makeFilter(srcDir string) func(f *os.FileInfo) bool {
	_, d := path.Split(srcDir)
	println("^&*^&* Specific files for " + d)
	if files, ok := specificFiles[d]; ok {
		println("^&*^&* found " + d)
		return func(f *os.FileInfo) bool {
			print("\n" + f.Name)
			for _, fName := range *files {
				print(" " + fName)
				if fName == f.Name {
					return true
				}
			}
			return false
		}
	}
	return utils.GoFilter

}
func getFullNameFiles(files *vector.StringVector,srcDir string) ([]string){
	var res vector.StringVector;
	for _,fName := range *files{
		res.Push(path.Join(srcDir, fName));
	}
	return res;
}
func getAstTree(srcDir string) (*token.FileSet, map[string]*ast.Package,os.Error){
	_, d := path.Split(srcDir);
	fileSet := token.NewFileSet();
	if files, ok := specificFiles[d]; ok {
		pckgs,err := parser.ParseFiles(fileSet,getFullNameFiles(files,srcDir),parser.ParseComments)
		return fileSet,pckgs,err
	}
	pckgs,err := parser.ParseDir(fileSet,srcDir, utils.GoFilter, parser.ParseComments)
	return fileSet,pckgs,err
	
}
func parsePack(srcDir string) {
	
	fileSet,packs,err := getAstTree(srcDir)
	if err!=nil{
		fmt.Printf("SOME ERRORS while parsing pack "+ srcDir);
	}

	_, d := path.Split(srcDir)
	if packTree, ok := packs[d]; !ok {
		panic("Couldn't find a package " + d + " in " + d + " directory")
	} else {
		pack := st.NewPackage( srcDir,fileSet, packTree)
		program.Packages[srcDir] = pack;
	}
}

func locatePackages(srcDir string) {

	fd, err := os.Open(srcDir, os.O_RDONLY, 0)
	if err != nil {
		panic("Couldn't open src directory")
	}
	defer fd.Close()

	list, err := fd.Readdir(-1)
	if err != nil {
		panic("Couldn't read src directory")
	}

	for i := 0; i < len(list); i++ {
		d := &list[i]
		if isPackageDir(d) { //current dir describes a package
			parsePack(srcDir)
			return
		}
	}

	//no package in this dir, look inside dirs' dirs
	for i := 0; i < len(list); i++ {
		d := &list[i]
		if d.IsDirectory() { //can still contain packages inside
			locatePackages(path.Join(srcDir, d.Name))
		}
	}

}

func ParseProgram(srcDir string,externSourceFolders *vector.StringVector) *Program {

	program = &Program{st.NewSymbolTable(nil), make(map[string]*st.Package)}

	initialize()
	externPackageTrees.Push(srcDir);
	if externSourceFolders != nil{
		externPackageTrees.AppendVector(externSourceFolders);
	}

	for _, pName := range specificFilesPackages {
		specificFiles[pName] = loadConfig(pName)
	}

	locatePackages(srcDir)

	packs := new(vector.Vector)
	for _, pack := range program.Packages {
		packs.Push(pack)
	}

	// Recursively fills program.Packages map.
	for _, ppack := range *packs {
		pack := ppack.(*st.Package)
		parseImports(pack)
	}

	for _, pack := range program.Packages {
		if IsGoSrcPackage(pack) {
			pack.IsGoPackage = true
			//ast.PackageExports(pack.AstPackage)
		}
	}

	for _, pack := range program.Packages {

		pack.Symbols.AddOpenedScope(program.BaseSymbolTable)
		go packageParser.ParsePackage(pack)
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
	fmt.Printf("===================All packages stopped fixing \n")

	for _, pack := range program.Packages {
		pack.Communication <- 0
		<-pack.Communication
	}

	// 	for _, pack := range program.Packages {
	// 		
	// 	}
	fmt.Printf("===================All packages stopped opening \n")

	for _, pack := range program.Packages {
		pack.Communication <- 0
		<-pack.Communication
	}

	// 	for _, pack := range program.Packages {
	// 		
	// 	}
	fmt.Printf("===================All packages stopped parsing globals \n")
	for _, pack := range program.Packages {
		pack.Communication <- 0
		<-pack.Communication
	}

// 	for _, pack := range program.Packages {
// 		
// 	}
	fmt.Printf("===================All packages stopped fixing globals \n")
	for _, pack := range program.Packages {
		pack.Communication <- 0
		<-pack.Communication
	}

	// 	for _, pack := range program.Packages {
	// 		
	// 	}
	fmt.Printf("===================All packages stopped parsing locals \n")
	
	return program
}

func IsGoSrcPackage(p *st.Package) bool {
	//fmt.Printf("IS GO? %s %s\n", p.QualifiedPath,goSrcDir)
	return strings.HasPrefix(p.QualifiedPath, goSrcDir)
}

func(p *Program)findPackageAndFileByFilename(filename string) (*st.Package,*ast.File){
	for _,pack:= range p.Packages{
		for fName, file := range pack.AstPackage.Files{
			if filename == fName{
				return pack, file
			}
		}
	}
	return nil,nil;
}

func (p *Program) FindSymbolByPosition(filename string, line int, column int) (symbol st.Symbol,containsIn *st.SymbolTable,error *errors.GoRefactorError) {
	packageIn,fileIn := p.findPackageAndFileByFilename(filename)
	if packageIn == nil{
		return nil,nil,errors.ArgumentError("filename","Program packages don't contain file '" + filename + "'");
	}
	
	obj,found := findObjectByPos(packageIn,fileIn,token.Position{Filename:filename,Line:line,Column:column})
	if !found{
		return nil,nil,errors.IdentifierNotFoundError(filename, line, column);
	}
	for _,el:= range *packageIn.SymbolTablePool{
		sT:= el.(*st.SymbolTable);
		if sym,found := sT.FindSymbolByObject(obj); found{
			return sym,sT,nil
		}
	}
	for _,el:= range *(packageIn.Imports[filename]){
		pSym := el.(*st.PackageSymbol)
		if sym,found := pSym.Package.Symbols.FindSymbolByObject(obj); found{
			return sym,pSym.Package.Symbols,nil
		}
	}
	return nil,nil,errors.IdentifierNotFoundError(filename, line, column);
}
