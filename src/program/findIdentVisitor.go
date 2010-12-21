package program

import "go/ast"
import "go/token"
import "utils"
import "st"


type findIdentVisitor struct {
	Package *st.Package
	Ident   *ast.Ident
	Pos     token.Position
}

func (fv *findIdentVisitor) Visit(node interface{}) ast.Visitor {

	if id, ok := node.(*ast.Ident); ok {

		if utils.ComparePosWithinFile(fv.Package.FileSet.Position(id.Pos()), fv.Pos) == 0 {
			fv.Ident = id
			return nil
		}
	}
	return fv
}

func getTopLevelDecl(Package *st.Package, file *ast.File, pos token.Position) ast.Decl {
	for i, decl := range file.Decls {
		if utils.ComparePosWithinFile(Package.FileSet.Position(decl.Pos()), pos) == 1 {
			return file.Decls[i-1]
		}
	}
	return file.Decls[len(file.Decls)-1]
}

func findIdentByPos(Package *st.Package, file *ast.File, pos token.Position) (obj *ast.Ident, found bool) {
	visitor := &findIdentVisitor{Package, nil, pos}
	declToSearch := getTopLevelDecl(Package, file, pos)
	ast.Walk(visitor, declToSearch)
	if visitor.Ident == nil {
		return nil, false
	}
	return visitor.Ident, true
}
