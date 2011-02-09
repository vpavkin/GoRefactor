package st

import (
	"go/ast"
	"go/token"
	"container/vector"
	"strings"
	"strconv"
)

func makeNamedExpr(s Symbol, pack *Package, filename string) ast.Expr {
	if s.PackageFrom() != pack {
		imp := pack.GetImport(filename, s.PackageFrom())
		prefix := imp.Name()
		return &ast.SelectorExpr{ast.NewIdent(prefix), ast.NewIdent(s.Name())}
	}
	return ast.NewIdent(s.Name())
}
func ArrayLenToAstExpr(Len int) ast.Expr {
	switch Len {
	case SLICE:
		return nil
	case ELLIPSIS:
		return &ast.Ellipsis{token.NoPos, nil}
	}
	return &ast.BasicLit{token.NoPos, token.INT, []uint8(strconv.Itoa(Len))}
}
func (s *UnresolvedTypeSymbol) ToAstExpr(pack *Package, filename string) ast.Expr {
	panic("can't make an expression of unresolved type")
}

func (s *BasicTypeSymbol) ToAstExpr(pack *Package, filename string) ast.Expr {

	return ast.NewIdent(s.Name())
}

func (s *AliasTypeSymbol) ToAstExpr(pack *Package, filename string) ast.Expr {
	return makeNamedExpr(s, pack, filename)
}
func (s *ArrayTypeSymbol) ToAstExpr(pack *Package, filename string) ast.Expr {
	if s.Name() != NO_NAME {
		return makeNamedExpr(s, pack, filename)
	}
	return &ast.ArrayType{token.NoPos, ArrayLenToAstExpr(s.Len), s.ElemType.ToAstExpr(pack, filename)}
}
func (s *ChanTypeSymbol) ToAstExpr(pack *Package, filename string) ast.Expr {
	if s.Name() != NO_NAME {
		return makeNamedExpr(s, pack, filename)
	}
	return &ast.ChanType{token.NoPos, s.Dir, s.ValueType.ToAstExpr(pack, filename)}
}
func (s *FunctionTypeSymbol) ToAstExpr(pack *Package, filename string) ast.Expr {
	if s.Name() != NO_NAME {
		return makeNamedExpr(s, pack, filename)
	}
	res := &ast.FuncType{token.NoPos, nil, nil}
	res.Params = s.Parameters.ToAstFieldList(pack, filename)
	if s.Results != nil {
		res.Results = s.Results.ToAstFieldList(pack, filename)
	}
	return res
}

func (s *InterfaceTypeSymbol) ToAstExpr(pack *Package, filename string) ast.Expr {
	if s.Name() != NO_NAME {
		return makeNamedExpr(s, pack, filename)
	}
	return &ast.InterfaceType{token.NoPos, nil, false}
}

func (s *MapTypeSymbol) ToAstExpr(pack *Package, filename string) ast.Expr {
	if s.Name() != NO_NAME {
		return makeNamedExpr(s, pack, filename)
	}
	return &ast.MapType{token.NoPos, s.KeyType.ToAstExpr(pack, filename), s.ValueType.ToAstExpr(pack, filename)}
}
func (s *PointerTypeSymbol) ToAstExpr(pack *Package, filename string) ast.Expr {

	return &ast.StarExpr{token.NoPos, s.BaseType.ToAstExpr(pack, filename)}
}

func (s *StructTypeSymbol) ToAstExpr(pack *Package, filename string) ast.Expr {
	if s.Name() != NO_NAME {
		return makeNamedExpr(s, pack, filename)
	}
	res := &ast.StructType{token.NoPos, nil, false}
	if s.Fields != nil {
		res.Fields = s.Fields.ToAstFieldList(pack, filename)
	}
	return res
}

func (ps *PackageSymbol) ToAstExpr(pack *Package, filename string) ast.Expr {
	panic("mustn't call ITypeSymbol methods on PackageSymbol")
}

//VariableSymbol & FunctionSymbol

func (s *VariableSymbol) ToAstField(pack *Package, filename string) *ast.Field {
	if s.Name() == NO_NAME || strings.HasPrefix(s.Name(), UNNAMED_PREFIX) {
		return &ast.Field{nil, nil, s.VariableType.ToAstExpr(pack, filename), nil, nil}
	}
	return &ast.Field{nil, []*ast.Ident{ast.NewIdent(s.Name())}, s.VariableType.ToAstExpr(pack, filename), nil, nil}
}

func (s *FunctionSymbol) ToAstField(pack *Package, filename string) *ast.Field {
	if s.Name() == NO_NAME {
		return &ast.Field{nil, nil, s.FunctionType.ToAstExpr(pack, filename), nil, nil}
	}
	return &ast.Field{nil, []*ast.Ident{ast.NewIdent(s.Name())}, s.FunctionType.ToAstExpr(pack, filename), nil, nil}
}

func (s *VariableSymbol) ToAstExpr(pack *Package, filename string) ast.Expr {
	if s.Name() != NO_NAME {
		return makeNamedExpr(s, pack, filename)
	}
	panic("can't make an expr out of unnamed variable/function")
}

func (s *FunctionSymbol) ToAstExpr(pack *Package, filename string) ast.Expr {
	if s.Name() != NO_NAME {
		return makeNamedExpr(s, pack, filename)
	}
	panic("can't make an expr out of unnamed variable/function")
}

// Symbol Table



func (s *SymbolTable) ToAstFieldList(pack *Package, filename string) *ast.FieldList {
	vect := new(vector.Vector)
	s.ForEachNoLock(func(sym Symbol) {
		switch t := sym.(type) {
		case *VariableSymbol:
			vect.Push(t.ToAstField(pack, filename))
		case *FunctionSymbol:
			vect.Push(t.ToAstField(pack, filename))
		default:
			panic("can't convert symbol table with non-variable symbols to ast.FieldList")
		}
	})

	list := make([]*ast.Field, len(*vect))
	for i, el := range *vect {
		list[i] = el.(*ast.Field)
	}
	return &ast.FieldList{token.NoPos, list, token.NoPos}
}

func (s *SymbolTable) ToAstExprSlice(pack *Package, filename string) []ast.Expr {
	vect := new(vector.Vector)
	s.ForEachNoLock(func(sym Symbol) {
		switch t := sym.(type) {
		case *VariableSymbol:
			vect.Push(t.ToAstExpr(pack, filename))
		case *FunctionSymbol:
			vect.Push(t.ToAstExpr(pack, filename))
		default:
			panic("can't convert symbol table with non-variable symbols to ast.FieldList")
		}

	})

	list := make([]ast.Expr, len(*vect))
	for i, el := range *vect {
		list[i] = el.(ast.Expr)
	}
	return list
}
