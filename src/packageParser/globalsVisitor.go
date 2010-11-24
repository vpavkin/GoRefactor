package packageParser

import (
	"go/ast"
	"container/vector"
	"st"
)
import "fmt"

//Represents an ast.Visitor, walking along ast.tree and registering all the global variables met
type globalsVisitor struct {
	Parser   *packageParser
	iotaType st.ITypeSymbol
}

/*^GlobalsVisitor Methods^^*/

//ast.Visitor.Visit(). Looks for top-level ast.ValueSpec nodes of ast.Tree to register global vars
func (gv globalsVisitor) Visit(node interface{}) ast.Visitor {

	switch t := node.(type) {
	case *ast.ValueSpec:

		ts := gv.Parser.parseTypeSymbol(t.Type)

		if ts == nil && t.Values == nil {
			ts = gv.iotaType
		}

		for i, n := range t.Names {

			//fmt.Printf("%s:	Variable %s\n", gv.Parser.Package.AstPackage.Name, n.Name)

			var exprT st.ITypeSymbol

			if t.Values != nil {
				exprT = gv.Parser.parseExpr(t.Values[i]).At(0).(st.ITypeSymbol)
				if n.Name == "BothDir" {
					fmt.Printf("((((( %p, %T\n", exprT, exprT)
				}
			}
			if arrT, ok := ts.(*st.ArrayTypeSymbol); ok {
				if arrT.Len == st.ELLIPSIS {
					arrT.Len = exprT.(*st.ArrayTypeSymbol).Len
				}
			}
			if exprT != nil && ts == nil {
				ts = exprT
			}

			n.Obj = &ast.Object{Kind: ast.Var, Name: n.Name}

			toAdd := &st.VariableSymbol{Obj: n.Obj, VariableType: ts, Posits: new(vector.Vector), PackFrom: gv.Parser.Package}
			if gv.Parser.RegisterPositions {
				toAdd.AddPosition(st.NewOccurence(n.Pos()))
			}
			gv.Parser.RootSymbolTable.AddSymbol(toAdd)
		}

	case *ast.GenDecl:

		if len(t.Specs) > 0 {
			if vs, ok := t.Specs[0].(*ast.ValueSpec); ok {
				var ts st.ITypeSymbol
				if gv.Parser.RegisterPositions {
					gv.Parser.RegisterPositions = false
					ts = gv.Parser.parseTypeSymbol(vs.Type)
					gv.Parser.RegisterPositions = true
				} else {
					ts = gv.Parser.parseTypeSymbol(vs.Type)
				}

				if ts == nil {
					ts, _ = st.PredeclaredTypes["int"]
				}
				gv.iotaType = ts
			}
		}

	case *ast.ForStmt, *ast.FuncDecl, *ast.FuncLit, *ast.IfStmt, *ast.RangeStmt, *ast.SelectStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
		//InnerScope, omitted
		gv.iotaType = nil
		return nil
	}
	return gv
}
