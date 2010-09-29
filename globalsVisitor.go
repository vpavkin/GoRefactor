package symbolTable

import (
	"go/ast"
	"container/vector"
)

//Represents an ast.Visitor, walking along ast.tree and registering all the global variables met
type GlobalsVisitor struct {
	Stb      *SymbolTableBuilder //pointer to parent SymbolTableBuilder
	IotaType ITypeSymbol
}

/*^GlobalsVisitor Methods^^*/

//ast.Visitor.Visit(). Looks for top-level ast.ValueSpec nodes of ast.Tree to register global vars
func (gv GlobalsVisitor) Visit(node interface{}) (w ast.Visitor) {
	w = gv
	switch t := node.(type) {
	case *ast.ValueSpec:
		ts := gv.Stb.BuildTypeSymbol(t.Type)

		if ts == nil {
			ts = gv.IotaType
		}

		for i, n := range t.Names {
			curTs := ts
			if len(t.Values) > i {
				curTs = gv.Stb.ParseExpr(t.Values[i], gv.Stb.RootSymbolTable).At(0).(ITypeSymbol)
			}
			toAdd := &VariableSymbol{Obj: CopyObject(n), VariableType: curTs, Posits: new(vector.Vector)}
			toAdd.Positions().Push(NewOccurence(n.Pos(), toAdd.Obj))
			gv.Stb.RootSymbolTable.AddSymbol(toAdd)
		}
		w = gv
	case *ast.ForStmt, *ast.FuncDecl, *ast.FuncLit, *ast.IfStmt, *ast.RangeStmt, *ast.SelectStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
		//InnerScope, omitted
		w = nil
	case *ast.GenDecl:
		var IotaType ITypeSymbol = nil
		if len(t.Specs) > 0 {
			if vs, ok := t.Specs[0].(*ast.ValueSpec); ok {
				ts := gv.Stb.BuildTypeSymbol(vs.Type)
				if ts == nil {
					ts = &TypeSymbol{Obj: &ast.Object{Name: "int"}, Posits: new(vector.Vector)}
				}
				IotaType = ts
			}
		}
		w = GlobalsVisitor{gv.Stb, IotaType}
	}
	return
}
