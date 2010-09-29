package symbolTable

import (
	"go/ast"
	"go/token"
	"container/vector"
)

type InMethodsVisitor struct {
	Stb *SymbolTableBuilder
}

type LocalsVisitor struct {
	Method   *FunctionSymbol
	Current  *SymbolTable
	Stb      *SymbolTableBuilder
	IotaType ITypeSymbol
}

func (mv InMethodsVisitor) Visit(node interface{}) (w ast.Visitor) {
	w = mv
	switch f := node.(type) {
	case *ast.ValueSpec: //global var/const spec
		for _, e := range f.Values {
			mv.Stb.ParseExpr(e, mv.Stb.RootSymbolTable)
		}
	case *ast.FuncDecl:

		scope := mv.Stb.RootSymbolTable
		if f.Recv != nil {
			for _, field := range f.Recv.List {
				mv.Stb.PositionsRegister = false
				rtype := mv.Stb.BuildTypeSymbol(field.Type)
				mv.Stb.PositionsRegister = true
				scope = rtype.Methods()
			}
		}
		m, _ := scope.FindSymbolByName(f.Name.Name())
		meth := m.(*FunctionSymbol)
		meth.Locals.AddOpenedScope(mv.Stb.RootSymbolTable)

		w = LocalsVisitor{meth, meth.Locals, mv.Stb, nil}
	}
	return
}

func (lv LocalsVisitor) ParseStmt(node interface{}) (w ast.Visitor) {
	w = lv
	temp := lv.Stb.CurrentSymbolTable
	lv.Stb.CurrentSymbolTable = lv.Current

	switch s := node.(type) {

	case *ast.GenDecl:
		var IotaType ITypeSymbol = nil
		if len(s.Specs) > 0 {
			if vs, ok := s.Specs[0].(*ast.ValueSpec); ok {
				ts := lv.Stb.BuildTypeSymbol(vs.Type)
				if ts == nil {
					ts = &TypeSymbol{Obj: &ast.Object{Name: "int"}, Posits: new(vector.Vector)}
				}
				IotaType = ts
			}
		}
		w = LocalsVisitor{lv.Method, lv.Current, lv.Stb, IotaType}

	case *ast.ValueSpec: //Specify a new variable

		ts := lv.Stb.BuildTypeSymbol(s.Type)

		if ts == nil {
			ts = lv.IotaType
		}

		for i, n := range s.Names {
			curTs := ts
			if len(s.Values) > i {
				curTs = lv.Stb.ParseExpr(s.Values[i], lv.Stb.CurrentSymbolTable).At(0).(ITypeSymbol)
			}
			toAdd := &VariableSymbol{Obj: CopyObject(n), VariableType: curTs, Posits: new(vector.Vector)}
			toAdd.Positions().Push(NewOccurence(n.Pos(), toAdd.Obj))
			lv.Stb.CurrentSymbolTable.AddSymbol(toAdd)
		}

	case *ast.AssignStmt:
		switch s.Tok {
		case token.DEFINE: //Specify a new variable
			eTypes := &vector.Vector{}
			for _, e := range s.Rhs {
				res := lv.Stb.ParseExpr(e, lv.Current)
				eTypes.AppendVector(res)
			}
			for i, n := range s.Lhs {
				if n.(*ast.Ident).Name() != "_" {
					if i < eTypes.Len() {
						toAdd := &VariableSymbol{Obj: CopyObject(n.(*ast.Ident)), VariableType: eTypes.At(i).(ITypeSymbol), Posits: new(vector.Vector)}
						toAdd.Posits.Push(NewOccurence(n.Pos(), toAdd.Obj))
						lv.Current.AddSymbol(toAdd)

					} else {
						toAdd := &VariableSymbol{Obj: CopyObject(n.(*ast.Ident)), VariableType: &ImportedType{&TypeSymbol{Obj: &ast.Object{Name: "imported"}, Posits: new(vector.Vector)}}, Posits: new(vector.Vector)}
						toAdd.Posits.Push(NewOccurence(n.Pos(), toAdd.Obj))
						lv.Current.AddSymbol(toAdd)
					}
				}
			}
		case token.ASSIGN: //Pos
			for _, e := range s.Rhs {
				lv.Stb.ParseExpr(e, lv.Current)
			}
			for _, n := range s.Lhs {
				lv.Stb.ParseExpr(n, lv.Current)
			}
		}
	case *ast.TypeSpec: //Specify a new type
		ts := lv.Stb.BuildTypeSymbol(s.Type)

		switch ts.(type) {
		case *ImportedType, *PointerTypeSymbol, *ArrayTypeSymbol, *StructTypeSymbol, *InterfaceTypeSymbol, *MapTypeSymbol, *ChanTypeSymbol, *FunctionTypeSymbol:
			if ts.Object() == nil {
				//No such symbol in CurrentSymbolTable
				ts.SetObject(CopyObject(s.Name))
			} else {
				//There is an equal type symbol with different name => create alias
				ts = &AliasTypeSymbol{&TypeSymbol{Obj: CopyObject(s.Name), Meths: NewSymbolTable(), Posits: new(vector.Vector)}, ts}
			}
		}
		ts.Positions().Push(NewOccurence(s.Name.Pos(), ts.Object()))
		lv.Current.AddSymbol(ts)
	case *ast.ExprStmt:
		lv.Stb.ParseExpr(s.X, lv.Current)
	case *ast.DeferStmt:
		lv.Stb.ParseExpr(s.Call, lv.Current)
	case *ast.GoStmt:
		lv.Stb.ParseExpr(s.Call, lv.Current)
	case *ast.IncDecStmt:
		lv.Stb.ParseExpr(s.X, lv.Current)
	case *ast.ReturnStmt:
		if s.Results != nil { //mb not needed
			for _, exp := range s.Results {
				lv.Stb.ParseExpr(exp, lv.Current)
			}
		}
	case *ast.SwitchStmt, *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.FuncLit, *ast.SelectStmt, *ast.TypeSwitchStmt, *ast.CaseClause, *ast.TypeCaseClause, *ast.CommClause:
		st := NewSymbolTable()
		st.AddOpenedScope(lv.Current)
		ww := LocalsVisitor{lv.Method, st, lv.Stb, nil}
		switch inNode := node.(type) {
		case *ast.ForStmt:
			ww.ParseStmt(inNode.Init)
			ww.Stb.ParseExpr(inNode.Cond, ww.Current)
			ww.ParseStmt(inNode.Post)
			ast.Walk(ww, inNode.Body)
			w = nil
		case *ast.IfStmt:
			ww.ParseStmt(inNode.Init)
			ww.Stb.ParseExpr(inNode.Cond, ww.Current)
			ww1 := LocalsVisitor{lv.Method, NewSymbolTable(), lv.Stb, nil}
			ww2 := LocalsVisitor{lv.Method, NewSymbolTable(), lv.Stb, nil}
			ww1.Current.AddOpenedScope(ww.Current)
			ww2.Current.AddOpenedScope(ww.Current)
			ast.Walk(ww1, inNode.Body)
			ast.Walk(ww2, inNode.Else)
			w = nil
		case *ast.RangeStmt:
			rangeType := ww.Stb.ParseExpr(inNode.X, ww.Current).At(0).(ITypeSymbol)
			switch inNode.Tok {
			case token.DEFINE:
				rangeType = GetBaseType(rangeType)
				var kT, vT ITypeSymbol
				switch rT := rangeType.(type) {
				case *ArrayTypeSymbol:
					kT = &TypeSymbol{Obj: &ast.Object{Name: "int"}, Meths: nil, Posits: new(vector.Vector)}
					vT = rT.ElemType
				case *MapTypeSymbol:
					kT = rT.KeyType
					vT = rT.ValueType
				case *TypeSymbol: //string
					kT = &TypeSymbol{Obj: &ast.Object{Name: "int"}, Meths: nil, Posits: new(vector.Vector)}
					vT = &TypeSymbol{Obj: &ast.Object{Name: "char"}, Meths: nil, Posits: new(vector.Vector)}
				case *ChanTypeSymbol:
					kT = rT.ValueType
				case *ImportedType:
					kT = rT
					vT = rT
				}
				iK := inNode.Key.(*ast.Ident)
				if iK.Name() != "_" {
					toAdd := &VariableSymbol{Obj: CopyObject(iK), VariableType: kT, Posits: new(vector.Vector)}
					toAdd.Posits.Push(NewOccurence(iK.Pos(), toAdd.Obj))
					ww.Current.AddSymbol(toAdd)
				}
				if inNode.Value != nil { // not channel, two range vars
					iV := inNode.Value.(*ast.Ident)
					if iV.Name() != "_" {
						toAdd := &VariableSymbol{Obj: CopyObject(iV), VariableType: vT, Posits: new(vector.Vector)}
						toAdd.Posits.Push(NewOccurence(iV.Pos(), toAdd.Obj))
						ww.Current.AddSymbol(toAdd)
					}
				}
			case token.ASSIGN:
				ww.Stb.ParseExpr(inNode.Key, ww.Current)
				if inNode.Value != nil {
					ww.Stb.ParseExpr(inNode.Value, ww.Current)
				}
			}
			ast.Walk(ww, inNode.Body)
			w = nil
		case *ast.SelectStmt:
			w = ww
		case *ast.SwitchStmt:
			ww.ParseStmt(inNode.Init)
			ww.Stb.ParseExpr(inNode.Tag, ww.Current)
			ast.Walk(ww, inNode.Body)
			w = nil
		case *ast.TypeSwitchStmt:
			ww.ParseStmt(inNode.Init)
			switch tsT := inNode.Assign.(type) {
			case *ast.AssignStmt:
				tsVar := tsT.Lhs[0].(*ast.Ident)
				tsTypeAss := tsT.Rhs[0].(*ast.TypeAssertExpr)
				tsType := ww.Stb.ParseExpr(tsTypeAss.X, ww.Current).At(0).(ITypeSymbol)
				toAdd := &VariableSymbol{Obj: CopyObject(tsVar), VariableType: tsType, Posits: new(vector.Vector)}
				toAdd.Posits.Push(NewOccurence(tsVar.Pos(), toAdd.Obj))
				toAdd.Obj.Kind = -1 //TypeSwitch var
				ww.Current.AddSymbol(toAdd)
			case *ast.ExprStmt:
				tsTypeAss := tsT.X.(*ast.TypeAssertExpr)
				ww.Stb.ParseExpr(tsTypeAss.X, ww.Current)
			}
			ast.Walk(ww, inNode.Body)
			w = nil
		case *ast.CaseClause:
			if inNode.Values != nil {
				for _, v := range inNode.Values {
					ww.Stb.ParseExpr(v, ww.Current)
				}
			}
			ast.Walk(ww, inNode.Body)
			w = nil
		case *ast.CommClause:
			switch {
			case inNode.Lhs != nil:
				switch inNode.Tok {
				case token.DEFINE:
					ccVar := inNode.Lhs.(*ast.Ident)
					ccType := ww.Stb.ParseExpr(inNode.Rhs, ww.Current).At(0).(ITypeSymbol)

					toAdd := &VariableSymbol{Obj: CopyObject(ccVar), VariableType: ccType, Posits: new(vector.Vector)}
					toAdd.Posits.Push(NewOccurence(ccVar.Pos(), toAdd.Obj))
					ww.Current.AddSymbol(toAdd)
				case token.ASSIGN:
					ww.Stb.ParseExpr(inNode.Lhs, ww.Current)
					ww.Stb.ParseExpr(inNode.Rhs, ww.Current)
				}
			case inNode.Rhs != nil:
				ww.Stb.ParseExpr(inNode.Rhs, ww.Current)
			}
			ast.Walk(ww, inNode.Body)
			w = nil
		case *ast.TypeCaseClause:
			switch {
			case inNode.Types == nil:
				//default
			case len(inNode.Types) == 1:
				tsType := ww.Stb.ParseExpr(inNode.Types[0], ww.Current).At(0).(ITypeSymbol)
				if tsVar, ok := lv.Current.FindTypeSwitchVar(); ok {
					toAdd := &VariableSymbol{Obj: tsVar.Obj, VariableType: tsType, Posits: tsVar.Posits}
					//No position, just register symbol
					ww.Current.AddSymbol(toAdd)
				}
			case len(inNode.Types) > 1:
				for _, t := range inNode.Types {
					ww.Stb.ParseExpr(t, ww.Current)
				}
			}
			ast.Walk(ww, inNode.Body)
			w = nil
		case *ast.FuncLit:
			meth := &FunctionSymbol{Obj: &ast.Object{Name: "#"}, FunctionType: lv.Stb.BuildTypeSymbol(inNode.Type), Locals: NewSymbolTable(), Posits: new(vector.Vector)}
			meth.Locals.AddOpenedScope(lv.Current)
			meth.Locals.AddOpenedScope(meth.FunctionType.(*FunctionTypeSymbol).Parameters)
			meth.Locals.AddOpenedScope(meth.FunctionType.(*FunctionTypeSymbol).Results)
			w = LocalsVisitor{meth, meth.Locals, lv.Stb, nil}
		}
	}
	lv.Stb.CurrentSymbolTable = temp
	return
}
func (lv LocalsVisitor) Visit(node interface{}) (w ast.Visitor) {

	w = lv.ParseStmt(node)
	return
}
