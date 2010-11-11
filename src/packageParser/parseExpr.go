package packageParser

import (
	"go/token"
	"go/ast"
	"container/vector"
	"fmt"
	"st"
)

type exprParser struct {
	IsTypeNameUsed    bool
	FieldsSymbolTable *st.SymbolTable
	SearchInFields    bool

	CompositeLiteralElementType st.ITypeSymbol
}

func (pp *packageParser) parseExpr(exp ast.Expr) (res *vector.Vector) {
	res = new(vector.Vector)
	if exp == nil {
		return nil
	}

	switch e := exp.(type) {
	case *ast.BasicLit:
		return pp.eParseBasicLit(e)
	case *ast.BinaryExpr:
		return pp.eParseBinaryExpr(e)
	case *ast.CallExpr:
		return pp.eParseCallExpr(e)
	case *ast.CompositeLit:
		return pp.eParseCompositeLit(e)
	case *ast.FuncLit: //later in locals visitor
		st.RegisterPositions = false
		res.Push(pp.parseExpr(e.Type).At(0))
		st.RegisterPositions = true
	case *ast.Ident:
		return pp.eParseIdent(e)
	case *ast.IndexExpr:
		return pp.eParseIndexExpr(e)
	case *ast.KeyValueExpr:
		return pp.eParseKeyValueExpr(e)
	case *ast.ParenExpr:
		res.AppendVector(pp.parseExpr(e.X))
	case *ast.SelectorExpr:
		return pp.eParseSelectorExpr(e)
	case *ast.SliceExpr:
		return pp.eParseSliceExpr(e)
	case *ast.StarExpr:
		return pp.eParseStarExpr(e)
	case *ast.TypeAssertExpr:
		return pp.eParseTypeAssertExpr(e)
	case *ast.UnaryExpr:
		return pp.eParseUnaryExpr(e)
	case *ast.ArrayType, *ast.StructType, *ast.FuncType, *ast.InterfaceType, *ast.MapType, *ast.ChanType:
		//type conversion, composite lits
		t := pp.parseTypeSymbol(exp)
		res.Push(t)
	}
	return
}

func (pp *packageParser) eParseBasicLit(e *ast.BasicLit) (res *vector.Vector) {
	res = new(vector.Vector)
	switch e.Kind {
	case token.INT:
		res.Push(st.PredeclaredTypes["int"])
	case token.FLOAT:
		res.Push(st.PredeclaredTypes["float"])
	case token.CHAR:
		res.Push(st.PredeclaredTypes["byte"])
	case token.STRING:
		res.Push(st.PredeclaredTypes["string"])
	case token.IMAG:
		res.Push(st.PredeclaredTypes["complex"])
	}
	return
}

func (pp *packageParser) eParseBinaryExpr(e *ast.BinaryExpr) (res *vector.Vector) {
	res = new(vector.Vector)
	xType := pp.parseExpr(e.X).At(0).(st.ITypeSymbol)
	yType := pp.parseExpr(e.Y).At(0).(st.ITypeSymbol)

	switch e.Op {
	case token.ADD, token.SUB, token.MUL, token.QUO, token.REM, token.AND, token.OR, token.XOR, token.SHL, token.SHR, token.AND_NOT, token.LAND, token.LOR:

		x, okx := st.PredeclaredTypes[xType.Name()]

		y, oky := st.PredeclaredTypes[yType.Name()]
		switch {
		case okx && oky:
			res.Push(x)
		case okx:
			res.Push(y)
		case oky:
			res.Push(x)
		default:
			res.Push(x)
		}
	case token.EQL, token.LSS, token.GTR, token.NEQ, token.LEQ, token.GEQ:
		res.Push(st.PredeclaredTypes["bool"])
	case token.ARROW: //ch <- val
		res.Push(st.PredeclaredTypes["bool"])
	}

	return
}

func (pp *packageParser) eParseCallExpr(e *ast.CallExpr) (res *vector.Vector) {
	res = new(vector.Vector)
	for _, par := range e.Args {
		pp.parseExpr(par)
	}

	if f, ok := e.Fun.(*ast.Ident); ok {
		name := f.Name
		if _, ok1 := st.PredeclaredFunctions[name]; ok1 {
			switch name {
			case "new":
				st.RegisterPositions = false
				tt := pp.parseTypeSymbol(e.Args[0])
				st.RegisterPositions = true

				var resT st.ITypeSymbol
				// LookUp or create a pointer type
				// check if it's possible to call new(*T)
				pd, found := 0, false
				if p, ok := tt.(*st.PointerTypeSymbol); ok {
					pd = p.Depth()
				}

				if resT, found = pp.RootSymbolTable.LookUpPointerType(tt.Name(), pd+1); !found {

					resT = &st.PointerTypeSymbol{&st.TypeSymbol{Obj: tt.Object(), Meths: nil, Posits: new(vector.Vector), PackFrom: pp.Package}, tt, nil}
					pp.RootSymbolTable.AddSymbol(resT)
				}

				res.Push(resT)
				return
			case "make":
				st.RegisterPositions = false
				tt := pp.parseTypeSymbol(e.Args[0])
				st.RegisterPositions = true
				res.Push(tt)
				return
			case "real", "imag":

				st.RegisterPositions = false
				tt := pp.parseTypeSymbol(e.Args[0])
				st.RegisterPositions = true

				switch tt.Name() {
				case "cmplx":
					res.Push(st.PredeclaredTypes["float"])
				case "cmplx64":
					res.Push(st.PredeclaredTypes["float32"])
				case "cmplx128":
					res.Push(st.PredeclaredTypes["float64"])
				}
				return
			case "cmplx":
				st.RegisterPositions = false
				t1 := pp.parseTypeSymbol(e.Args[0])
				t2 := pp.parseTypeSymbol(e.Args[1])
				st.RegisterPositions = true

				switch {
				case t1.Name() == "float64" || t2.Name() == "float64":
					res.Push(st.PredeclaredTypes["cmplx128"])
				case t1.Name() == "float32" || t2.Name() == "float32":
					res.Push(st.PredeclaredTypes["cmplx64"])
				default:
					res.Push(st.PredeclaredTypes["float"])
				}
				return
			case "append":
				st.RegisterPositions = false
				tt := pp.parseTypeSymbol(e.Args[0])
				st.RegisterPositions = true

				res.Push(tt)

				return
			}
		}
	}

	tt := pp.parseExpr(e.Fun).At(0).(st.ITypeSymbol)
	x, cyc := st.GetBaseType(tt)
	if cyc {
		panic("ERROR: cycle wasn't expected. eParseCallExpr, parseExpr.go")
	}
	switch x.(type) {
	case *st.FunctionTypeSymbol:
		ft := x.(*st.FunctionTypeSymbol)
		if ft.Results != nil {
			for v := range ft.Results.Iter() {
				res.Push(v.(*st.VariableSymbol).VariableType)
			}
		}
	// sould be resolved later
	case *st.UnresolvedTypeSymbol:
		res.Push(x)
	default: // Type conversion,push original type
		res.Push(tt)
	}

	return
}
func (pp *packageParser) eParseCompositeLit(e *ast.CompositeLit) (res *vector.Vector) {
	res = new(vector.Vector)

	var clType st.ITypeSymbol

	if e.Type != nil {
		clType = pp.parseExpr(e.Type).At(0).(st.ITypeSymbol)
		if arr, ok := clType.(*st.ArrayTypeSymbol); ok {
			pp.ExprParser.CompositeLiteralElementType = arr.ElemType
		}
	} else {
		clType = pp.ExprParser.CompositeLiteralElementType
	}

	realClType, cyc := st.GetBaseType(clType)
	if cyc {
		panic("ERROR: cycle wasn't expected. eParseCompositeLit, parseExpr.go")
	}
	if str, ok := realClType.(*st.StructTypeSymbol); ok {
		pp.ExprParser.SearchInFields = true
		pp.ExprParser.FieldsSymbolTable = str.Fields
		defer func() { pp.ExprParser.SearchInFields = false; pp.ExprParser.FieldsSymbolTable = nil }()
	}
	var max int = 0
	if len(e.Elts) > 0 {
		if _, ok := e.Elts[0].(*ast.KeyValueExpr); !ok {
			pp.ExprParser.SearchInFields = false
			pp.ExprParser.FieldsSymbolTable = nil
		}
	}
	for _, elt := range e.Elts {
		v := pp.parseExpr(elt)
		if v.Len() > 0 {
			if i, ok := v.At(0).(int); ok && max < i {
				max = v.At(0).(int)
			}
		}
	}

	// get length for [...]array type
	if arr, ok := clType.(*st.ArrayTypeSymbol); ok {
		if arr.Len == st.ELLIPSIS {
			arr.Len = max + 1
		}
	}

	res.Push(clType)
	return
}

func (pp *packageParser) eParseIdent(e *ast.Ident) (res *vector.Vector) {
	res = new(vector.Vector)
	fmt.Printf("%s:	Parse Ident %v\n", pp.Package.AstPackage.Name, e.Name)

	var lookupST *st.SymbolTable
	if pp.ExprParser.SearchInFields {
		lookupST = pp.ExprParser.FieldsSymbolTable
	} else {
		lookupST = pp.CurrentSymbolTable
	}

	if t, found := lookupST.LookUp(e.Name, pp.CurrentFileName); found {
		switch v := t.(type) {
		case *st.VariableSymbol:
			e.Obj = v.Obj
			v.AddPosition(st.NewOccurence(e.Pos()))
			res.Push(v.VariableType)

		case *st.FunctionSymbol:
			e.Obj = v.Obj
			v.AddPosition(st.NewOccurence(e.Pos()))
			res.Push(v.FunctionType)

		default: //PackageSymbol or type
			if _, ok := t.(*st.PackageSymbol); !ok {
				fmt.Printf("%s:	<><><><>  %v - %T \n", pp.Package.AstPackage.Name, t.Name(), t)
				pp.ExprParser.IsTypeNameUsed = true
			}
			e.Obj = v.Object()
			v.AddPosition(st.NewOccurence(e.Pos()))
			res.Push(v)
		}
	} else {
		//sould be resolved later
		fmt.Printf("%s:	WARNING! Ident %v wasn't found\n", pp.Package.AstPackage.Name, e.Name)
		res.Push(&st.UnresolvedTypeSymbol{&st.TypeSymbol{Obj: e.Obj, Posits: new(vector.Vector), PackFrom: nil}, e})
	}

	return
}
func (pp *packageParser) eParseIndexExpr(e *ast.IndexExpr) (res *vector.Vector) {

	res = new(vector.Vector)
	x, cyc := st.GetBaseType(pp.parseExpr(e.X).At(0).(st.ITypeSymbol))
	if cyc {
		fmt.Println("ERROR: cycle wasn't expected. eParseIndexExpr, parseExpr.go")
	}
	//Sould be resolved later
	/*if ContainsUnresolvedTypeSymbol(x) {
	res.Push(x)
	return
	}*/
	switch s := x.(type) {
	case *st.ArrayTypeSymbol:
		res.Push(s.ElemType)
	case *st.MapTypeSymbol:
		res.Push(s.ValueType)
		//if RightHand
		res.Push(st.PredeclaredTypes["bool"])
	case *st.BasicTypeSymbol: //string
		res.Push(st.PredeclaredTypes["byte"])

	case *st.UnresolvedTypeSymbol: //string
		res.Push(s)
	}
	return
}
func (pp *packageParser) eParseKeyValueExpr(e *ast.KeyValueExpr) (res *vector.Vector) {

	pp.parseExpr(e.Key) // struct fields or array indexes
	pp.parseExpr(e.Value)

	res = new(vector.Vector)
	//return array index to count it's length
	if l, ok := e.Key.(*ast.BasicLit); ok {
		if l.Kind == token.INT || l.Kind == token.CHAR {
			res.Push(getIntValue(l))
		}
	}
	return
}
func (pp *packageParser) eParseSelectorExpr(e *ast.SelectorExpr) (res *vector.Vector) {
	res = new(vector.Vector)

	pp.ExprParser.IsTypeNameUsed = false

	t := pp.parseExpr(e.X).At(0).(st.ITypeSymbol)

	// find method
	var toSearch *st.SymbolTable
	if s, ok := t.(*st.PackageSymbol); ok {
		toSearch = s.Package.Symbols
	} else if t.Methods() != nil {
		toSearch = t.Methods()
	}
	if toSearch != nil {
		if ff, ok := toSearch.LookUp(e.Sel.Name, ""); ok {
			if f, ok := ff.(*st.FunctionSymbol); ok {

				e.Sel.Obj = f.Object()

				f.AddPosition(st.NewOccurence(e.Sel.Pos()))

				if pp.ExprParser.IsTypeNameUsed {
					pp.ExprParser.IsTypeNameUsed = false
					f = pp.makeMethodExpression(f)
				}
				res.Push(f.FunctionType)
				return
			}
		}
	}

	// field
	x, cyc := st.GetBaseType(t)

	if cyc {
		panic("ERROR: cycle wasn't expected. eParseSelectorExpr, parseExpr.go")
	}

	switch s := x.(type) {

	case *st.StructTypeSymbol:
		toSearch = s.Fields
	case *st.PointerTypeSymbol:
		toSearch = s.Fields
	case *st.PackageSymbol:
		toSearch = s.Package.Symbols
	}
	if vv, ok := toSearch.LookUp(e.Sel.Name, ""); ok {
		if va, ok := vv.(*st.VariableSymbol); ok {
			e.Sel.Obj = va.Obj
			va.AddPosition(st.NewOccurence(e.Sel.Pos()))
			res.Push(va.VariableType)
			return
		}
	}

	//type from other packageParser
	if s, ok := t.(*st.PackageSymbol); ok {
		if tt, ok := s.Package.Symbols.LookUp(e.Sel.Name, ""); ok {
			e.Sel.Obj = tt.Object()
			tt.AddPosition(st.NewOccurence(e.Sel.Pos()))
			res.Push(tt)
			pp.ExprParser.IsTypeNameUsed = true
			return
		}
	}

	//Sould be resolved
	res.Push(&st.UnresolvedTypeSymbol{&st.TypeSymbol{Obj: e.Sel.Obj, Posits: new(vector.Vector), PackFrom: nil}, e})
	return
}
func (pp *packageParser) eParseSliceExpr(e *ast.SliceExpr) (res *vector.Vector) {
	res = new(vector.Vector)

	x := pp.parseExpr(e.X).At(0).(st.ITypeSymbol)
	pp.parseExpr(e.Index)
	pp.parseExpr(e.End)
	r := x
	// slicing an array results a slice
	if arr, ok := x.(*st.ArrayTypeSymbol); ok {
		if arr.Len != st.SLICE {
			sl := arr.Copy().(*st.ArrayTypeSymbol)
			sl.Len = st.SLICE
			r = sl
		}
	}
	res.Push(r)
	return
}

func (pp *packageParser) eParseStarExpr(e *ast.StarExpr) (res *vector.Vector) {
	res = new(vector.Vector)
	base := pp.parseExpr(e.X).At(0).(st.ITypeSymbol)
	res.Push(pp.getOrAddPointer(base))
	return
}
func (pp *packageParser) eParseTypeAssertExpr(e *ast.TypeAssertExpr) (res *vector.Vector) {
	res = new(vector.Vector)
	pp.parseExpr(e.X) //pos
	t := pp.parseExpr(e.Type)
	res.Push(t.At(0))
	res.Push(st.PredeclaredTypes["bool"])
	return
}
func (pp *packageParser) eParseUnaryExpr(e *ast.UnaryExpr) (res *vector.Vector) {
	res = new(vector.Vector)
	t := pp.parseExpr(e.X).At(0).(st.ITypeSymbol)
	switch e.Op {
	case token.ADD, token.SUB, token.XOR, token.NOT:
		res.Push(t)
	case token.ARROW:
		chh, cyc := st.GetBaseType(t)
		if cyc {
			fmt.Println("ERROR: cycle wasn't expected. eParseUnaryExpr, parseExpr.go")
		}
		ch := chh.(*st.ChanTypeSymbol)
		res.Push(ch.ValueType)
		res.Push(st.PredeclaredTypes["bool"])
	case token.AND:
		base := t
		res.Push(pp.getOrAddPointer(base))
	}
	return
}


func (pp *packageParser) makeMethodExpression(fs *st.FunctionSymbol) (res *st.FunctionSymbol) {
	res = fs.Copy()
	fft, cyc := st.GetBaseType(fs.FunctionType)
	if cyc {
		fmt.Println("ERROR: cycle wasn't expected. eParseUnaryExpr, parseExpr.go")
	}
	ft := fft.(*st.FunctionTypeSymbol)

	newFt := &st.FunctionTypeSymbol{Parameters: st.NewSymbolTable(pp.Package)}

	if ft.Reciever == nil {
		fmt.Println("ERROR : f.Reciever == nil. makeMethodExpression,parseExpr.go")
	}
	for sym := range ft.Reciever.Iter() {
		newFt.Parameters.AddSymbol(sym)
	}
	if ft.Parameters != nil {
		for sym := range ft.Parameters.Iter() {
			newFt.Parameters.AddSymbol(sym)
		}
	}
	if ft.Results != nil {
		newFt.Results = st.NewSymbolTable(pp.Package)
		for sym := range ft.Results.Iter() {
			newFt.Results.AddSymbol(sym)
		}
	}
	res.FunctionType = newFt
	return
}
