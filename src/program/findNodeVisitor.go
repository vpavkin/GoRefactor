package program

import "go/ast"
import "go/token"
import "utils"

type findNodeVisitor struct{
	Node interface{}
	Pos token.Position
}

func (fv *findNodeVisitor) Visit(node interface{}) ast.Visitor{
	if astNode,ok := node.(ast.Node);ok{
		//this code is to avoid issue 1326
		if _,ok := astNode.(*ast.BasicLit);ok{
			return fv;
		}
		if utils.ComparePosWithinFile(astNode.Pos(),fv.Pos) == 0{
			fv.Node = astNode;
			return nil;
		}
	}
	return fv;
}

func getTopLevelDecl (file *ast.File,pos token.Position) ast.Decl{
	for i,decl := range file.Decls{
		if utils.ComparePosWithinFile(decl.Pos(), pos) == 1 {
			return file.Decls[i-1]
		}
	}
    return file.Decls[len(file.Decls) - 1];

}

func findNodeByPos(file *ast.File,pos token.Position) (node interface{},found bool){
	visitor := &findNodeVisitor{nil,pos}
	declToSearch:= getTopLevelDecl(file,pos);
	ast.Walk(visitor,declToSearch);
	if visitor.Node == nil{
		return nil,false;
	}
	return visitor.Node,true;
}

func findObjectByPos(file *ast.File,pos token.Position) (obj *ast.Object,found bool){
	if node,ok := findNodeByPos(file,pos);ok{
		if id,ok :=  node.(*ast.Ident);ok{
			return id.Obj,true;
		}
	}
	return nil,false;
}
