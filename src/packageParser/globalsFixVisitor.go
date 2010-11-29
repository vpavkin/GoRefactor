package packageParser

import (
	"go/ast"
	"st"
)
import "fmt"

type globalsFixVisitor struct {
	Parser *packageParser
}

//ast.Visitor.Visit(). Looks for top-level ast.ValueSpec nodes of ast.Tree to register global vars
func (gv globalsFixVisitor) Visit(node interface{}) ast.Visitor {

	switch t := node.(type) {
	case *ast.ValueSpec:

		gv.Parser.parseTypeSymbol(t.Type)

		for i, n := range t.Names {
			var v *st.VariableSymbol
			if sym, ok := gv.Parser.Package.Symbols.LookUp(n.Name, gv.Parser.CurrentFileName); !ok {
				panic("Lost variable or something...  " + n.Name)
			} else {
				v = sym.(*st.VariableSymbol)
			}

			var exprT st.ITypeSymbol = nil

			if t.Values != nil {
				exprT = gv.Parser.parseExpr(t.Values[i]).At(0).(st.ITypeSymbol)
				if _, ok := v.VariableType.(*st.UnresolvedTypeSymbol); ok {
					if _, ok := exprT.(*st.UnresolvedTypeSymbol); !ok {
						fmt.Printf("%s: =)))) var %s found its type type\n", gv.Parser.Package.AstPackage.Name, v.Name())
					}
					v.VariableType = exprT
				}
			}
			if _, ok := exprT.(*st.UnresolvedTypeSymbol); ok {
				fmt.Printf("%s: @!@!@!@ var %s still has unresolved type\n", gv.Parser.Package.AstPackage.Name, v.Name())
			}

			if arrT, ok := v.VariableType.(*st.ArrayTypeSymbol); ok {
				if arrT.Len == st.ELLIPSIS {
					arrT.Len = exprT.(*st.ArrayTypeSymbol).Len
				}
			}
			v.AddPosition(n.Pos())
		}
	case *ast.ForStmt, *ast.FuncDecl, *ast.FuncLit, *ast.IfStmt, *ast.RangeStmt, *ast.SelectStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
		//InnerScope, omitted
		return nil
	}
	return gv
}
