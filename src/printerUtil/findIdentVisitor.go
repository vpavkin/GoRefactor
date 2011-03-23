package printerUtil

import "go/ast"
import "go/token"
import "refactoring/utils"

type findIdentVisitor struct {
	FileSet *token.FileSet
	Ident   *ast.Ident
	Pos     token.Position
}

func (fv *findIdentVisitor) Visit(node ast.Node) ast.Visitor {

	if id, ok := node.(*ast.Ident); ok {

		if utils.ComparePosWithinFile(fv.FileSet.Position(id.Pos()), fv.Pos) == 0 {
			fv.Ident = id
			return nil
		}
	}
	return fv
}

func getTopLevelDecl(FileSet *token.FileSet, file *ast.File, pos token.Position) ast.Decl {
	for i, decl := range file.Decls {
		if utils.ComparePosWithinFile(FileSet.Position(decl.Pos()), pos) == 1 {
			return file.Decls[i-1]
		}
	}
	return file.Decls[len(file.Decls)-1]
}

func FindIdentByPos(FileSet *token.FileSet, file *ast.File, pos token.Position) (obj *ast.Ident, found bool) {
	visitor := &findIdentVisitor{FileSet, nil, pos}
	declToSearch := getTopLevelDecl(FileSet, file, pos)
	ast.Walk(visitor, declToSearch)
	if visitor.Ident == nil {
		return nil, false
	}
	return visitor.Ident, true
}
