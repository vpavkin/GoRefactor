package packageParser

import (
	"go/ast"
	"go/token"
	"container/vector"
	"st"
	"fmt"
)

type localsVisitor struct {
	Parser *packageParser
}

type innerScopeVisitor struct {
	Method   *st.FunctionSymbol
	Current  *st.SymbolTable
	Parser   *packageParser
	IotaType st.ITypeSymbol
}

func (lv *localsVisitor) Visit(node interface{}) (w ast.Visitor) {
	w = lv
	switch f := node.(type) {

	case *ast.FuncDecl:

		scope := lv.Parser.CurrentSymbolTable
		if f.Recv != nil {
			for _, field := range f.Recv.List {
				rtype := lv.Parser.parseTypeSymbol(field.Type)
				if _, ok := rtype.(*st.UnresolvedTypeSymbol); ok || rtype == nil {
					panic("couldn't find method reciever")
				}
				scope = rtype.Methods()
			}
		}
		m, ok := scope.LookUp(f.Name.Name, "")
		if !ok {
			panic("couldn't find method " + f.Name.Name)
		}
		meth := m.(*st.FunctionSymbol)
		meth.Locals.AddOpenedScope(lv.Parser.RootSymbolTable)
		fmt.Printf("method %s\n", meth.Name())
		ww := &innerScopeVisitor{meth, meth.Locals, lv.Parser, nil}
		ast.Walk(ww, f.Body)
		w = nil
	}
	return
}

func (lv *innerScopeVisitor) getValuesTypesAss(s *ast.AssignStmt) (valuesTypes *vector.Vector) {
	if len(s.Rhs) == 1 && len(s.Rhs) < len(s.Lhs) {
		valuesTypes = lv.Parser.parseExpr(s.Rhs[0])
	} else {
		valuesTypes = new(vector.Vector)
		for _, n := range s.Rhs {
			valuesTypes.Push(lv.Parser.parseExpr(n).At(0))
		}
	}
	return
}
func (lv *innerScopeVisitor) getValuesTypesValSpec(s *ast.ValueSpec) (valuesTypes *vector.Vector) {
	if len(s.Values) == 1 && len(s.Values) < len(s.Names) {
		valuesTypes = lv.Parser.parseExpr(s.Values[0])
	} else {
		valuesTypes = new(vector.Vector)
		for _, n := range s.Values {
			valuesTypes.Push(lv.Parser.parseExpr(n).At(0))
		}
	}
	return
}

func (lv *innerScopeVisitor) parseStmt(node interface{}) (w ast.Visitor) {
	if node == nil {
		return nil
	}
	w = lv
	fmt.Printf("ps %p %p %p %T ", lv.Parser.CurrentSymbolTable, lv.Current, lv.Method.Locals, node)
	if id, ok := node.(*ast.Ident); ok {
		fmt.Printf("%s ", id.Name)
	}
	println()
	temp := lv.Parser.CurrentSymbolTable
	lv.Parser.CurrentSymbolTable = lv.Current
	defer func() { lv.Parser.CurrentSymbolTable = temp }()
	switch s := node.(type) {

	case *ast.GenDecl:
		var IotaType st.ITypeSymbol = nil
		if len(s.Specs) > 0 {
			if vs, ok := s.Specs[0].(*ast.ValueSpec); ok {
				switch{
					case vs.Type != nil:
						ts := lv.Parser.parseTypeSymbol(vs.Type)
						if _, ok := ts.(*st.UnresolvedTypeSymbol); (ts == nil) || ok {
							panic("unresolved type at locals scope: " + ts.Name())
						}
						IotaType = ts;
					case vs.Values!=nil && len(vs.Values) > 0:
						ts := lv.Parser.parseExpr(vs.Values[0]).At(0).(st.ITypeSymbol)
						IotaType = ts;
					default:
						panic("decl without either type or value????")
				}
			}
		}
		w = &innerScopeVisitor{lv.Method, lv.Current, lv.Parser, IotaType}

	case *ast.ValueSpec: //Specify a new variable

		ts := lv.Parser.parseTypeSymbol(s.Type)

		if ts == nil {
			ts = lv.IotaType
		}
		valuesTypes := lv.getValuesTypesValSpec(s)
		for i, n := range s.Names {
			curTs := ts
			if ts == nil {
				curTs = valuesTypes.At(i).(st.ITypeSymbol)
			}
			n.Obj = &ast.Object{Kind: ast.Var, Name: n.Name}

			toAdd := &st.VariableSymbol{Obj: n.Obj, VariableType: curTs, Posits: make(map[string]token.Position), PackFrom: lv.Parser.Package}
			toAdd.AddPosition(n.Pos())
			lv.Parser.CurrentSymbolTable.AddSymbol(toAdd)
		}

	case *ast.AssignStmt:
		valuesTypes := lv.getValuesTypesAss(s)
		switch s.Tok {
		case token.DEFINE: //Specify a new variable

			for i, nn := range s.Lhs {
				n := nn.(*ast.Ident)
				if n.Name != "_" {
					n.Obj = &ast.Object{Kind: ast.Var, Name: n.Name}
					toAdd := &st.VariableSymbol{Obj: n.Obj, VariableType: valuesTypes.At(i).(st.ITypeSymbol), Posits: make(map[string]token.Position), PackFrom: lv.Parser.Package}
					toAdd.AddPosition(n.Pos())
					lv.Current.AddSymbol(toAdd)
					fmt.Printf("DEFINED %s\n", toAdd.Name())
				}
			}
		case token.ASSIGN: //Pos
			for _, nn := range s.Lhs {
				fmt.Printf("<!>")
				lv.Parser.parseExpr(nn)
			}
			w = nil
		}
	case *ast.TypeSpec: //Specify a new type
		ts := lv.Parser.parseTypeSymbol(s.Type)

		switch ts.(type) {
		case *st.PointerTypeSymbol, *st.ArrayTypeSymbol, *st.StructTypeSymbol, *st.InterfaceTypeSymbol, *st.MapTypeSymbol, *st.ChanTypeSymbol, *st.FunctionTypeSymbol:

			s.Name.Obj = &ast.Object{Kind: ast.Typ, Name: s.Name.Name}

			if ts.Object() == nil {
				//No such symbol in CurrentSymbolTable
				ts.SetObject(s.Name.Obj)
			} else {
				//There is an equal type symbol with different name => create alias
				ts = &st.AliasTypeSymbol{&st.TypeSymbol{Obj: s.Name.Obj, Posits: make(map[string]token.Position), PackFrom: lv.Parser.Package}, ts}
			}
		default:
			panic("shit, no type symbol returned")
		}
		ts.AddPosition(s.Name.Pos())
		lv.Current.AddSymbol(ts)
	case *ast.ExprStmt:
		lv.Parser.parseExpr(s.X)
	case *ast.DeferStmt:
		lv.Parser.parseExpr(s.Call)
	case *ast.GoStmt:
		lv.Parser.parseExpr(s.Call)
	case *ast.IncDecStmt:
		lv.Parser.parseExpr(s.X)
	case *ast.ReturnStmt:
		if s.Results != nil { //mb not needed
			for _, exp := range s.Results {
				lv.Parser.parseExpr(exp)
			}
		}
	case *ast.SwitchStmt, *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.FuncLit, *ast.SelectStmt, *ast.TypeSwitchStmt, *ast.CaseClause, *ast.TypeCaseClause, *ast.CommClause:

		w = lv.parseBlockStmt(node)

	}

	return
}
func (lv *innerScopeVisitor) parseBlockStmt(node interface{}) (w ast.Visitor) {
	if node == nil {
		return nil
	}
	w = lv
	table := lv.Parser.registerNewSymbolTable()
	fmt.Printf(" %p %p %p \n", lv.Parser.CurrentSymbolTable, lv.Current, lv.Method.Locals)
	table.AddOpenedScope(lv.Current)
	ww := &innerScopeVisitor{lv.Method, table, lv.Parser, nil}

	temp := lv.Parser.CurrentSymbolTable
	lv.Parser.CurrentSymbolTable = table
	defer func() { lv.Parser.CurrentSymbolTable = temp }()

	switch inNode := node.(type) {
	case *ast.ForStmt:
		ww.parseStmt(inNode.Init)
		ww.Parser.parseExpr(inNode.Cond)
		ww.parseStmt(inNode.Post)
		ast.Walk(ww, inNode.Body)
		w = nil
	case *ast.IfStmt:
		ww.parseStmt(inNode.Init)
		ww.Parser.parseExpr(inNode.Cond)
		ww1 := &innerScopeVisitor{lv.Method, lv.Parser.registerNewSymbolTable(), lv.Parser, nil}
		ww2 := &innerScopeVisitor{lv.Method, lv.Parser.registerNewSymbolTable(), lv.Parser, nil}
		ww1.Current.AddOpenedScope(ww.Current)
		ww2.Current.AddOpenedScope(ww.Current)
		ast.Walk(ww1, inNode.Body)
		ast.Walk(ww2, inNode.Else)
		w = nil
	case *ast.RangeStmt:
		rangeType := ww.Parser.parseExpr(inNode.X).At(0).(st.ITypeSymbol)
		fmt.Printf("range type = %s, %T\n", rangeType.Name(), rangeType)
		switch inNode.Tok {
		case token.DEFINE:
			if rangeType, _ = st.GetBaseType(rangeType); rangeType == nil {
				panic("unexpected cycle")
			}

			var kT, vT st.ITypeSymbol
			switch rT := rangeType.(type) {
			case *st.ArrayTypeSymbol:
				kT = st.PredeclaredTypes["int"]
				vT = rT.ElemType
			case *st.MapTypeSymbol:
				kT = rT.KeyType
				vT = rT.ValueType
			case *st.TypeSymbol: //string
				kT = st.PredeclaredTypes["int"]
				vT = st.PredeclaredTypes["byte"]
			case *st.ChanTypeSymbol:
				kT = rT.ValueType
			case *st.UnresolvedTypeSymbol:
				panic("unresolved at range")
			}
			iK := inNode.Key.(*ast.Ident)
			if iK.Name != "_" {
				iK.Obj = &ast.Object{Kind: ast.Var, Name: iK.Name}
				toAdd := &st.VariableSymbol{Obj: iK.Obj, VariableType: kT, Posits: make(map[string]token.Position), PackFrom: ww.Parser.Package}
				toAdd.AddPosition(iK.Pos())
				ww.Current.AddSymbol(toAdd)
				fmt.Printf("range key added %s %T\n", toAdd.Name(), toAdd)
			}
			if inNode.Value != nil { // not channel, two range vars
				iV := inNode.Value.(*ast.Ident)
				if iV.Name != "_" {
					iV.Obj = &ast.Object{Kind: ast.Var, Name: iV.Name}
					toAdd := &st.VariableSymbol{Obj: iV.Obj, VariableType: vT, Posits: make(map[string]token.Position), PackFrom: ww.Parser.Package}
					toAdd.AddPosition(iV.Pos())
					ww.Current.AddSymbol(toAdd)
					fmt.Printf("range value added %s %T\n", toAdd.Name(), toAdd)
				}
			}
		case token.ASSIGN:
			ww.Parser.parseExpr(inNode.Key)
			if inNode.Value != nil {
				ww.Parser.parseExpr(inNode.Value)
			}
		}
		ast.Walk(ww, inNode.Body)
		fmt.Printf("end of range\n")
		w = nil
	case *ast.SelectStmt:
		w = ww
	case *ast.SwitchStmt:
		ww.parseStmt(inNode.Init)
		ww.Parser.parseExpr(inNode.Tag)
		ast.Walk(ww, inNode.Body)
		w = nil
	case *ast.TypeSwitchStmt:
		ww.parseStmt(inNode.Init)
		switch tsT := inNode.Assign.(type) {
		case *ast.AssignStmt:
			tsVar := tsT.Lhs[0].(*ast.Ident)
			tsTypeAss := tsT.Rhs[0].(*ast.TypeAssertExpr)
			tsType := ww.Parser.parseExpr(tsTypeAss.X).At(0).(st.ITypeSymbol)

			tsVar.Obj = &ast.Object{Kind: ast.Var, Name: tsVar.Name}
			toAdd := &st.VariableSymbol{Obj: tsVar.Obj, VariableType: tsType, Posits: make(map[string]token.Position), PackFrom: ww.Parser.Package}
			toAdd.AddPosition(tsVar.Pos())
			toAdd.Obj.Kind = -1 //TypeSwitch var
			ww.Current.AddSymbol(toAdd)
		case *ast.ExprStmt:
			tsTypeAss := tsT.X.(*ast.TypeAssertExpr)
			ww.Parser.parseExpr(tsTypeAss.X)
		}
		ast.Walk(ww, inNode.Body)
		w = nil
	case *ast.CaseClause:
		if inNode.Values != nil {
			for _, v := range inNode.Values {
				ww.Parser.parseExpr(v)
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
				ccType := ww.Parser.parseExpr(inNode.Rhs).At(0).(st.ITypeSymbol)

				ccVar.Obj = &ast.Object{Kind: ast.Var, Name: ccVar.Name}
				toAdd := &st.VariableSymbol{Obj: ccVar.Obj, VariableType: ccType, Posits: make(map[string]token.Position), PackFrom: ww.Parser.Package}
				toAdd.AddPosition(ccVar.Pos())
				ww.Current.AddSymbol(toAdd)
			case token.ASSIGN:
				ww.Parser.parseExpr(inNode.Lhs)
				ww.Parser.parseExpr(inNode.Rhs)
			}
		case inNode.Rhs != nil:
			ww.Parser.parseExpr(inNode.Rhs)
		}
		ast.Walk(ww, inNode.Body)
		w = nil
	case *ast.TypeCaseClause:
		switch {
		case inNode.Types == nil:
			//default
		case len(inNode.Types) == 1:
			tsType := ww.Parser.parseExpr(inNode.Types[0]).At(0).(st.ITypeSymbol)
			if tsVar, ok := lv.Current.FindTypeSwitchVar(); ok {
				toAdd := &st.VariableSymbol{Obj: tsVar.Obj, VariableType: tsType, Posits: tsVar.Posits, PackFrom: ww.Parser.Package}
				//No position, just register symbol
				ww.Current.AddSymbol(toAdd)
			}
		case len(inNode.Types) > 1:
			for _, t := range inNode.Types {
				ww.Parser.parseExpr(t)
			}
		}
		ast.Walk(ww, inNode.Body)
		w = nil
	case *ast.FuncLit:
		meth := &st.FunctionSymbol{Obj: &ast.Object{Name: "#"}, FunctionType: lv.Parser.parseTypeSymbol(inNode.Type), Locals: lv.Parser.registerNewSymbolTable(), Posits: make(map[string]token.Position), PackFrom: ww.Parser.Package}
		meth.Locals.AddOpenedScope(lv.Current)
		if meth.FunctionType.(*st.FunctionTypeSymbol).Parameters != nil {
			meth.Locals.AddOpenedScope(meth.FunctionType.(*st.FunctionTypeSymbol).Parameters)
		}
		if meth.FunctionType.(*st.FunctionTypeSymbol).Results != nil {
			meth.Locals.AddOpenedScope(meth.FunctionType.(*st.FunctionTypeSymbol).Results)
		}
		w = &innerScopeVisitor{meth, meth.Locals, lv.Parser, nil}
	}
	return w
}
func (isv *innerScopeVisitor) Visit(node interface{}) (w ast.Visitor) {

	return isv.parseStmt(node)

}
