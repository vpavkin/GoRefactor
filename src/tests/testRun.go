package main

import (
	"fmt"
	"go/printer"
	"os"
	"program"
	//"packageParser"
	"st"
	//"container/vector"
	"path"
	
	"refactoring"
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
	table.ForEachNoLock(func(sym st.Symbol) {
		if uts, ok := sym.(*st.UnresolvedTypeSymbol); ok {
			fmt.Printf("Unresolved %v from %v\n", uts.Name(),uts.PackageFrom().AstPackage.Name);
			countUnres++;
		} else {
			//Start recursive walk
			checkType(sym)
		}
	})
}

func checkAliasTypeSymbol(t *st.AliasTypeSymbol) {
	if uts, ok := t.BaseType.(*st.UnresolvedTypeSymbol); ok {
		fmt.Printf("Unresolved %v from %v\n", uts.Name(),uts.PackageFrom().AstPackage.Name);
		countUnres++;
	} else {
		checkType(t.BaseType)
	}
}

func checkPointerTypeSymbol(t *st.PointerTypeSymbol) {
	if uts, ok := t.BaseType.(*st.UnresolvedTypeSymbol); ok {
		fmt.Printf("Unresolved %v from %v\n", uts.Name(),uts.PackageFrom().AstPackage.Name);
		countUnres++;
	} else {
		checkType(t.BaseType)
	}
}

func checkArrayTypeSymbol(t *st.ArrayTypeSymbol) {
	if uts, ok := t.ElemType.(*st.UnresolvedTypeSymbol); ok {
		fmt.Printf("Unresolved %v from %v\n", uts.Name(),uts.PackageFrom().AstPackage.Name);
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
		fmt.Printf("Unresolved %v from %v\n", uts.Name(),uts.PackageFrom().AstPackage.Name);
		countUnres++;
	} else {
		checkType(t.KeyType)
	}
	
	if uts, ok := t.ValueType.(*st.UnresolvedTypeSymbol); ok {
		fmt.Printf("Unresolved %v from %v\n", uts.Name(),uts.PackageFrom().AstPackage.Name);
		countUnres++;
	} else {
		checkType(t.ValueType)
	}
}

func checkChanTypeSymbol(t *st.ChanTypeSymbol) {
	if uts, ok := t.ValueType.(*st.UnresolvedTypeSymbol); ok {
		fmt.Printf("Unresolved %v from %v\n", uts.Name(),uts.PackageFrom().AstPackage.Name);
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
	//fmt.Printf("/// %s %s\n", t.Name(),t.PackageFrom().QualifiedPath)
	if uts, ok := t.VariableType.(*st.UnresolvedTypeSymbol); ok {
		fmt.Printf("Unresolved %v from %v\n", uts.Name(),uts.PackageFrom().AstPackage.Name);
		countUnres++;
	} else {
		checkType(t.VariableType)
	}
}
func checkFunctionSymbol(t *st.FunctionSymbol) {
	if uts, ok := t.FunctionType.(*st.UnresolvedTypeSymbol); ok {
		fmt.Printf("Unresolved %v from %v\n", uts.Name(),uts.PackageFrom().AstPackage.Name);
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
	
// 	if pr,ok := sym.(*st.BasicTypeSymbol); ok{
// 		//fmt.Printf("%T %v\n",pr.BaseType,pr.BaseType.Name());
// 		//fmt.Printf("%p %p\n",sym,pr);
// 	}
	
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


	p:=program.ParseProgram("/home/rulerr/GoRefactor/src",nil);
	
	//print imports
	/*
	for	 _,pack := range p.Packages{
		fmt.Printf("%s :\n",pack.QualifiedPast.NewOccurence(th);
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
	
// 	for	 _,pack := range p.Packages{
// 		fmt.Printf("%s :\n",pack.QualifiedPath);
// 		visited = make(map[string]bool);
// 		curPack = pack
// 		checkTypesInSymbolTable(pack.Symbols);
// 	}

	for _,sym := range p.IdentMap {
		if uts, ok := sym.(*st.UnresolvedTypeSymbol); ok {
			fmt.Printf("unresolved %v from %v\n", uts.Name(),uts.PackageFrom().AstPackage.Name);
			countUnres++;
		}
	}
	
	fmt.Printf("%d symbols unresolved\n",countUnres);
	
	sst := p.Packages["/home/rulerr/GoRefactor/src/st"].Symbols
	sst.ForEach(func (sym st.Symbol){
		fmt.Printf("%s(from %s):\n",sym.Name(),sym.PackageFrom().AstPackage.Name);
		for _,pos := range sym.Positions(){
			_,f:=path.Split(pos.Filename)
			fmt.Printf("\t[%s,%d,%d]\n",f,pos.Line,pos.Column)
		}
		
	})
	
	ssst := p.Packages["/home/rulerr/GoRefactor/src/program"].Symbols;
	vect := ssst.String();
	fmt.Println(*vect);
	
	if ok,err := refactoring.Rename(p,"/home/rulerr/GoRefactor/src/utils/utils.go",14,6,"IssssGoFIle");!ok{
		fmt.Println(err.Message);
	}else{
		cfg:=&printer.Config{printer.TabIndent,8,nil}
		cfg.Fprint(os.Stdout,p.Packages["/home/rulerr/GoRefactor/src/utils"].FileSet,p.Packages["/home/rulerr/GoRefactor/src/utils"].AstPackage.Files["/home/rulerr/GoRefactor/src/utils/utils.go"])
		cfg.Fprint(os.Stdout,p.Packages["/home/rulerr/GoRefactor/src/refactoring"].FileSet,p.Packages["/home/rulerr/GoRefactor/src/refactoring"].AstPackage.Files["/home/rulerr/GoRefactor/src/refactoring/rename.go"])
	}
	
	/*fmt.Printf("Methods:\n")
	
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
