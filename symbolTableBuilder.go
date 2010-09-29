package symbolTable

import (
	"go/ast"
	"strconv"
	"container/vector"
	"fmt"
)

func init() {
	STB = NewSymbolTableBuilder()
	STB.SymbolTablePool.Push(STB.RootSymbolTable)
}

const (
	TYPES_MODE = iota
	GLOBALS_MODE
	METHODS_MODE
	LOCALS_MODE
)

var STB *SymbolTableBuilder

//Represents a builder object, which provides a symbolTable in a few passes
type SymbolTableBuilder struct {
	RootSymbolTable    *SymbolTable      //Root symbol table, containing all global types, methods and variables.
	CurrentSymbolTable *SymbolTable      // A symbol table which is curently being filled
	TypesBuilder       *TypeVisitor      // An ast.Visitor to register all global types
	GlobalsBuilder     *GlobalsVisitor   // An ast.Visitor to register all global variables
	MethodsBuilder     *MethodsVisitor   // An ast.Visitor to register all methods
	LocalsBuilder      *InMethodsVisitor // An ast.Visitor to register inner scopes of methods
	Mode               int               // Current builder mode (TYPE_MODE, GLOBALS_MODE, FUNCTIONS_MODE or LOCALS_MODE)
	PositionsRegister  bool              // If true, information about positions is gathered

	SymbolTablePool *vector.Vector //links to all symbol tables including nested
}

func IsGoFile(fileName string) bool {
	l := len(fileName)
	if l < 3 {
		return false
	}
	return fileName[l-1] == 'o' && fileName[l-2] == 'g' && fileName[l-3] == '.'
}

func (stb SymbolTableBuilder) FindSymbolByPosition(fileName string, line int, column int) (Symbol, *SymbolTable, bool) {
	for s := range stb.SymbolTablePool.Iter() {
		st := s.(*SymbolTable)
		if sym, found := st.FindSymbolByPosition(fileName, line, column); found {
			return sym, st, true
		}
	}
	return nil, nil, false
}

func (stb SymbolTableBuilder) RenameSymbol(sym Symbol, newName string) int {
	for pos := range sym.Positions().Iter() {
		obj := pos.(Occurence).Obj
		obj.Name = newName
	}
	return sym.Positions().Len()
}

/*^^SymbolTableBuilder Methods^^*/
func NewSymbolTableBuilder() *SymbolTableBuilder {

	CurrentSymbolTable := NewSymbolTable()
	stb := &SymbolTableBuilder{CurrentSymbolTable, CurrentSymbolTable, nil, nil, nil, nil, TYPES_MODE, true, new(vector.Vector)}
	tv := &TypeVisitor{stb, make(chan ITypeSymbol), make(chan int), make(map[string]bool)}
	gv := &GlobalsVisitor{stb, nil}
	mv := &MethodsVisitor{stb}
	imv := &InMethodsVisitor{stb}
	stb.TypesBuilder = tv
	stb.GlobalsBuilder = gv
	stb.MethodsBuilder = mv
	stb.LocalsBuilder = imv
	return stb
}


func (stb *SymbolTableBuilder) BuildSymbolTable(pack *ast.Package) {

	stb.SymbolTablePool = new(vector.Vector)
	stb.RootSymbolTable = NewSymbolTable()
	stb.CurrentSymbolTable = stb.RootSymbolTable
	stb.Mode = TYPES_MODE

	go stb.TypesBuilder.RegisterTypes(stb.CurrentSymbolTable)

	go func() { stb.TypesBuilder.Ack <- 1 }()

	for filename, atree := range pack.Files {
		if IsGoFile(filename) {
			ast.Walk(stb.TypesBuilder, atree.Decls)
		}
	}

	<-stb.TypesBuilder.Ack
	stb.TypesBuilder.Types <- nil
	<-stb.TypesBuilder.Ack

	stb.Mode = METHODS_MODE
	for filename, atree := range pack.Files {
		if IsGoFile(filename) {
			ast.Walk(stb.MethodsBuilder, atree.Decls)
		}
	}

	stb.Mode = GLOBALS_MODE
	for filename, atree := range pack.Files {
		if IsGoFile(filename) {
			ast.Walk(stb.GlobalsBuilder, atree.Decls)
		}
	}

	//Locals
	for filename, atree := range pack.Files {
		if IsGoFile(filename) {
			ast.Walk(stb.LocalsBuilder, atree.Decls)
		}
	}
}

//Builds a type symbol according to given ast.Expression
func (stb *SymbolTableBuilder) BuildTypeSymbol(typ ast.Expr) (result ITypeSymbol) {
	switch t := typ.(type) {
	case *ast.StarExpr:
		base := stb.BuildTypeSymbol(t.X)
		r, _ := GetOrAddPointer(base, stb.CurrentSymbolTable, stb.RootSymbolTable)
		result = r
	case *ast.ArrayType:
		result = &ArrayTypeSymbol{&TypeSymbol{Meths: NewSymbolTable(), Posits: new(vector.Vector)}, stb.BuildTypeSymbol(t.Elt)}
	case *ast.ChanType:
		result = &ChanTypeSymbol{&TypeSymbol{Meths: NewSymbolTable(), Posits: new(vector.Vector)}, stb.BuildTypeSymbol(t.Value)}
	case *ast.InterfaceType:
		result = &InterfaceTypeSymbol{&TypeSymbol{Meths: NewSymbolTable(), Posits: new(vector.Vector)}}
		for _, method := range t.Methods.List {

			if len(method.Names) == 0 {
				ft := stb.BuildTypeSymbol(method.Type)
				result.AddMethod(ft)
			}
			for _, name := range method.Names {
				ft := stb.BuildTypeSymbol(method.Type).(*FunctionTypeSymbol)
				toAdd := &FunctionSymbol{Obj: CopyObject(name), FunctionType: ft, Locals: nil, Posits: new(vector.Vector)}
				if stb.PositionsRegister {
					toAdd.Positions().Push(NewOccurence(method.Pos(), toAdd.Obj))
				}
				result.AddMethod(toAdd)
			}
		}
	case *ast.MapType:
		result = &MapTypeSymbol{&TypeSymbol{Meths: NewSymbolTable(), Posits: new(vector.Vector)}, stb.BuildTypeSymbol(t.Key), stb.BuildTypeSymbol(t.Value)}
	case *ast.Ident:
		fmt.Printf("IIIIIIIIII %v %p\n", t.Name(), t)
		if IsPredeclaredType(t.Obj.Name) {
			result = &TypeSymbol{Obj: t.Obj, Meths: NewSymbolTable(), Posits: new(vector.Vector)}
			return
		}

		if sym, found := stb.CurrentSymbolTable.FindSymbolByName(t.Obj.Name); found {

			result = sym.(ITypeSymbol)
			t.Obj = sym.Object()
			if stb.PositionsRegister {
				sym.Positions().Push(NewOccurence(t.Pos(), t.Obj))
			}
		} else {
			result = &ImportedType{&TypeSymbol{Obj: t.Obj, Meths: NewSymbolTable(), Posits: new(vector.Vector)}}
			if stb.PositionsRegister {
				result.Positions().Push(NewOccurence(t.Pos(), t.Obj))
			}
		}
	case *ast.FuncType:
		res := &FunctionTypeSymbol{&TypeSymbol{Meths: NewSymbolTable(), Posits: new(vector.Vector)}, NewSymbolTable(), NewSymbolTable(), NewSymbolTable()}
		if t.Params != nil {
			e_count := 0
			for _, field := range t.Params.List {
				ftype := stb.BuildTypeSymbol(field.Type)
				if len(field.Names) == 0 {
					res.Parameters.AddSymbol(&VariableSymbol{Obj: &ast.Object{4, field.Type.Pos(), "*unnamed" + strconv.Itoa(e_count) + "*"}, VariableType: ftype, Posits: new(vector.Vector)})
					e_count += 1
				}
				for _, name := range field.Names {
					toAdd := &VariableSymbol{Obj: CopyObject(name), VariableType: ftype, Posits: new(vector.Vector)}
					if stb.PositionsRegister {
						toAdd.Positions().Push(NewOccurence(name.Pos(), toAdd.Obj))
					}
					res.Parameters.AddSymbol(toAdd)
				}
			}
		}
		if t.Results != nil {
			e_count := 0
			for _, field := range t.Results.List {
				ftype := stb.BuildTypeSymbol(field.Type)
				if len(field.Names) == 0 {
					res.Results.AddSymbol(&VariableSymbol{Obj: &ast.Object{4, field.Type.Pos(), "*unnamed" + strconv.Itoa(e_count) + "*"}, VariableType: ftype, Posits: new(vector.Vector)})
					e_count += 1
				}
				for _, name := range field.Names {
					toAdd := &VariableSymbol{Obj: CopyObject(name), VariableType: ftype, Posits: new(vector.Vector)}
					if stb.PositionsRegister {
						toAdd.Positions().Push(NewOccurence(name.Pos(), toAdd.Obj))
					}
					res.Results.AddSymbol(toAdd)
				}
			}
		}
		result = res
	case *ast.StructType:
		res := &StructTypeSymbol{&TypeSymbol{Meths: NewSymbolTable(), Posits: new(vector.Vector)}, NewSymbolTable()}
		for _, field := range t.Fields.List {
			ftype := stb.BuildTypeSymbol(field.Type)
			if len(field.Names) == 0 {
				res.Fields.AddSymbol(ftype)
			}
			for _, name := range field.Names {
				toAdd := &VariableSymbol{Obj: CopyObject(name), VariableType: ftype, Posits: new(vector.Vector)}
				if stb.PositionsRegister {
					toAdd.Positions().Push(NewOccurence(name.Pos(), toAdd.Obj))
				}
				res.Fields.AddSymbol(toAdd)
			}
		}
		result = res
	case *ast.SelectorExpr:
		switch stb.Mode {
		case GLOBALS_MODE, TYPES_MODE, METHODS_MODE:
			pref := stb.BuildTypeSymbol(t.X)
			result = &ImportedType{&TypeSymbol{Obj: &ast.Object{Name: pref.Name() + "." + t.Sel.Name()}, Meths: NewSymbolTable(), Posits: new(vector.Vector)}}
		}
	}

	return
}
