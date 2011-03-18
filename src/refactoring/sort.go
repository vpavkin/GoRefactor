package refactoring

import (
	"refactoring/utils"
	"refactoring/errors"
	"refactoring/program"
	"sort"
	"go/token"
	"go/ast"
	"os"
	"go/printer"
	"bytes"
	"unicode"
)

const DEFAULT_ORDER string = "cvtmf"
const CONST_ORDER_ENTRY int = 'c'
const VAR_ORDER_ENTRY int = 'v'
const TYPE_ORDER_ENTRY int = 't'
const METHOD_ORDER_ENTRY int = 'm'
const FUNCTION_ORDER_ENTRY int = 'f'
const IMPORT_ORDER_ENTRY int = 'i' //doesn't belong to order string

var (
	fullOrder                string
	groupMethodsByType       bool
	groupMethodsByVisibility bool
	sortImports              bool
)

// uses variables fullOrder,groupMethodsByType,groupMethodsByVisibility to work
type DeclCollection []ast.Decl

func (dc DeclCollection) Len() int {
	return len(dc)
}

func (dc DeclCollection) Less(i, j int) bool {
	di, dj := dc[i], dc[j]
	ti, tj := getDeclOrderEntry(di), getDeclOrderEntry(dj)
	if compareOrder(ti, tj) == -1 {
		return true
	}
	if compareOrder(ti, tj) == 1 {
		return false
	}
	//ti == tj
	ni, nj := getDeclName(di), getDeclName(dj)
	if ti == IMPORT_ORDER_ENTRY {
		if !sortImports {
			return false
		}
	}
	if ti == METHOD_ORDER_ENTRY {
		r := 0
		if groupMethodsByType {
			r = CompareMethodsByTypeName(di.(*ast.FuncDecl), dj.(*ast.FuncDecl))
		} else if groupMethodsByVisibility {
			r = CompareMethodsByVisibility(di.(*ast.FuncDecl), dj.(*ast.FuncDecl))
		}
		switch {
		case r < 0:
			return true
		case r > 0:
			return false
		}
	}
	return ni < nj
}

func (dc DeclCollection) Swap(i, j int) {
	temp := dc[i]
	dc[i] = dc[j]
	dc[j] = temp
}

type SpecCollection []ast.Spec

func (sc SpecCollection) Len() int {
	return len(sc)
}

func (dc SpecCollection) Less(i, j int) bool {
	di, dj := dc[i], dc[j]
	switch t := di.(type) {
	case *ast.ImportSpec:
		si := t
		sj := dj.(*ast.ImportSpec)
		return string(si.Path.Value) < string(sj.Path.Value)
	}
	return true
}

func (dc SpecCollection) Swap(i, j int) {
	temp := dc[i]
	dc[i] = dc[j]
	dc[j] = temp
}


func getEntryIndex(o int) int {
	for i, b := range fullOrder {
		if b == o {
			return i
		}
	}
	return -1
}

func compareOrder(o1 int, o2 int) int {
	i1 := getEntryIndex(o1)
	i2 := getEntryIndex(o2)
	switch {
	case i1 == i2:
		return 0
	case i1 > i2:
		return 1
	}
	return -1
}

func getDeclName(d ast.Decl) string {
	switch t := d.(type) {
	case *ast.GenDecl:
		if len(t.Specs) == 0 {
			return ""
		}
		switch tt := t.Specs[0].(type) {
		case *ast.ImportSpec:
			if tt.Name == nil {
				return string(tt.Path.Value)
			}
			return tt.Name.Name
		case *ast.ValueSpec:
			return tt.Names[0].Name //len(Names) > 0
		case *ast.TypeSpec:
			return tt.Name.Name
		}
	case *ast.FuncDecl:
		return t.Name.Name
	}
	panic("unknown case")
}

func getDeclOrderEntry(d ast.Decl) int {
	switch t := d.(type) {
	case *ast.GenDecl:
		switch t.Tok {
		case token.IMPORT:
			return IMPORT_ORDER_ENTRY
		case token.CONST:
			return CONST_ORDER_ENTRY
		case token.TYPE:
			return TYPE_ORDER_ENTRY
		case token.VAR:
			return VAR_ORDER_ENTRY
		}
	case *ast.FuncDecl:
		if t.Recv == nil {
			return FUNCTION_ORDER_ENTRY
		} else {
			return METHOD_ORDER_ENTRY
		}
	}
	panic("unknown decl type")
}
func getRecvTypeName(d *ast.FuncDecl) string {
	b := new(bytes.Buffer)
	printer.Fprint(b, token.NewFileSet(), d.Recv.List[0].Type)
	println(b.String())
	return b.String()
}

func CompareMethodsByTypeName(d1 *ast.FuncDecl, d2 *ast.FuncDecl) int {
	n1 := getRecvTypeName(d1)
	n2 := getRecvTypeName(d2)
	switch {
	case n1 < n2:
		return -1
	case n1 > n2:
		return 1
	}
	return 0
}

func CompareMethodsByVisibility(d1 *ast.FuncDecl, d2 *ast.FuncDecl) int {
	n1 := getDeclName(d1)
	n2 := getDeclName(d2)
	b1 := unicode.IsUpper(int(n1[0]))
	b2 := unicode.IsUpper(int(n2[0]))
	switch {
	case b1 && !b2:
		return 1
	case !b1 && b2:
		return -1
	}
	return 0
}

//only correct order input
func getFullOrder(order string) string {
	result := ""
	used := make(map[int]bool)
	for _, b := range order {
		result += string(b)
		used[b] = true
	}
	for _, b := range DEFAULT_ORDER {
		if _, ok := used[b]; !ok {
			result += string(b)
		}
	}
	return result
}

func isOrderEntry(b int) bool {
	switch b {
	case CONST_ORDER_ENTRY:
		fallthrough
	case VAR_ORDER_ENTRY:
		fallthrough
	case TYPE_ORDER_ENTRY:
		fallthrough
	case METHOD_ORDER_ENTRY:
		fallthrough
	case FUNCTION_ORDER_ENTRY:
		return true
	}
	return false
}

func isCorrectOrder(order string) bool {
	used := make(map[int]bool)
	for _, b := range order {
		if !isOrderEntry(b) {
			return false
		}
		if _, ok := used[b]; ok {
			return false
		}
		used[b] = true
	}
	return true
}
func CheckSortParameters(filename string, order string) (bool, *errors.GoRefactorError) {
	switch {
	case filename == "" || !utils.IsGoFile(filename):
		return false, errors.ArgumentError("filename", "It's not a valid go file name")
	case !isCorrectOrder(order):
		return false, errors.ArgumentError("order", "invalid order string, type \"goref sort\" to see usage")
	}
	return true, nil
}
func Sort(programTree *program.Program, filename string, _groupMethodsByType bool, _groupMethodsByVisibility bool, _sortImports bool, order string) (bool, *errors.GoRefactorError) {

	if ok, err := CheckSortParameters(filename, order); !ok {
		return false, err
	}
	pack, file := programTree.FindPackageAndFileByFilename(filename)
	if pack == nil {
		return false, errors.ArgumentError("filename", "Program packages don't contain file '"+filename+"'")
	}
	groupMethodsByType = _groupMethodsByType
	groupMethodsByVisibility = _groupMethodsByVisibility
	sortImports = _sortImports
	fullOrder = getFullOrder(order)
	decls := DeclCollection(file.Decls)
	if sortImports {
		for _, d := range decls {
			if gd, ok := d.(*ast.GenDecl); ok {
				if gd.Tok == token.IMPORT {
					sort.Sort(SpecCollection(gd.Specs))
				}
			}
		}
	}
	sort.Sort(decls)
	printer.Fprint(os.Stdout, token.NewFileSet(), file)
	return true, nil
}
