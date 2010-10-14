package program

import (
	"container/vector"
	"st"
	"os"
	"utils/utils"
	"path"
	"packageParser"
	"go/parser"
	"go/ast"
	"strings"
)
import "fmt"


var program *Program
var externPackageTrees *vector.StringVector // [dir][packagename]package
var goSrcDir string

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
	externPackageTrees.Push("/home/rulerr/GoRefactor/src") // for tests on self

}

type Program struct {
	BaseSymbolTable *st.SymbolTable        //Base sT for parsing any package. Contains basic language symbols
	Packages        map[string]*st.Package //map[qualifiedPath] package
}

func locatePackages(srcDir string) {

	var isPackageDir bool

	fd, err := os.Open(srcDir, os.O_RDONLY, 0)
	if err != nil {
		panic("Couldn't open src directory")
		return
	}
	defer fd.Close()

	list, err := fd.Readdir(-1)
	if err != nil {
		panic("Couldn't read src directory")
		return
	}

	for i := 0; i < len(list); i++ {
		d := &list[i]
		if !d.IsDirectory() && utils.IsGoFile(d.Name) { //current dir describes a package
			isPackageDir = true
			break
		}
	}
	if isPackageDir {
		//fmt.Printf("%s: \n", path.Join(srcDir))
		packs, _ := parser.ParseDir(srcDir, utils.GoFilter, parser.ParseComments)

		_, d := path.Split(srcDir)
		if packTree, ok := packs[d]; !ok {
			fmt.Printf("Couldn't find a package " + d + " in " + d + " directory\n")
			return
		} else {
			pack := st.NewPackage(srcDir, packTree)
			program.Packages[srcDir] = pack
		}
	} else {
		for i := 0; i < len(list); i++ {
			d := &list[i]
			if d.IsDirectory() { //current dir describes a package
				locatePackages(path.Join(srcDir, d.Name))
			}
		}
	}
}

func ParseProgram(srcDir string) *Program {

	program = &Program{st.NewSymbolTable(nil), make(map[string]*st.Package)}

	initialize()

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
			ast.PackageExports(pack.AstPackage)
		}
	}

	for _, pack := range program.Packages {
		
		fmt.Printf("PACKAGE %s:\n", pack.AstPackage.Name);
		
		pack.Symbols.AddOpenedScope(program.BaseSymbolTable)
		go packageParser.ParsePackage(pack);
	}
	
	// type resolving
	for _, pack := range program.Packages {
		<- pack.Communication;
	}
	for _, pack := range program.Packages {
		pack.Communication <- 0;
	}
	for _, pack := range program.Packages {
		<- pack.Communication;
	}
	fmt.Printf("===================All packages stopped fixing \n");
	
	/*df,err := os.Open(path.Join(srcDir, d.Name), os.O_RDONLY, 0)
	if err != nil {
		panic("Couldn't open package" + d.Name + " directory")
		return nil
	}
	fileList, err := df.Readdir(-1)
	if err != nil {
		panic("Couldn't read package " + d.Name + " directory")
		return nil
	}
	for j := 0; j < len(fileList); j++ {
		f := &fileList[j]
		fmt.Printf("	%s: \n", f.Name)
	}*/
	return program
}

func IsGoSrcPackage(p *st.Package) bool {
	//fmt.Printf("IS GO? %s %s\n", p.QualifiedPath,goSrcDir)
	return strings.HasPrefix(p.QualifiedPath, goSrcDir)
}

func (p *Program) renameSymbol(sym st.Symbol, newName string) int {
	sym.Object().Name = newName

	/*for _,pos := range *sym.Positions() {
		obj := pos.(symbolTable.Occurence).Obj
		obj.Name = newName
	}*/

	return sym.Positions().Len()
}
