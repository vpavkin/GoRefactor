package symbolTable

import (
	"go/token"
	"go/ast"
	"container/vector"
	"fmt"
)

var IntegerTypes map[string]bool = map[string]bool{"uintptr": true, "byte": true, "int8": true, "int16": true, "int32": true, "int64": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true, "int": true, "uint": true}
var FloatTypes map[string]bool = map[string]bool{"float32": true, "float64": true, "float": true}
var ComplexTypes map[string]bool = map[string]bool{"complex64": true, "complex128": true, "complex": true}
var BuiltInFunctions map[string]bool = map[string]bool{"len": true, "cap": true, "new": true, "make": true, "copy": true, "cmplx": true, "real": true, "imag": true, "panic": true, "recover": true, "print": true, "println": true}

func IsBuiltInFunction(name string) (r bool) {
	_, r = BuiltInFunctions[name]
	return
}
func IsIntegerType(name string) (r bool) {
	_, r = IntegerTypes[name]
	return
}
func IsFloatType(name string) (r bool) {
	_, r = FloatTypes[name]
	return
}
func IsComplexType(name string) (r bool) {
	_, r = ComplexTypes[name]
	return
}

//Expression parse module
func (stb *SymbolTableBuilder) ParseExpr(exp ast.Expr, curSt *SymbolTable) (res *vector.Vector) {
	if exp == nil {
		return nil
	}

	res = &vector.Vector{}
	switch e := exp.(type) {
	case *ast.BasicLit:
		switch e.Kind {
		case token.INT:
			res.Push(&TypeSymbol{Obj: &ast.Object{Name: "int"}, Meths: nil, Posits: new(vector.Vector)})
		case token.FLOAT:
			res.Push(&TypeSymbol{Obj: &ast.Object{Name: "float"}, Meths: nil, Posits: new(vector.Vector)})
		case token.CHAR:
			res.Push(&TypeSymbol{Obj: &ast.Object{Name: "char"}, Meths: nil, Posits: new(vector.Vector)})
		case token.STRING:
			res.Push(&TypeSymbol{Obj: &ast.Object{Name: "string"}, Meths: nil, Posits: new(vector.Vector)})
		}
	case *ast.BinaryExpr: //Imported + Imported ????
		xType := stb.ParseExpr(e.X, curSt).At(0).(ITypeSymbol)
		yType := stb.ParseExpr(e.Y, curSt).At(0).(ITypeSymbol)

		switch e.Op {
		case token.ADD, token.SUB, token.MUL, token.QUO, token.REM:
			switch {
			case xType.Object().Name == "string" || yType.Object().Name == "string":
				res.Push(&TypeSymbol{Obj: &ast.Object{Name: "string"}, Meths: nil, Posits: new(vector.Vector)})
				return
			case IsComplexType(xType.Object().Name) || IsComplexType(yType.Object().Name):
				res.Push(&TypeSymbol{Obj: &ast.Object{Name: "complex"}, Meths: nil, Posits: new(vector.Vector)})
				return
			case IsFloatType(xType.Object().Name) || IsFloatType(yType.Object().Name):
				res.Push(&TypeSymbol{Obj: &ast.Object{Name: "float"}, Meths: nil, Posits: new(vector.Vector)})
				return
			default: // int/int
				if e.Op == token.QUO {
					res.Push(&TypeSymbol{Obj: &ast.Object{Name: "float"}, Meths: nil, Posits: new(vector.Vector)})
				} else {
					res.Push(&TypeSymbol{Obj: &ast.Object{Name: "int"}, Meths: nil, Posits: new(vector.Vector)})
				}
			}
		case token.AND, token.OR, token.XOR, token.SHL, token.SHR, token.AND_NOT:
			res.Push(&TypeSymbol{Obj: &ast.Object{Name: "int"}, Meths: nil, Posits: new(vector.Vector)})
		case token.LAND, token.LOR, token.EQL, token.LSS, token.GTR, token.NEQ, token.LEQ, token.GEQ:
			res.Push(&TypeSymbol{Obj: &ast.Object{Name: "bool"}, Meths: nil, Posits: new(vector.Vector)})
		case token.ARROW: //ch <- val
			res.Push(&TypeSymbol{Obj: &ast.Object{Name: "bool"}, Meths: nil, Posits: new(vector.Vector)})
		}
		return
	case *ast.CallExpr:
		for _, par := range e.Args {
			if par != nil {
				stb.ParseExpr(par, curSt)
			}
		}

		if f, ok := e.Fun.(*ast.Ident); ok {
			name := f.Name()
			if IsBuiltInFunction(name) {
				switch name {
				case "len", "cap", "copy":
					res.Push(&TypeSymbol{Obj: &ast.Object{Name: "int"}, Meths: nil, Posits: new(vector.Vector)})
				case "new":
					stb.PositionsRegister = false
					tt := stb.BuildTypeSymbol(e.Args[0])
					t, _ := GetOrAddPointer(tt, curSt, stb.RootSymbolTable)
					stb.PositionsRegister = true
					res.Push(t)
				case "make":
					stb.PositionsRegister = false
					tt := stb.BuildTypeSymbol(e.Args[0])
					stb.PositionsRegister = true
					res.Push(tt)
				case "real", "imag":
					res.Push(&TypeSymbol{Obj: &ast.Object{Name: "float"}, Meths: nil, Posits: new(vector.Vector)})
				case "cmplx":
					res.Push(&TypeSymbol{Obj: &ast.Object{Name: "complex"}, Meths: nil, Posits: new(vector.Vector)})
				case "recover":
					res.Push(&InterfaceTypeSymbol{&TypeSymbol{Meths: NewSymbolTable(), Posits: new(vector.Vector)}})
				}
				return
			}
		}

		x := stb.ParseExpr(e.Fun, curSt).At(0).(ITypeSymbol)
		switch x.(type) {
		case *FunctionTypeSymbol:
			ft := x.(*FunctionTypeSymbol)
			for _, v := range ft.Results.Table {
				res.Insert(0, v.(*VariableSymbol).VariableType)
			}
		case *ImportedType:
			res.Push(x)
		default:
			res.Push(x)
		}
	case *ast.CompositeLit:
		clType := stb.ParseExpr(e.Type, curSt).At(0).(ITypeSymbol)
		realClType := GetBaseType(clType)
		toSearchKeys := curSt
		if str, ok := realClType.(*StructTypeSymbol); ok {
			toSearchKeys = str.Fields
		}
		for _, elt := range e.Elts {
			if _, ok := elt.(*ast.KeyValueExpr); ok {
				stb.ParseExpr(elt, toSearchKeys)
			} else {
				stb.ParseExpr(elt, curSt)
			}
		}
		res.Push(clType)
	case *ast.FuncLit:
		stb.PositionsRegister = false
		res.Push(stb.ParseExpr(e.Type, curSt).At(0))
		stb.PositionsRegister = true

	case *ast.Ident:
		fmt.Printf("IIIIIIIIII %v %p\n", e.Name(), e)
		if IsPredeclaredType(e.Name()) {
			res.Push(&TypeSymbol{Obj: e.Obj, Posits: new(vector.Vector)})
			return
		}
		if e.Name() == "false" || e.Name() == "true" {
			res.Push(&TypeSymbol{Obj: &ast.Object{Name: "bool"}, Meths: nil, Posits: new(vector.Vector)})
			return
		}
		if e.Name() == "nil" {
			res.Push(&InterfaceTypeSymbol{&TypeSymbol{Obj: nil, Meths: nil, Posits: new(vector.Vector)}})
			return
		}
		if m, found := curSt.FindSymbolByName(e.Name()); found {
			switch v := m.(type) {
			case *VariableSymbol:
				e.Obj = v.Obj
				v.Posits.Push(NewOccurence(e.Pos(), e.Obj))
				res.Push(v.VariableType)
			case *FunctionSymbol:
				e.Obj = v.Obj
				v.Posits.Push(NewOccurence(e.Pos(), e.Obj))
				res.Push(v.FunctionType)
			default:
				v.Positions().Push(NewOccurence(e.Pos(), e.Obj))
				res.Push(m)
			}
		} else {
			res.Push(&ImportedType{&TypeSymbol{Obj: e.Obj, Posits: new(vector.Vector)}})
		}
	case *ast.IndexExpr:
		x := stb.ParseExpr(e.X, curSt).At(0).(ITypeSymbol)
		if ContainsImportedType(x) {
			res.Push(x)
			return
		}
		switch s := x.(type) {
		case *ArrayTypeSymbol:
			res.Push(s.ElemType)
		case *MapTypeSymbol:
			res.Push(s.ValueType)
			//if RightHand
			res.Push(&TypeSymbol{Obj: &ast.Object{Name: "bool"}, Meths: nil, Posits: new(vector.Vector)})
		case *TypeSymbol: //string
			res.Push(&TypeSymbol{Obj: &ast.Object{Name: "char"}, Meths: nil, Posits: new(vector.Vector)})
		}
	case *ast.KeyValueExpr:
		// Pos marking
		stb.ParseExpr(e.Key, curSt) // struct fields
		stb.ParseExpr(e.Value, stb.CurrentSymbolTable)
	case *ast.ParenExpr:
		res.AppendVector(stb.ParseExpr(e.X, curSt))
	case *ast.SelectorExpr:
		x := stb.ParseExpr(e.X, curSt).At(0).(ITypeSymbol)
		if ContainsImportedType(x) {
			res.Push(x)
			return
		}
		switch s := x.(type) {

		case *StructTypeSymbol:
			if f, found := s.Fields.FindSymbolByName(e.Sel.Name()); found {

				v := f.(*VariableSymbol)
				v.Posits.Push(NewOccurence(e.Sel.Pos(), e.Sel.Obj))
				res.Push(f.(*VariableSymbol).VariableType)
				return
			}
		case *PointerTypeSymbol:
			if f, found := s.Fields.FindSymbolByName(e.Sel.Name()); found {

				v := f.(*VariableSymbol)
				v.Posits.Push(NewOccurence(e.Sel.Pos(), e.Sel.Obj))
				res.Push(f.(*VariableSymbol).VariableType)
				return
			}
		}
		if f, ok := x.Methods().FindSymbolByName(e.Sel.Name()); ok {

			ff := f.(*FunctionSymbol)
			ff.Posits.Push(NewOccurence(e.Sel.Pos(), e.Sel.Obj))
			res.Push(ff.FunctionType)
			return
		}

		res.Push(&ImportedType{&TypeSymbol{Obj: e.Sel.Obj, Posits: new(vector.Vector)}})
		return

	case *ast.SliceExpr:
		x := stb.ParseExpr(e.X, curSt).At(0).(ITypeSymbol)
		stb.ParseExpr(e.Index, curSt)
		stb.ParseExpr(e.End, curSt)
		res.Push(x)
	case *ast.StarExpr:
		base := stb.ParseExpr(e.X, curSt).At(0).(ITypeSymbol)
		r, _ := GetOrAddPointer(base, curSt, stb.RootSymbolTable)
		res.Push(r)
	case *ast.TypeAssertExpr:
		stb.ParseExpr(e.X, curSt) //pos
		t := stb.ParseExpr(e.Type, curSt)
		res.Push(t.At(0))
		res.Push(&TypeSymbol{Obj: &ast.Object{Name: "bool"}, Meths: nil, Posits: new(vector.Vector)})
	case *ast.UnaryExpr:
		t := stb.ParseExpr(e.X, curSt).At(0).(ITypeSymbol)
		switch e.Op {
		case token.ADD, token.SUB, token.XOR, token.NOT:
			res.Push(t)
		case token.ARROW:
			if _, ok := t.(*ImportedType); ok {
				res.Push(t)
			} else {
				ch := t.(*ChanTypeSymbol)
				res.Push(ch.ValueType)
			}
			res.Push(&TypeSymbol{Obj: &ast.Object{Name: "bool"}, Meths: nil, Posits: new(vector.Vector)})
		case token.AND:
			base := t
			r, _ := GetOrAddPointer(base, curSt, stb.RootSymbolTable)
			res.Push(r)
		}
	case *ast.ArrayType, *ast.StructType, *ast.FuncType, *ast.InterfaceType, *ast.MapType, *ast.ChanType:
		//type conversion
		t := stb.BuildTypeSymbol(exp)
		t.SetObject(&ast.Object{Name: t.String()})
		res.Push(t)

	}
	return
}
