package program

import (
	"go/ast"
	//"go/parser"
	"refactoring/st"
	"container/vector"
	//"go/token"
	"path"
)
//import "fmt"

//Represents an ast.Visitor, walking along ast.tree and registering all the imports met
type importsVisitor struct {
	Package         *st.Package
	FileName        string
	specialPackages map[string][]string
}

func (iv *importsVisitor) Visit(node ast.Node) (w ast.Visitor) {
	w = iv
	if is, ok := node.(*ast.ImportSpec); ok {

		Path := string(is.Path.Value)
		Path = Path[1 : len(Path)-1] //remove quotes

		var (
			name     string
			found    bool
			pack     *st.Package
			packTree *ast.Package
		)

		if is.Name != nil {
			switch is.Name.Name {
			case "_":
				return // package imported only for side-effects
			case ".":
				name = "." //, hasLocalName = ".", false
			default:
				name = is.Name.Name //, hasLocalName =, true
			}
		} else {
			_, name = path.Split(Path)
			//hasLocalName = false
		}

		pack, found = program.FindPackageByGoPath(Path)

		if !found {
			_, f := path.Split(Path)
			fileSet, dirTree, _ := getAstTree(path.Join(goSrcDir, Path), iv.specialPackages[Path])
			if dirTree != nil {
				if packTree, found = dirTree[f]; found {
					pack = st.NewPackage(path.Join(goSrcDir, Path), Path, fileSet, packTree)
					program.Packages[pack.QualifiedPath] = pack
					parseImports(pack, iv.specialPackages)
				} else {
					panic("package not found where expected: " + path.Join(goSrcDir, Path))
				}
			}
			for dir, goPath := range packages {
				if goPath == Path {
					fileSet, dirTree, _ := getAstTree(dir, iv.specialPackages[Path])
					if dirTree != nil {
						if packTree, found = dirTree[f]; found {
							pack = st.NewPackage(dir, Path, fileSet, packTree)
							program.Packages[pack.QualifiedPath] = pack
							parseImports(pack, iv.specialPackages)
							break
						} else {
							panic("package not found where expected: " + dir)
						}
					} else {
						panic("package not found where expected: " + dir)
					}
				}
			}
		}
		if _, isIn := iv.Package.Imports[iv.FileName]; !isIn {
			iv.Package.Imports[iv.FileName] = new(vector.Vector)
		}

		sym := st.MakePackage(name, iv.Package.Symbols, Path, pack)

		if is.Name != nil {

			sym.AddIdent(is.Name)
			program.IdentMap.AddIdent(is.Name, sym)

			sym.AddPosition(iv.Package.FileSet.Position(is.Name.Pos()))

			if is.Name.Name == "." {
				iv.Package.Symbols.AddOpenedScope(pack.Symbols)
			}
		}

		iv.Package.Imports[iv.FileName].Push(sym)
		//------------

		return

	}
	return
}

func parseImports(pack *st.Package, specialPackages map[string][]string) {
	for fName, f := range pack.AstPackage.Files {
		iv := &importsVisitor{pack, fName, specialPackages}
		ast.Walk(iv, f)
	}
}
