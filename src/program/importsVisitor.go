package program

import (
	"go/ast"
	//"go/parser"
	"st"
	"container/vector"
	//"go/token"
	"path"
)
//import "fmt"

//Represents an ast.Visitor, walking along ast.tree and registering all the types met
type importsVisitor struct {
	Package  *st.Package
	FileName string
}

//ast.Visitor.Visit(). Looks for ast.TypeSpec nodes of ast.Tree to register new types
func (iv *importsVisitor) Visit(node interface{}) (w ast.Visitor) {
	w = iv
	if is, ok := node.(*ast.ImportSpec); ok {

		Path := string(is.Path.Value)
		Path = Path[1 : len(Path)-1] //remove quotes

		var (
			name         string
// 			hasLocalName bool
			found        bool
			pack         *st.Package
			packTree     *ast.Package
		)

		if is.Name != nil {
			switch is.Name.Name {
			case "_":
				return // package imported only for side-effects
			case ".":
				name = "."//, hasLocalName = ".", false
			default:
				name =  is.Name.Name//, hasLocalName =, true
			}
		} else {
			_, name = path.Split(Path)
			//hasLocalName = false
		}

		for _, dir := range *externPackageTrees {
			if pack, found = program.Packages[path.Join(dir, Path)]; found {
				break
			}
		}
		if !found {
			_, f := path.Split(Path)
			for _, dir := range *externPackageTrees {
				fileSet, dirTree, _ := getAstTree(path.Join(dir, Path))
				if dirTree != nil {
					if packTree, found = dirTree[f]; found {
						pack = st.NewPackage(path.Join(dir, Path), fileSet, packTree)
						program.Packages[pack.QualifiedPath] = pack

						parseImports(pack)

						break
					} else {
						panic("package not found where expected: " + path.Join(dir, Path))
					}

				}
			}
		}
		if _, isIn := iv.Package.Imports[iv.FileName]; !isIn {
			iv.Package.Imports[iv.FileName] = new(vector.Vector)
		}

		sym := st.MakePackage(name,iv.Package, Path, pack)
		
		if is.Name != nil {
			
			sym.AddIdent(is.Name)
			program.IdentMap.AddIdent(is.Name,sym)
			
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

func parseImports(pack *st.Package) {
	for fName, f := range pack.AstPackage.Files {
		iv := &importsVisitor{pack, fName}
		ast.Walk(iv, f)
	}
}
