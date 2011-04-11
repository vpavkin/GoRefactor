package refactoring

import (
	"refactoring/utils"
	"refactoring/errors"
	"refactoring/program"
	"refactoring/printerUtil"
	"sort"
	"go/token"
	"go/ast"
	//"os"
	"go/printer"
	"bytes"
	"unicode"

	"fmt"
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
type DeclCollection struct {
	Arr     []ast.Decl
	file    *ast.File
	fset    *token.FileSet
	tokFile *token.File
}

func (dc *DeclCollection) Len() int {
	return len(dc.Arr)
}

func (dc *DeclCollection) Less(i, j int) bool {
	di, dj := dc.Arr[i], dc.Arr[j]
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

func (dc *DeclCollection) Swap(i, j int) {
	switch {
	case j == i:
		return
	case i > j:
		//make i < j
		t := i
		i = j
		j = t
	}

	diff := int(dc.Arr[j].End()-dc.Arr[j].Pos()) - int(dc.Arr[i].End()-dc.Arr[i].Pos())

	//lines
	lines := printerUtil.GetLines(dc.tokFile)
	newLines := make([]int, len(lines))
	iLines, iFline := getRangeLinesAtLeastOne(dc.tokFile, dc.Arr[i].Pos(), dc.Arr[i].End(), dc.tokFile.Size())
	jLines, jFline := getRangeLinesAtLeastOne(dc.tokFile, dc.Arr[j].Pos(), dc.Arr[j].End(), dc.tokFile.Size())
	iLineBefore, jLineBefore := lines[iFline-2], lines[jFline-2]

	jLineMod := iLineBefore - jLineBefore
	iLineMod := diff - jLineMod

	fmt.Printf("iLines: %v\n", iLines)
	fmt.Printf("jLines: %v\n", jLines)

	fmt.Printf("before: %v\n", lines)
	for k := 0; k < iFline-1; k++ {
		newLines[k] = lines[k]
	}
	mod := iFline - 1
	fmt.Printf("step 1: %v\n", newLines)

	for k := 0; k < len(jLines); k++ {
		newLines[k+mod] = jLines[k] + jLineMod
	}
	mod += len(jLines)
	fmt.Printf("step 2: %v\n", newLines)

	for k := 0; k < jFline-iFline-len(iLines); k++ {
		newLines[k+mod] = lines[iFline-1+len(iLines)+k] + diff
	}
	mod += jFline - iFline - len(iLines)
	fmt.Printf("step 3: %v\n", newLines)

	for k := 0; k < len(iLines) && k+mod < len(lines); k++ {
		newLines[k+mod] += iLines[k] + iLineMod
	}
	mod += len(iLines)
	fmt.Printf("step 4: %v\n", newLines)

	if mod < len(lines)-1 {
		for k := jFline - 1 + len(jLines); k < len(lines); k++ {
			newLines[k] = lines[k]
		}
	}

	if !dc.tokFile.SetLines(newLines) {
		panic("couldn't set lines for file " + tokFile.Name())
	}
	fmt.Printf("after : %v\n", newLines)
	//positions
	jmod := int(dc.Arr[i].Pos() - dc.Arr[j].Pos())
	imod := -jmod + diff

	fmt.Printf("i,j linemods: %d,%d\n", iLineMod, jLineMod)
	fmt.Printf("i,j mods: %d,%d\n", imod, jmod)
	fmt.Printf("diff: %d\n", diff)

	ist, iend := dc.Arr[i].Pos(), dc.Arr[i].End()
	jst, jend := dc.Arr[j].Pos(), dc.Arr[j].End()
	fmt.Printf("before ist,iend,jst,jend : %d,%d,%d,%d\n", ist, iend, jst, jend)

	for _, cg := range dc.file.Comments {
		switch {
		//inside iDecl
		case cg.Pos() >= dc.Arr[i].Pos() && cg.End() <= dc.Arr[i].End():
			printerUtil.FixPositions(0, imod, cg, true)
			//between decls
		case cg.Pos() >= dc.Arr[i].End() && cg.End() <= dc.Arr[j].Pos():
			printerUtil.FixPositions(0, diff, cg, true)
			//inside jDecl
		case cg.Pos() >= dc.Arr[j].Pos() && cg.End() <= dc.Arr[j].End():
			printerUtil.FixPositions(0, jmod, cg, true)
		case cg.End() <= dc.Arr[i].Pos() || cg.Pos() >= dc.Arr[j].End():
			fmt.Printf("not affected \"%s\" pos,end : %d,%d\n", cg.List[0].Text, cg.Pos(), cg.End())
		default:
			fmt.Printf("WTF pos,end : %d,%d\n", cg.Pos(), cg.End())
			panic("<<<<")
		}
	}

	cc := CommentCollection(dc.file.Comments)
	sort.Sort(cc)

	printerUtil.FixPositions(0, jmod, dc.Arr[j], false)
	for k := i + 1; k < j; k++ {
		printerUtil.FixPositions(0, diff, dc.Arr[k], false)
	}
	printerUtil.FixPositions(0, imod, dc.Arr[i], false)

	// 	visited := printerUtil.FixPositionsInRange(iend, jst, diff, dc.file, true, map[ast.Node]bool{})
	// 	visited = printerUtil.FixPositionsInRange(ist, iend, imod, dc.file, true, visited)
	// 	visited = printerUtil.FixPositionsInRange(jst, jend, jmod, dc.file, true, visited)

	fmt.Printf("after ist,iend,jst,jend : %d,%d,%d,%d\n", dc.Arr[i].Pos(), dc.Arr[i].End(), dc.Arr[j].Pos(), dc.Arr[j].End())

	//swap
	temp := dc.Arr[i]
	dc.Arr[i] = dc.Arr[j]
	dc.Arr[j] = temp
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

type CommentCollection []*ast.CommentGroup

func (sc CommentCollection) Len() int {
	return len(sc)
}

func (dc CommentCollection) Less(i, j int) bool {
	di, dj := dc[i], dc[j]
	return di.Pos() < dj.Pos()
}

func (dc CommentCollection) Swap(i, j int) {
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
	fset := pack.FileSet
	tokFile := printerUtil.GetFileFromFileSet(fset, filename)

	groupMethodsByType = _groupMethodsByType
	groupMethodsByVisibility = _groupMethodsByVisibility
	sortImports = _sortImports
	fullOrder = getFullOrder(order)
	decls := &DeclCollection{file.Decls, file, fset, tokFile}
	if sortImports {
		for _, d := range decls.Arr {
			if gd, ok := d.(*ast.GenDecl); ok {
				if gd.Tok == token.IMPORT {
					sort.Sort(SpecCollection(gd.Specs))
				}
			}
		}
	}

	printDecls(tokFile, file)
	//test
	//decls.Swap(2, decls.Len()-1)
	sort.Sort(decls)
	printDecls(tokFile, file)
	//printer.Fprint(os.Stdout, fset, file)
	return true, nil
}
