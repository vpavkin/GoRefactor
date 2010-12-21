package refactoring

import(
	"go/ast"
	"container/vector"
	"st"
	"go/token"
	"fmt"
	"utils"
	"errors"
	"program"
)

type extractedSetVisitor struct{
	Package *st.Package
	
	resultBlock *vector.Vector
	firstNodePos token.Position  //in
	lastNodePos token.Position	//in
	firstNode ast.Node
	lastNode ast.Node
	
	listFrom []ast.Stmt;
	
}

func (vis *extractedSetVisitor) foundFirst() bool{
	return vis.firstNode != nil;
}

func (vis *extractedSetVisitor) isValid() bool{
	return (vis.firstNode != nil) && (vis.lastNode != nil);
}

func (vis *extractedSetVisitor) checkStmtList(list []ast.Stmt){
	
	for i,stmt := range list{
		if !vis.foundFirst(){
			if utils.ComparePosWithinFile(vis.Package.FileSet.Position(stmt.Pos()),vis.firstNodePos) == 0 {
				vis.firstNode = stmt;
				vis.listFrom = list;
				vis.resultBlock.Push(stmt);
				
				if utils.ComparePosWithinFile(vis.Package.FileSet.Position(stmt.Pos()),vis.lastNodePos) == 0 {
					//one stmt method
					vis.lastNode = stmt;
				}
			}
		}else{
			if !vis.isValid(){
				vis.resultBlock.Push(stmt)
				fmt.Printf("met pos %v\n", vis.Package.FileSet.Position(stmt.Pos()));
				if utils.ComparePosWithinFile(vis.Package.FileSet.Position(stmt.Pos()),vis.lastNodePos) == 0{
					vis.lastNode = list[i];
				}
			}
		}
	}
}
func (vis *extractedSetVisitor) Visit(node interface{}) ast.Visitor{
	
	if vis.foundFirst(){
		return nil
	}
	switch t := node.(type){
		case *ast.BlockStmt:
			vis.checkStmtList(t.List);
		
// 			if vis.foundFirst() && !vis.isValid(){
// 				if utils.ComparePosWithinFile(vis.Package.FileSet.Position(t.Rbrace),vis.lastNodePos) == 0{
// 					vis.lastNode = t.List[len(t.List)-1];
// 				}
// 			}
		case *ast.SwitchStmt:
			for _,cc := range t.Body.List{
				caseClause := cc.(*ast.CaseClause);
				vis.checkStmtList(caseClause.Body);
			}
		case *ast.TypeSwitchStmt:
			for _,cc := range t.Body.List{
				caseClause := cc.(*ast.TypeCaseClause);
				vis.checkStmtList(caseClause.Body);
			}
		case *ast.SelectStmt:
			for _,cc := range t.Body.List{
				caseClause := cc.(*ast.CommClause);
				vis.checkStmtList(caseClause.Body);
			}
		case *ast.BasicLit:
			return nil;
		case ast.Expr:
			if utils.ComparePosWithinFile(vis.Package.FileSet.Position(t.Pos()),vis.lastNodePos) == 0{
				if  utils.ComparePosWithinFile(vis.firstNodePos,vis.lastNodePos) == 0{
					vis.firstNode = t;
					vis.lastNode = t;
					vis.resultBlock.Push(t);
					//*t = &ast.CallExpr{};
				}
			}
	}
	return vis;
}

func CheckExtractMethodParameters(filename string,lineStart int,colStart int,lineEnd int,colEnd int,methodName string,recieverVarLine int,recieverVarCol int) (bool,*errors.GoRefactorError){
	switch {
		case filename == "" || !utils.IsGoFile(filename):
			return false,errors.ArgumentError("filename", "It's not a valid go file name")
		case lineStart < 1 :
			return false, errors.ArgumentError("lineStart", "Must be > 1")
		case lineEnd < 1 || lineEnd < lineStart :
			return false, errors.ArgumentError("lineEnd", "Must be > 1 and >= lineStart")
		case colStart < 1 :
			return false, errors.ArgumentError("colStart", "Must be > 1")
		case colEnd < 1:
			return false, errors.ArgumentError("colEnd", "Must be > 1")
		case !IsGoIdent(methodName) :
			return false, errors.ArgumentError("methodName", "It's not a valid go identifier")
	}
	return true,nil;
}

func ExtractMethod(programTree *program.Program,filename string,lineStart int,colStart int,lineEnd int,colEnd int,methodName string,recieverVarLine int,recieverVarCol int) (bool,*errors.GoRefactorError){
	
	if ok,err := CheckExtractMethodParameters(filename,lineStart,colStart,lineEnd,colEnd,methodName,recieverVarLine,recieverVarCol);!ok{
		return false,err;
	}
	
	pack,file := programTree.FindPackageAndFileByFilename(filename);
	if pack == nil {
		return false, errors.ArgumentError("filename", "Program packages don't contain file '"+filename+"'")
	}
	
	vis := &extractedSetVisitor{pack,new(vector.Vector),token.Position{filename,0,lineStart,colStart},token.Position{filename,0,lineEnd,colEnd},nil,nil,nil}
	fmt.Printf("poses %v %v\n",vis.firstNodePos,vis.lastNodePos);
	ast.Walk(vis,file);
	if vis.isValid(){
		fmt.Println("valid:");
		for _,el := range *vis.resultBlock{
			fmt.Printf("%v\n",el);
		}
	}else{
		fmt.Println("invalid:");
		fmt.Printf("first : %v\n",vis.firstNode);
		fmt.Printf("last : %v\n",vis.lastNode);
	}
	return true,nil
}
//TODO: if, switch, for init
