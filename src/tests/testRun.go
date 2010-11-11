package main

import (
	"fmt"
	//"go/parser"
	//"os"
	"program"
	//"packageParser"
	"st"
	//"container/vector"
)

func init(){
	
}
var curPack *st.Package;
var visited map[string]bool;
var countUnres int;

func checkTypesInSymbolTable(table *st.SymbolTable) {
	if table == nil {
		return
	}
	for sym := range table.Iter() {
		if uts, ok := sym.(*st.UnresolvedTypeSymbol); ok {
			fmt.Printf("Unresolved %v %v:%v\n", uts.Name(),uts.Declaration.Pos().Filename, uts.Declaration.Pos().Line);
			countUnres++;
		} else {
			//Start recursive walk
			checkType(sym)
		}
	}
}

func checkAliasTypeSymbol(t *st.AliasTypeSymbol) {
	if uts, ok := t.BaseType.(*st.UnresolvedTypeSymbol); ok {
		fmt.Printf("Unresolved %v %v:%v\n", uts.Name(),uts.Declaration.Pos().Filename, uts.Declaration.Pos().Line);
		countUnres++;
	} else {
		checkType(t.BaseType)
	}
}

func checkPointerTypeSymbol(t *st.PointerTypeSymbol) {
	if uts, ok := t.BaseType.(*st.UnresolvedTypeSymbol); ok {
		fmt.Printf("Unresolved %v %v:%v\n", uts.Name(),uts.Declaration.Pos().Filename, uts.Declaration.Pos().Line);
		countUnres++;
	} else {
		checkType(t.BaseType)
	}
}

func checkArrayTypeSymbol(t *st.ArrayTypeSymbol) {
	if uts, ok := t.ElemType.(*st.UnresolvedTypeSymbol); ok {
		fmt.Printf("Unresolved %v %v:%v\n", uts.Name(),uts.Declaration.Pos().Filename, uts.Declaration.Pos().Line);
		countUnres++;
	} else {
		checkType(t.ElemType)
	}
}

func checkStructTypeSymbol(t *st.StructTypeSymbol) {
	checkTypesInSymbolTable(t.Fields)
}

func checkInterfaceTypeSymbol(t *st.InterfaceTypeSymbol) {
	checkTypesInSymbolTable(t.Meths)
}

func checkMapTypeSymbol(t *st.MapTypeSymbol) {
	if uts, ok := t.KeyType.(*st.UnresolvedTypeSymbol); ok {
		fmt.Printf("Unresolved %v %v:%v\n", uts.Name(),uts.Declaration.Pos().Filename, uts.Declaration.Pos().Line);
		countUnres++;
	} else {
		checkType(t.KeyType)
	}
	
	if uts, ok := t.ValueType.(*st.UnresolvedTypeSymbol); ok {
		fmt.Printf("Unresolved %v %v:%v\n", uts.Name(),uts.Declaration.Pos().Filename, uts.Declaration.Pos().Line);
		countUnres++;
	} else {
		checkType(t.ValueType)
	}
}

func checkChanTypeSymbol(t *st.ChanTypeSymbol) {
	if uts, ok := t.ValueType.(*st.UnresolvedTypeSymbol); ok {
		fmt.Printf("Unresolved %v %v:%v\n", uts.Name(),uts.Declaration.Pos().Filename, uts.Declaration.Pos().Line);
		countUnres++;
	} else {
		checkType(t.ValueType)
	}
}

func checkFunctionTypeSymbol(t *st.FunctionTypeSymbol) {
	checkTypesInSymbolTable(t.Parameters)
	checkTypesInSymbolTable(t.Results)
	//fixTypesInSymbolTable(t.Reciever)
}
func checkVariableSymbol(t *st.VariableSymbol) {
	if t.Name() == "Rparen"{
		fmt.Printf("??????????????????Rparen %s\n", t.VariableType.Name())
	}
	if uts, ok := t.VariableType.(*st.UnresolvedTypeSymbol); ok {
		fmt.Printf("Unresolved %v %v:%v\n", uts.Name(),uts.Declaration.Pos().Filename, uts.Declaration.Pos().Line);
		countUnres++;
	} else {
		checkType(t.VariableType)
	}
}
func checkFunctionSymbol(t *st.FunctionSymbol) {
	if uts, ok := t.FunctionType.(*st.UnresolvedTypeSymbol); ok {
		fmt.Printf("Unresolved %v %v:%v\n", uts.Name(),uts.Declaration.Pos().Filename, uts.Declaration.Pos().Line);
		countUnres++;
	} else {
		checkType(t.FunctionType)
	}
}

//Fixes Type and its' subtypes recursively
func checkType(sym st.Symbol) {
	
	if sym == nil {
		fmt.Printf("ERROR: sym == nil. func fixType, typesVisitor.go\n")
	}
	
	fmt.Printf("%T\n",sym)
// 	on := ""
// 	if sym.Object() != nil{
// 		on = sym.Object().Name;
// 	}
// 	
// 	pn:="";
// 	if sym.PackageFrom() != nil{
// 		pn= sym.PackageFrom().AstPackage.Name;
// 	}
// 	
// 	
// 	fmt.Printf("%s: %s %T\n",pn, on ,sym);

	if st.IsPredeclaredIdentifier(sym.Name()) {
		return
	}
	
	if(sym.PackageFrom() != curPack){
		return;
	}
	if checkIsVisited(sym) {
		return
	}
	switch t := sym.(type) {
		case *st.AliasTypeSymbol:
			checkAliasTypeSymbol(t)
		case *st.PointerTypeSymbol:
			checkPointerTypeSymbol(t)
		case *st.ArrayTypeSymbol:
			checkArrayTypeSymbol(t)
		case *st.StructTypeSymbol:
			checkStructTypeSymbol(t)
		case *st.InterfaceTypeSymbol:
			checkInterfaceTypeSymbol(t)
		case *st.MapTypeSymbol:
			checkMapTypeSymbol(t)
		case *st.ChanTypeSymbol:
			checkChanTypeSymbol(t)
		case *st.FunctionTypeSymbol:
			checkFunctionTypeSymbol(t)
		case *st.VariableSymbol:
			checkVariableSymbol(t)
		case *st.FunctionSymbol:
			checkFunctionSymbol(t)
	}
}

func checkIsVisited(sym st.Symbol) bool {
	if _,ok := sym.(st.ITypeSymbol);!ok{
		return false;
	}
	symName := sym.Name()
	if v, ok := visited[symName]; ok && v { //Symbol already checked
		return true
	} else if _,ok := sym.(st.ITypeSymbol); symName!="" && ok { //Mark as checked
		visited[symName] = true
	}
	return false
}

func main() {


	p:=program.ParseProgram("/home/rulerr/GoRefactor/src");
	
	//print imports
	/*
	for	 _,pack := range p.Packages{
		fmt.Printf("%s :\n",pack.QualifiedPath);
		if !pack.IsGoPackage {
			for	f,v := range pack.Imports{
				fmt.Printf("	%s imports:\n",f);
				for	_,s := range *v{
					sym := s.(*st.PackageSymbol);
					fmt.Printf("		%s \"%s\"\n",sym.Obj.Name,sym.Path);
				}
			}
		}
	}*/
	/*
	fmt.Printf("------------------------------------------------------------\n");
	
	st := p.Packages["/home/rulerr/GoRefactor/src/st"].Symbols;
	vect := st.String();
	fmt.Println(*vect);
	*/
	
	for	 _,pack := range p.Packages{
		fmt.Printf("%s :\n",pack.QualifiedPath);
		visited = make(map[string]bool);
		curPack = pack
		checkTypesInSymbolTable(pack.Symbols);
	}
	
	fmt.Printf("%d symbols unresolved\n",countUnres);
	
	/*fmt.Printf("Methods:\n");
	
	for	 _,pack := range p.Packages{
		fmt.Printf("%s :\n",pack.QualifiedPath);
		for sym := range pack.Symbols.Iter(){
			switch ts:= sym.(type){
				case st.ITypeSymbol:
					if ts.Methods() != nil{
						for meth := range ts.Methods().Iter(){
							fmt.Printf("	%s\n",meth.String());
						}
					}
				case *st.FunctionSymbol:
					fmt.Printf("	%s\n",ts.String());
			}
		}
	}*/
	
}
