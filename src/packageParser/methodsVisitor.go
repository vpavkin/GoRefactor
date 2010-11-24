package packageParser

import (
	"go/ast"
	"strconv"
	"container/vector"
	"st"
)
import "fmt"
//Represents an ast.Visitor, walking along ast.tree and registering all methods and functions met
type methodsVisitor struct {
	Parser *packageParser
}

func (mv *methodsVisitor) Visit(node interface{}) (w ast.Visitor) {
	w = mv
	switch f := node.(type) {
	case *ast.FuncDecl:

		var rtype st.ITypeSymbol
		fft, cyc := st.GetBaseType(mv.Parser.parseTypeSymbol(f.Type))
		if cyc {
			fmt.Printf("ERROR: cycle wasn't expected. Visit, methodsVisitor.go")
		}
		ft := fft.(*st.FunctionTypeSymbol)
		locals := mv.Parser.registerNewSymbolTable()
		//Parameters and Results maybe nil, Reciever 100% is
		if ft.Parameters != nil {
			locals.AddOpenedScope(ft.Parameters)
		}
		if ft.Results != nil {
			locals.AddOpenedScope(ft.Results)
		}

		if f.Recv != nil {
			ft.Reciever = mv.Parser.registerNewSymbolTable()
			locals.AddOpenedScope(ft.Reciever)
			e_count := 0
			for _, field := range f.Recv.List {

				rtype = mv.Parser.parseTypeSymbol(field.Type)
				if rtype.Methods() == nil {
					rtype.SetMethods(mv.Parser.registerNewSymbolTable())
				}

				if len(field.Names) == 0 {
					toAdd := &st.VariableSymbol{Obj: &ast.Object{Kind: ast.Var, Name: "$unnamed receiver" + strconv.Itoa(e_count)}, VariableType: rtype, Posits: new(vector.Vector), PackFrom: mv.Parser.Package}
					ft.Reciever.AddSymbol(toAdd)

					e_count += 1
				}

				for _, name := range field.Names {
					name.Obj = &ast.Object{Kind: ast.Var, Name: name.Name}

					toAdd := &st.VariableSymbol{Obj: name.Obj, VariableType: rtype, Posits: new(vector.Vector), PackFrom: mv.Parser.Package}
					if mv.Parser.RegisterPositions {
						toAdd.AddPosition(st.NewOccurence(name.Pos()))
					}
					ft.Reciever.AddSymbol(toAdd)
				}
			}
		}
		f.Name.Obj = &ast.Object{Kind: ast.Var, Name: f.Name.Name}

		toAdd := &st.FunctionSymbol{Obj: f.Name.Obj, FunctionType: ft, Locals: locals, Posits: new(vector.Vector), PackFrom: mv.Parser.Package}
		if mv.Parser.RegisterPositions {
			toAdd.AddPosition(st.NewOccurence(f.Name.Pos()))
		}
		if f.Recv != nil {
			rtype.AddMethod(toAdd)
		} else {
			mv.Parser.RootSymbolTable.AddSymbol(toAdd)
		}
	}
	return
}
func (pp *packageParser) fixMethodsAndFields() {

	pp.visited = make(map[string]bool)

	for s := range pp.RootSymbolTable.Iter() {
		pp.openMethodsAndFields(s)
	}
}
func (pp *packageParser) openMethodsAndFields(sym st.Symbol) {

	//fmt.Printf("opening %s %T\n", sym.Name(), sym)
	if pp.checkIsVisited(sym) {
		return
	}
	if st.IsPredeclaredIdentifier(sym.Name()) {
		return
	}

	switch t := sym.(type) {
	case *st.ArrayTypeSymbol:
		pp.openMethodsAndFields(t.ElemType)
	case *st.ChanTypeSymbol:
		pp.openMethodsAndFields(t.ValueType)
	case *st.FunctionTypeSymbol:
		if t.Parameters != nil {
			for variable := range t.Parameters.Iter() {
				v := variable.(*st.VariableSymbol)
				pp.openMethodsAndFields(v.VariableType)
			}
		}
		if t.Results != nil {
			for variable := range t.Results.Iter() {
				v := variable.(*st.VariableSymbol)
				pp.openMethodsAndFields(v.VariableType)
			}
		}
	case *st.InterfaceTypeSymbol:
		if t.Meths != nil {
			for sym := range t.Meths.Iter() {
				if _, ok := sym.(*st.FunctionSymbol); !ok {
					//EmbeddedInterface
					ts := sym.(*st.InterfaceTypeSymbol)
					t.Meths.AddOpenedScope(ts.Meths)
					//Delete functionSymbol which is now useles from interface
					t.Meths.RemoveSymbol(sym.Name())
				}
				pp.openMethodsAndFields(sym)
			}
		}
	case *st.MapTypeSymbol:
		pp.openMethodsAndFields(t.KeyType)
		pp.openMethodsAndFields(t.ValueType)
	case *st.StructTypeSymbol:
		for variable := range t.Fields.Iter() {
			if _, ok := variable.(*st.VariableSymbol); !ok {

				ts := variable.(st.ITypeSymbol)
				//Methods
				if ts.Methods() != nil {
					if t.Methods() == nil {
						t.SetMethods(pp.registerNewSymbolTable())
					}
					t.Methods().AddOpenedScope(ts.Methods())
				}
				//Fields
				switch ttt := ts.(type) {
				case *st.StructTypeSymbol:
					t.Fields.AddOpenedScope(ttt.Fields)
				case *st.PointerTypeSymbol:
					if table, ok1 := ttt.GetBaseStruct(); ok1 {
						t.Fields.AddOpenedScope(table.Fields)
					}
				}
				//No longer need in type thumb, replace with var
				typeVar := &st.VariableSymbol{ts.Object(), ts, ts.Positions(), pp.Package, ts.IsReadOnly()}
				t.Fields.ReplaceSymbol(ts.Name(), typeVar) //replaces old one
			}
		}
	case *st.PointerTypeSymbol:
		//fmt.Printf("%v %T \n", t.BaseType.Name(), t.BaseType)

		if str, ok := t.GetBaseStruct(); ok {

			t.Fields = pp.registerNewSymbolTable()
			t.Fields.AddOpenedScope(str.Fields)

			if str.Methods() != nil {
				if t.Methods() == nil {
					t.SetMethods(pp.registerNewSymbolTable())
				}
				t.Methods().AddOpenedScope(str.Methods())
			}
		}
		pp.openMethodsAndFields(t.BaseType)
	}
}
