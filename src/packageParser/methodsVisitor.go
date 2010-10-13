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
		locals.AddOpenedScope(ft.Parameters)
		locals.AddOpenedScope(ft.Results)
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
					toAdd := &st.VariableSymbol{Obj: &ast.Object{Kind: ast.Var, Name: "*unnamed receiver" + strconv.Itoa(e_count) + "*"}, VariableType: rtype, Posits: new(vector.Vector)}
					ft.Reciever.AddSymbol(toAdd)

					e_count += 1
				}

				for _, name := range field.Names {
					name.Obj = &ast.Object{Kind: ast.Var, Name: name.Name}

					toAdd := &st.VariableSymbol{Obj: name.Obj, VariableType: rtype, Posits: new(vector.Vector)}
					toAdd.AddPosition(st.NewOccurence(name.Pos()))
					ft.Reciever.AddSymbol(toAdd)
				}
			}
		}
		f.Name.Obj = &ast.Object{Kind: ast.Var, Name: f.Name.Name}

		toAdd := &st.FunctionSymbol{Obj: f.Name.Obj, FunctionType: ft, Locals: locals, Posits: new(vector.Vector)}
		toAdd.AddPosition(st.NewOccurence(f.Name.Pos()))
		if f.Recv != nil {
			rtype.AddMethod(toAdd)
		} else {
			mv.Parser.RootSymbolTable.AddSymbol(toAdd)
		}
	}
	return
}
func (pp *packageParser) fixMethods() {

	visited = make(map[string]bool)

	for _, s := range pp.RootSymbolTable.Table {
		pp.openMethods(s)
	}

}
func (pp *packageParser) openMethods(sym st.Symbol) {

	fmt.Printf("opening %s %T\n", sym.Name(), sym)
	if checkIsVisited(sym) {
		return
	}
	if st.IsPredeclaredIdentifier(sym.Name()) {
		return
	}

	switch t := sym.(type) {
	case *st.ArrayTypeSymbol:
		pp.openMethods(t.ElemType)
	case *st.ChanTypeSymbol:
		pp.openMethods(t.ValueType)
	case *st.FunctionTypeSymbol:
		if t.Parameters != nil {
			for _, variable := range t.Parameters.Table {
				v := variable.(*st.VariableSymbol)
				pp.openMethods(v.VariableType)
			}
		}
		if t.Results != nil {
			for _, variable := range t.Results.Table {
				v := variable.(*st.VariableSymbol)
				pp.openMethods(v.VariableType)
			}
		}
	case *st.MapTypeSymbol:
		pp.openMethods(t.KeyType)
		pp.openMethods(t.ValueType)
	case *st.StructTypeSymbol:
		for _, variable := range t.Fields.Table {
			if _, ok := variable.(*st.VariableSymbol); !ok {

				ts := variable.(st.ITypeSymbol)
				if ts.Methods() != nil {
					if t.Methods() == nil {
						t.SetMethods(pp.registerNewSymbolTable())
					}
					t.Methods().AddOpenedScope(ts.Methods())
				}

				//No longer need in type thumb, replace with var
				typeVar := &st.VariableSymbol{ts.Object(), ts, ts.Positions(), ts.IsReadOnly()}
				t.Fields.AddSymbol(typeVar) //replaces old one

				pp.openMethods(variable)
			}
		}
	case *st.PointerTypeSymbol:
		fmt.Printf("%v %T \n", t.BaseType.Name(), t.BaseType)
		if str, ok := t.GetBaseStruct(); ok {
			if str.Methods() != nil {
				if t.Methods() == nil {
					t.SetMethods(pp.registerNewSymbolTable())
				}
				t.Methods().AddOpenedScope(str.Methods())
			}
		}
		pp.openMethods(t.BaseType)
	}
}
