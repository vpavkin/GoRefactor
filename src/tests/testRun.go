package main

import (
	"fmt"
	//"go/parser"
	//"os"
	"program"
	//"packageParser"
	"st"
)

func init(){
	
}

func main() {

	/*
	ps,_ := parser.ParseDir("/home/rulerr/GoRefactor/src/tests",GoFilter,parser.ParseComments);
	
	pack := ps["testtest"];

	st,_ := packageParser.ParsePackage(pack);
	
	vect := st.String()
	
	fmt.Println(vect);
	*/
	
	p := program.ParseProgram("/home/rulerr/GoRefactor/src");
	
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
	}
	fmt.Printf("------------------------------------------------------------\n");
	/*st := p.Packages["/home/rulerr/GoRefactor/src/st"].Symbols;
	vect := st.String();
	fmt.Println(*vect);*/
	i:=0;
	for	 _,pack := range p.Packages{
		fmt.Printf("%s :\n",pack.QualifiedPath);
		for sym := range pack.Symbols.Iter(){
			name := sym.Name();
			if _,ok := sym.(*st.UnresolvedTypeSymbol);ok{
				fmt.Printf("	%s\n",name);
				i++;
			}
		}
	}
	fmt.Printf("%d symbols unresolved\n",i);
	
}
