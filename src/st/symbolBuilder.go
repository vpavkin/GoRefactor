package st

import (
"go/ast"
)

type SymbolType int

const (
	ARRAY_TYPE SymbolType = iota
	MAP_TYPE
	CHANNEL_TYPE
	BASIC_TYPE
	FUNCTION_TYPE
	POINTER_TYPE
	ALIAS_TYPE
	INTERFACE_TYPE
	STRUCT_TYPE
	UNRESOLVED_TYPE
	PACKAGE
	VARIABLE
	FUNCTION
)

func makeTypeSymbol(name string, packFrom *Package) *TypeSymbol {
	return &TypeSymbol{name, NewIdentSet(), NewSymbolTable(packFrom), NewPositionSet(), packFrom}
}

func MakeArrayType(name string, packFrom *Package, elemType ITypeSymbol, Len int) *ArrayTypeSymbol {
	return &ArrayTypeSymbol{makeTypeSymbol(name, packFrom), elemType, Len}
}

func MakeMapType(name string, packFrom *Package, keyType ITypeSymbol, elemType ITypeSymbol) *MapTypeSymbol {
	return &MapTypeSymbol{makeTypeSymbol(name, packFrom), keyType, elemType}
}

func MakeChannelType(name string, packFrom *Package, valueType ITypeSymbol) *ChanTypeSymbol {
	return &ChanTypeSymbol{makeTypeSymbol(name, packFrom), valueType}
}

func MakeBasicType(name string) *BasicTypeSymbol {
	return &BasicTypeSymbol{makeTypeSymbol(name, nil)}
}

func MakeFunctionType(name string, packFrom *Package) *FunctionTypeSymbol {
	return &FunctionTypeSymbol{makeTypeSymbol(name, packFrom), NewSymbolTable(packFrom), NewSymbolTable(packFrom), NewSymbolTable(packFrom)}
}

func MakePointerType(packFrom *Package, baseType ITypeSymbol) *PointerTypeSymbol {
	return &PointerTypeSymbol{makeTypeSymbol(NO_NAME, packFrom), baseType, nil}
}

func MakeAliasType(name string, packFrom *Package, baseType ITypeSymbol) *AliasTypeSymbol {
	return &AliasTypeSymbol{makeTypeSymbol(name, packFrom), baseType}
}

func MakeInterfaceType(name string, packFrom *Package) *InterfaceTypeSymbol {
	return &InterfaceTypeSymbol{makeTypeSymbol(name, packFrom)}
}

func MakeStructType(name string, packFrom *Package) *StructTypeSymbol {
	return &StructTypeSymbol{makeTypeSymbol(name, packFrom), NewSymbolTable(packFrom)}
}

func MakeUnresolvedType(name string, packFrom *Package, decl ast.Expr) *UnresolvedTypeSymbol {
	return &UnresolvedTypeSymbol{makeTypeSymbol(name, packFrom), decl}
}

func MakePackage(name string, packFrom *Package, shortPath string, pack *Package) *PackageSymbol {
	return &PackageSymbol{name, NewIdentSet(), shortPath, NewPositionSet(), pack, packFrom}
}

func MakeFunction(name string, packFrom *Package, fType ITypeSymbol) *FunctionSymbol {
	return &FunctionSymbol{name, NewIdentSet(), fType, nil, NewPositionSet(), packFrom,false}
}

func MakeVariable(name string, packFrom *Package, vType ITypeSymbol) *VariableSymbol {
	return &VariableSymbol{name, NewIdentSet(), vType,false, NewPositionSet(), packFrom}
}
