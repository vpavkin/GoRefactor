package main;

import ("fmt"
//"go/printer"
"go/ast"
"go/parser"
"go/token"
//"go/typechecker"
//"packageParser"
//"os"
//"utils"
);

//ast.Visitor which prints ast.tree
type PrintNodeVisitor int

func (p PrintNodeVisitor) Visit(node interface{}) (w ast.Visitor) {
	if node != nil {
		for i := 0; i < int(p); i++ {
			fmt.Print(" ")
		}
		
		fmt.Printf("%T", node)
		
		if bl, ok := node.(*ast.BasicLit); ok {
			fmt.Printf(" (%s) ", bl.Kind)
		}
		
		if id, ok := node.(*ast.Ident); ok {
			fmt.Printf(" (%s) ", id.Name)
			if id.Obj != nil{
				fmt.Printf(" (%s) ", id.Obj.Name)
			}
		}
		
		fmt.Printf("\n")
	}
	p += 1
	return p
}

func main(){

	
	tree,err := parser.ParseFile(token.NewFileSet(), "/home/rulerr/go/src/pkg/unicode/tables.go",nil,parser.ParseComments);
	if err != nil {
		fmt.Println(err);
		return;
	}

	//typechecker.CheckFile(tree,nil);
	pnv := new(PrintNodeVisitor)
	ast.Walk(pnv, tree)

}
