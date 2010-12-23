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

func getPackage(scope *SymbolTable) *Package {
	if scope != nil {
		return scope.Package
	}
	return nil
}
func makeTypeSymbol(name string, scope *SymbolTable) *TypeSymbol {
	return &TypeSymbol{name, NewIdentSet(), NewSymbolTable(getPackage(scope)), NewPositionSet(), scope}
}

func MakeArrayType(name string, scope *SymbolTable, elemType ITypeSymbol, Len int) *ArrayTypeSymbol {
	return &ArrayTypeSymbol{makeTypeSymbol(name, scope), elemType, Len}
}

func MakeMapType(name string, scope *SymbolTable, keyType ITypeSymbol, elemType ITypeSymbol) *MapTypeSymbol {
	return &MapTypeSymbol{makeTypeSymbol(name, scope), keyType, elemType}
}

func MakeChannelType(name string, scope *SymbolTable, valueType ITypeSymbol, dir ast.ChanDir) *ChanTypeSymbol {
	return &ChanTypeSymbol{makeTypeSymbol(name, scope), dir, valueType}
}

func MakeBasicType(name string, scope *SymbolTable) *BasicTypeSymbol {
	return &BasicTypeSymbol{makeTypeSymbol(name, scope)}
}

func MakeFunctionType(name string, scope *SymbolTable) *FunctionTypeSymbol {
	return &FunctionTypeSymbol{makeTypeSymbol(name, scope), NewSymbolTable(getPackage(scope)), NewSymbolTable(getPackage(scope)), NewSymbolTable(getPackage(scope))}
}

func MakePointerType(scope *SymbolTable, baseType ITypeSymbol) *PointerTypeSymbol {
	return &PointerTypeSymbol{makeTypeSymbol(NO_NAME, scope), baseType, nil}
}

func MakeAliasType(name string, scope *SymbolTable, baseType ITypeSymbol) *AliasTypeSymbol {
	return &AliasTypeSymbol{makeTypeSymbol(name, scope), baseType}
}

func MakeInterfaceType(name string, scope *SymbolTable) *InterfaceTypeSymbol {
	return &InterfaceTypeSymbol{makeTypeSymbol(name, scope)}
}

func MakeStructType(name string, scope *SymbolTable) *StructTypeSymbol {
	return &StructTypeSymbol{makeTypeSymbol(name, scope), NewSymbolTable(getPackage(scope))}
}

func MakeUnresolvedType(name string, scope *SymbolTable, decl ast.Expr) *UnresolvedTypeSymbol {
	return &UnresolvedTypeSymbol{makeTypeSymbol(name, scope), decl}
}

func MakePackage(name string, scope *SymbolTable, shortPath string, pack *Package) *PackageSymbol {
	return &PackageSymbol{name, NewIdentSet(), shortPath, NewPositionSet(), pack, scope}
}

func MakeFunction(name string, scope *SymbolTable, fType ITypeSymbol) *FunctionSymbol {
	return &FunctionSymbol{name, NewIdentSet(), fType, nil, NewPositionSet(), scope, false}
}

func MakeVariable(name string, scope *SymbolTable, vType ITypeSymbol) *VariableSymbol {
	return &VariableSymbol{name, NewIdentSet(), vType, false, NewPositionSet(), scope}
}
