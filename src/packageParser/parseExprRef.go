package packageParser

import (
	"go/ast"
	"container/vector"
	"st"
)

func (pp *packageParser) eParseIdentRef(e *ast.Ident) (res *vector.Vector) {
	res = new(vector.Vector)

	sym := pp.IdentMap.GetSymbol(e)

	switch v := sym.(type) {
	case *st.VariableSymbol:
		res.Push(v.VariableType)
	case *st.FunctionSymbol:
		res.Push(v.FunctionType)
	case *st.UnresolvedTypeSymbol:
		panic("Unresolved symbol found in refactoring mode")
	default: //PackageSymbol or type
		res.Push(v)
	}
	return
}
