package packageParser

import (
	"go/token"
	"go/ast"
	"container/vector"
	"fmt"
	"st"
)

var isTypeNameUsed bool = false
var (
	fieldsSymbolTable *st.SymbolTable = nil
	searchInFields    bool            = false
)

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

					resT = &st.PointerTypeSymbol{&st.TypeSymbol{Obj: tt.Object(), Meths: nil, Posits: new(vector.Vector)}, tt, nil}
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
			}
		}
	}

	x, cyc := st.GetBaseType(pp.parseExpr(e.Fun).At(0).(st.ITypeSymbol))
	if cyc {
		fmt.Println("ERROR: cycle wasn't expected. eParseCallExpr, parseExpr.go")
	}
	switch x.(type) {
	case *st.FunctionTypeSymbol:
		ft := x.(*st.FunctionTypeSymbol)
		if ft.Results != nil { //Correct order is not guarantied
			for _, v := range ft.Results.Table {
				res.Insert(0, v.(*st.VariableSymbol).VariableType)
			}
		}
	}
	return
}
func (pp *packageParser) eParseCompositeLit(e *ast.CompositeLit) (res *vector.Vector) {
	res = new(vector.Vector)
	clType := pp.parseExpr(e.Type).At(0).(st.ITypeSymbol)
	realClType, cyc := st.GetBaseType(clType)
	if cyc {
		fmt.Println("ERROR: cycle wasn't expected. eParseCompositeLit, parseExpr.go")
	}
	if str, ok := realClType.(*st.StructTypeSymbol); ok {
		searchInFields = true
		fieldsSymbolTable = str.Fields
		defer func() { searchInFields = false; fieldsSymbolTable = nil }()
	}
	var max int = 0
	for _, elt := range e.Elts {
		v := pp.parseExpr(elt)
		if v.Len() > 0 {
			if max < v.At(0).(int) {
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
	fmt.Printf("Parse Ident %v\n", e.Name)

	var lookupST *st.SymbolTable
	if searchInFields {
		lookupST = fieldsSymbolTable
	} else {
		lookupST = pp.CurrentSymbolTable
	}

	if t, found := lookupST.LookUp(e.Name,pp.CurrentFileName); found {
		switch v := t.(type) {
		case *st.VariableSymbol:
			e.Obj = v.Obj
			v.AddPosition(st.NewOccurence(e.Pos()))
			fmt.Printf("AAAA %T\n")
			res.Push(v.VariableType)

		case *st.FunctionSymbol:
			e.Obj = v.Obj
			v.AddPosition(st.NewOccurence(e.Pos()))
			res.Push(v.FunctionType)
		default:
			e.Obj = v.Object()
			v.AddPosition(st.NewOccurence(e.Pos()))
			res.Push(v)
		}
	} else {
		//sould be resolved later
		res.Push(&st.UnresolvedTypeSymbol{&st.TypeSymbol{Obj: e.Obj, Posits: new(vector.Vector)}, e})
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
	}
	return
}
func (pp *packageParser) eParseKeyValueExpr(e *ast.KeyValueExpr) (res *vector.Vector) {

	res = new(vector.Vector)
	//return array index to count it's length
	if l, ok := e.Key.(*ast.BasicLit); ok {
		if l.Kind == token.INT || l.Kind == token.CHAR {
			res.Push(getIntValue(l))
		}
	}

	pp.parseExpr(e.Key) // struct fields or array indexes
	pp.parseExpr(e.Value)
	return
}
func (pp *packageParser) eParseSelectorExpr(e *ast.SelectorExpr) (res *vector.Vector) {
	res = new(vector.Vector)

	t := pp.parseExpr(e.X).At(0).(st.ITypeSymbol)
	if t.Methods() != nil {
		if f, ok := t.Methods().LookUp(e.Sel.Name,pp.CurrentFileName); ok {
			ff := f.(*st.FunctionSymbol)
			e.Sel.Obj = ff.Object()

			ff.AddPosition(st.NewOccurence(e.Sel.Pos()))

			if isTypeNameUsed {
				ff = pp.makeMethodExpression(ff)
			}
			res.Push(ff.FunctionType)
			return
		}
	}

	x, cyc := st.GetBaseType(t)

	if cyc {
		fmt.Println("ERROR: cycle wasn't expected. eParseSelectorExpr, parseExpr.go")
	}
	var v st.Symbol = nil

	switch s := x.(type) {

	case *st.StructTypeSymbol:
		v, _ = s.Fields.LookUp(e.Sel.Name,pp.CurrentFileName)
	case *st.PointerTypeSymbol:
		v, _ = s.Fields.LookUp(e.Sel.Name,pp.CurrentFileName)
		// add PackageSymbol case here
	}
	if v != nil {
		va := v.(*st.VariableSymbol)
		e.Sel.Obj = va.Obj

		va.AddPosition(st.NewOccurence(e.Sel.Pos()))
		res.Push(v.(*st.VariableSymbol).VariableType)
		return
	}

	//Sould be resolved
	res.Push(&st.UnresolvedTypeSymbol{&st.TypeSymbol{Obj: e.Sel.Obj, Posits: new(vector.Vector)}, e})
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

	if base.Name() == "" {
		res.Push(&st.PointerTypeSymbol{BaseType: base})
		return
	}

	pd := 0
	if p, ok := base.(*st.PointerTypeSymbol); ok {
		pd = p.Depth()
	}
	if result, found := pp.CurrentSymbolTable.LookUpPointerType(base.Name(), pd+1); found {
		res.Push(result)
		return
	}

	result := &st.PointerTypeSymbol{&st.TypeSymbol{Obj: base.Object(), Meths: nil, Posits: new(vector.Vector)}, base, nil}
	res.Push(result)

	//Solve where to place new pointer
	if _, found := pp.RootSymbolTable.LookUp(base.Name(),pp.CurrentFileName); found {
		pp.RootSymbolTable.AddSymbol(result)
	} else {
		pp.CurrentSymbolTable.AddSymbol(result)
	}

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

		if base.Name() == "" {
			res.Push(&st.PointerTypeSymbol{BaseType: base})
			return
		}

		pd := 0
		if p, ok := base.(*st.PointerTypeSymbol); ok {
			pd = p.Depth()
		}
		if result, found := pp.CurrentSymbolTable.LookUpPointerType(base.Name(), pd+1); found {
			res.Push(result)
			return
		}

		result := &st.PointerTypeSymbol{&st.TypeSymbol{Obj: base.Object(), Meths: nil, Posits: new(vector.Vector)}, base, nil}
		res.Push(result)

		//Solve where to place new pointer
		if _, found := pp.RootSymbolTable.LookUp(base.Name(),pp.CurrentFileName); found {
			pp.RootSymbolTable.AddSymbol(result)
		} else {
			pp.CurrentSymbolTable.AddSymbol(result)
		}
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
	for _, sym := range ft.Reciever.Table {
		newFt.Parameters.AddSymbol(sym)
	}
	if ft.Parameters != nil {
		for _, sym := range ft.Parameters.Table {
			newFt.Parameters.AddSymbol(sym)
		}
	}
	if ft.Results != nil {
		newFt.Results = st.NewSymbolTable(pp.Package)
		for _, sym := range ft.Results.Table {
			newFt.Results.AddSymbol(sym)
		}
	}
	res.FunctionType = newFt
	return
}
