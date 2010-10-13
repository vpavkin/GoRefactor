package packageParser

import (
	"go/ast"
	"container/vector"
	"st"
)
import "fmt"

//Represents an ast.Visitor, walking along ast.tree and registering all the global variables met
type globalsVisitor struct {
	Parser *packageParser
}

var iotaType st.ITypeSymbol
/*^GlobalsVisitor Methods^^*/

//ast.Visitor.Visit(). Looks for top-level ast.ValueSpec nodes of ast.Tree to register global vars
func (gv globalsVisitor) Visit(node interface{}) ast.Visitor {

	switch t := node.(type) {
	case *ast.ValueSpec:

		ts := gv.Parser.parseTypeSymbol(t.Type)

		if ts == nil && t.Values == nil {
			ts = iotaType
		}

		for i, n := range t.Names {

			fmt.Printf("VARIABLE %s\n", n.Name)

			var exprT st.ITypeSymbol = nil

			if t.Values != nil {
				exprT = gv.Parser.parseExpr(t.Values[i]).At(0).(st.ITypeSymbol)
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

			toAdd := &st.VariableSymbol{Obj: n.Obj, VariableType: ts, Posits: new(vector.Vector)}
			toAdd.AddPosition(st.NewOccurence(n.Pos()))
			gv.Parser.RootSymbolTable.AddSymbol(toAdd)
		}

	case *ast.GenDecl:

		if len(t.Specs) > 0 {
			if vs, ok := t.Specs[0].(*ast.ValueSpec); ok {

				st.RegisterPositions = false
				ts := gv.Parser.parseTypeSymbol(vs.Type)
				st.RegisterPositions = true

				if ts == nil {
					ts, _ = st.PredeclaredTypes["int"]
				}
				iotaType = ts
			}
		}

	case *ast.ForStmt, *ast.FuncDecl, *ast.FuncLit, *ast.IfStmt, *ast.RangeStmt, *ast.SelectStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
		//InnerScope, omitted
		return nil
	}
	return gv
}
