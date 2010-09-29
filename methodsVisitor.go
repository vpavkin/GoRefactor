package symbolTable

import (
	"go/ast"
	"strconv"
	"container/vector"
)

//Represents an ast.Visitor, walking along ast.tree and registering all methods and functions met
type MethodsVisitor struct {
	Stb *SymbolTableBuilder //pointer to parent SymbolTableBuilder
}

func (mv MethodsVisitor) Visit(node interface{}) (w ast.Visitor) {
	w = mv
	switch f := node.(type) {
	case *ast.FuncDecl:

		ft := mv.Stb.BuildTypeSymbol(f.Type).(*FunctionTypeSymbol)
		locals := NewSymbolTable()
		locals.AddOpenedScope(ft.Parameters)
		locals.AddOpenedScope(ft.Results)
		locals.AddOpenedScope(ft.Reciever)
		if f.Recv != nil {
			e_count := 0
			for _, field := range f.Recv.List {
				rtype := mv.Stb.BuildTypeSymbol(field.Type)
				if len(field.Names) == 0 {
					toAdd := &VariableSymbol{Obj: &ast.Object{4, field.Type.Pos(), "*unnamed " + strconv.Itoa(e_count) + "*"}, VariableType: rtype, Posits: new(vector.Vector)}
					ft.Reciever.AddSymbol(toAdd)

					e_count += 1
				}
				for _, name := range field.Names {
					toAdd := &VariableSymbol{Obj: CopyObject(name), VariableType: rtype, Posits: new(vector.Vector)}
					toAdd.Posits.Push(NewOccurence(name.Pos(), toAdd.Obj))
					ft.Reciever.AddSymbol(toAdd)
				}
				locals.AddOpenedScope(ft.Reciever)

				toAdd := &FunctionSymbol{Obj: f.Name.Obj, FunctionType: ft, Locals: locals, Posits: new(vector.Vector)}
				toAdd.Posits.Push(NewOccurence(f.Name.Pos(), toAdd.Obj))
				rtype.AddMethod(toAdd)
			}
		} else {
			toAdd := &FunctionSymbol{Obj: CopyObject(f.Name), FunctionType: ft, Locals: locals, Posits: new(vector.Vector)}
			toAdd.Posits.Push(NewOccurence(f.Name.Pos(), toAdd.Obj))
			mv.Stb.RootSymbolTable.AddSymbol(toAdd)
		}
	}
	return
}
