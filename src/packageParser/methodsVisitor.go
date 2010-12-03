package packageParser

import (
	"go/ast"
	"strconv"
	"go/token"
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
		
		var basertype,rtype st.ITypeSymbol
		if f.Recv != nil {
			ft.Reciever = mv.Parser.registerNewSymbolTable()
			locals.AddOpenedScope(ft.Reciever)
			e_count := 0
			for _, field := range f.Recv.List {
				basertype = mv.Parser.parseTypeSymbol(field.Type)
				if prtype,ok := basertype.(*st.PointerTypeSymbol);ok{
					rtype = prtype.BaseType;
				}else{
					rtype = basertype
				}
				
				if mv.Parser.Package.AstPackage.Name == "os" {
					fmt.Printf("###@@@### (%s) %s\n", rtype.Name(), f.Name.Name)
				}
				if rtype.Methods() == nil {
					rtype.SetMethods(mv.Parser.registerNewSymbolTable())
				}

				if len(field.Names) == 0 {
					toAdd := &st.VariableSymbol{Obj: &ast.Object{Kind: ast.Var, Name: "$unnamed receiver" + strconv.Itoa(e_count)}, VariableType: basertype, Posits: make(map[string]token.Position), PackFrom: mv.Parser.Package}
					ft.Reciever.AddSymbol(toAdd)

					e_count += 1
				}

				for _, name := range field.Names {
					name.Obj = &ast.Object{Kind: ast.Var, Name: name.Name}

					toAdd := &st.VariableSymbol{Obj: name.Obj, VariableType: basertype, Posits: make(map[string]token.Position), PackFrom: mv.Parser.Package}

					toAdd.AddPosition(name.Pos())

					ft.Reciever.AddSymbol(toAdd)
				}
			}
		}
		f.Name.Obj = &ast.Object{Kind: ast.Var, Name: f.Name.Name}

		toAdd := &st.FunctionSymbol{Obj: f.Name.Obj, FunctionType: ft, Locals: locals, Posits: make(map[string]token.Position), PackFrom: mv.Parser.Package}

		toAdd.AddPosition(f.Name.Pos())

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

	if pp.checkIsVisited(sym) {
		return
	}
	fmt.Printf("opening %s %T from %s\n", sym.Name(), sym, func(p *st.Package) string {
		if p == nil {
			return "nil"
		}
		return p.AstPackage.Name
	}(sym.PackageFrom()))
	
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
		if t.Name() == "Package" {
			fmt.Printf("YEAHHH YEAHHH %p\n", t)
		}
		for variable := range t.Fields.Iter() {
			if _, ok := variable.(*st.VariableSymbol); !ok {

				ts := variable.(st.ITypeSymbol)
				pp.openMethodsAndFields(ts)
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
					t.Fields.AddOpenedScope(ttt.Fields)
					
				}
				//No longer need in type thumb, replace with var
				typeVar := &st.VariableSymbol{ts.Object(), ts, ts.Positions(), pp.Package, ts.IsReadOnly()}
				t.Fields.ReplaceSymbol(ts.Name(), typeVar) //replaces old one
			}

		}
	case *st.PointerTypeSymbol:
		//fmt.Printf("%v %T \n", t.BaseType.Name(), t.BaseType)
		pp.openMethodsAndFields(t.BaseType)
		switch str := t.BaseType.(type){
			case *st.StructTypeSymbol:
				t.Fields = str.Fields
			case *st.PointerTypeSymbol:
				t.Fields = str.Fields
		}
		
		if _, cyc := st.GetBaseType(t); cyc {
			fmt.Printf("%s from %s\n", t.Name(), t.PackageFrom().AstPackage.Name)
			panic("cycle, don't work with that")
		}

		if t.BaseType.Methods() != nil {
			if t.Methods() == nil {
				t.SetMethods(pp.registerNewSymbolTable())
			}
			t.Methods().AddOpenedScope(t.BaseType.Methods())
		}
		if t.BaseType.Name() == "Package" {
			fmt.Printf("YEAHHH YEAHHH %p %p\n", t, t.BaseType)
			//fmt.Println(*t.Methods().String());
		}
		pp.openMethodsAndFields(t.BaseType)
	}
}
