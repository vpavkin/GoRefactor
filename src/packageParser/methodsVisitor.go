package packageParser

import (
	"go/ast"
	"strconv"
	//"go/token"
	"refactoring/st"
)
//import "fmt"
//Represents an ast.Visitor, walking along ast.tree and registering all methods and functions met
type methodsVisitor struct {
	Parser *packageParser
}

func (mv *methodsVisitor) Visit(node ast.Node) (w ast.Visitor) {
	w = mv
	switch f := node.(type) {
	case *ast.FuncDecl:

		fft, cyc := st.GetBaseType(mv.Parser.parseTypeSymbol(f.Type))
		if cyc {
			panic("unexpected cycle")
		}
		ft := fft.(*st.FunctionTypeSymbol)
		locals := st.NewSymbolTable(mv.Parser.Package)

		locals.AddOpenedScope(ft.Parameters)
		locals.AddOpenedScope(ft.Results)
		locals.AddOpenedScope(ft.Reciever)

		var basertype, rtype st.ITypeSymbol
		if f.Recv != nil {

			e_count := 0
			for _, field := range f.Recv.List {
				basertype = mv.Parser.parseTypeSymbol(field.Type)
				if prtype, ok := basertype.(*st.PointerTypeSymbol); ok {
					rtype = prtype.BaseType
				} else {
					rtype = basertype
				}

				if mv.Parser.Package.AstPackage.Name == "os" {
					// 					fmt.Printf("###@@@### (%s) %s\n", rtype.Name(), f.Name.Name)
				}
				if rtype.Methods() == nil {
					panic("ok, this is a test panic")
					rtype.SetMethods(st.NewSymbolTable(mv.Parser.Package))
				}

				if len(field.Names) == 0 {
					toAdd := st.MakeVariable("$unnamed receiver"+strconv.Itoa(e_count), ft.Reciever, basertype)
					ft.Reciever.AddSymbol(toAdd)
					e_count += 1
				}

				for _, name := range field.Names {

					toAdd := st.MakeVariable(name.Name, ft.Reciever, basertype)
					mv.Parser.registerIdent(toAdd, name)
					ft.Reciever.AddSymbol(toAdd)
				}
			}
		}

		toAdd := st.MakeFunction(f.Name.Name, nil, ft) // Scope is set 5 lines down
		toAdd.Locals = locals

		mv.Parser.registerIdent(toAdd, f.Name)

		if f.Recv != nil {
			rtype.AddMethod(toAdd)
			toAdd.Scope_ = rtype.Methods()
		} else {
			mv.Parser.RootSymbolTable.AddSymbol(toAdd)
			toAdd.Scope_ = mv.Parser.RootSymbolTable
		}
	}
	return
}
func (pp *packageParser) fixMethodsAndFields() {

	pp.visited = make(map[st.Symbol]bool)

	pp.RootSymbolTable.ForEachNoLock(func(sym st.Symbol) {
		pp.openMethodsAndFields(sym)
	})
}
func (pp *packageParser) openMethodsAndFields(sym st.Symbol) {

	if pp.checkIsVisited(sym) {
		return
	}
	// 	fmt.Printf("opening %s %T from %s\n", sym.Name(), sym, func(p *st.Package) string {
	// 		if p == nil {
	// 			return "nil"
	// 		}
	// 		return p.AstPackage.Name
	// 	}(sym.PackageFrom()))

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
			t.Parameters.ForEachNoLock(func(sym st.Symbol) {
				v := sym.(*st.VariableSymbol)
				pp.openMethodsAndFields(v.VariableType)
			})
		}
		if t.Results != nil {
			t.Results.ForEachNoLock(func(sym st.Symbol) {
				v := sym.(*st.VariableSymbol)
				pp.openMethodsAndFields(v.VariableType)
			})
		}
	case *st.InterfaceTypeSymbol:
		if t.Meths != nil {
			t.Meths.ForEachNoLock(func(sym st.Symbol) {
				if _, ok := sym.(*st.FunctionSymbol); !ok {
					//EmbeddedInterface
					pp.openMethodsAndFields(sym)

					ts := sym.(*st.InterfaceTypeSymbol)
					t.Meths.AddOpenedScope(ts.Meths)
					//Delete functionSymbol which is now useles from interface
					t.Meths.RemoveSymbol(sym.Name())
				}
			})
		}
	case *st.MapTypeSymbol:
		pp.openMethodsAndFields(t.KeyType)
		pp.openMethodsAndFields(t.ValueType)
	case *st.StructTypeSymbol:
		// 		if t.Name() == "Package" {
		// 			fmt.Printf("YEAHHH YEAHHH %p\n", t)
		// 		}
		t.Fields.ForEachNoLock(func(variable st.Symbol) {
			if _, ok := variable.(*st.VariableSymbol); !ok {

				ts := variable.(st.ITypeSymbol)
				pp.openMethodsAndFields(ts)
				//Methods
				if ts.Methods() != nil {
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
				typeVar := st.MakeVariable(ts.Name(), t.Fields, ts)
				typeVar.Idents = ts.Identifiers()
				typeVar.Posits = ts.Positions()
				t.Fields.ReplaceSymbol(ts.Name(), typeVar) //replaces old one
			}

		})
	case *st.PointerTypeSymbol:
		//fmt.Printf("%v %T \n", t.BaseType.Name(), t.BaseType)
		pp.openMethodsAndFields(t.BaseType)
		switch str := t.BaseType.(type) {
		case *st.StructTypeSymbol:
			t.Fields = str.Fields
		case *st.PointerTypeSymbol:
			t.Fields = str.Fields
		}

		if _, cyc := st.GetBaseType(t); cyc {
			// 			fmt.Printf("%s from %s\n", t.Name(), t.PackageFrom().AstPackage.Name)
			panic("cycle, don't work with that")
		}

		if t.BaseType.Methods() != nil {
			t.Methods().AddOpenedScope(t.BaseType.Methods())
		}
		// 		if t.BaseType.Name() == "Package" {
		// 			fmt.Printf("YEAHHH YEAHHH %p %p\n", t, t.BaseType)
		// 			fmt.Println(*t.Methods().String());
		// 		}
		pp.openMethodsAndFields(t.BaseType)
	}
}
