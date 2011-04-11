package utils

import (
	"os"
	"path"
	"strings"
	"go/token"
	"go/ast"
	"go/printer"
	"bytes"
	"io/ioutil"
	"unicode"
	"bufio"
)

func GoFilter(f *os.FileInfo) bool {
	return IsGoFile(f.Name)
}

func IsGoFile(fileName string) bool {
	return (path.Ext(fileName) == ".go") && !(strings.HasSuffix(fileName, "_test.go"))
}

func ComparePosWithinFile(pos1 token.Position, pos2 token.Position) int {
	switch {
	case pos1.Line > pos2.Line:
		fallthrough
	case pos1.Line == pos2.Line && pos1.Column > pos2.Column:
		return 1
	case pos1.Line == pos2.Line && pos1.Column == pos2.Column:
		return 0
	case pos1.Line == pos2.Line && pos1.Column < pos2.Column:
		fallthrough
	case pos1.Line < pos2.Line:
		return -1
	}
	panic("invalid positions")
}

func CopyExprList(list []ast.Expr) []ast.Expr {
	if list == nil {
		return nil
	}
	res := make([]ast.Expr, len(list))
	for i, stmt := range list {
		res[i] = CopyAstNode(stmt).(ast.Expr)
	}
	return res
}
func CopyStmtList(list []ast.Stmt) []ast.Stmt {
	if list == nil {
		return nil
	}
	res := make([]ast.Stmt, len(list))
	for i, stmt := range list {
		res[i] = CopyAstNode(stmt).(ast.Stmt)
	}
	return res
}
func CopyIdentList(list []*ast.Ident) []*ast.Ident {
	if list == nil {
		return nil
	}
	res := make([]*ast.Ident, len(list))
	for i, stmt := range list {
		res[i] = CopyAstNode(stmt).(*ast.Ident)
	}
	return res
}
func CopyFieldList(list []*ast.Field) []*ast.Field {
	if list == nil {
		return nil
	}
	res := make([]*ast.Field, len(list))
	for i, stmt := range list {
		res[i] = CopyAstNode(stmt).(*ast.Field)
	}
	return res
}
func CopySpecList(list []ast.Spec) []ast.Spec {
	if list == nil {
		return nil
	}
	res := make([]ast.Spec, len(list))
	for i, stmt := range list {
		res[i] = CopyAstNode(stmt).(ast.Spec)
	}
	return res
}
func CopyAstNode(node ast.Node) ast.Node {
	if node == nil || IsNullReally(node) {
		return nil
	}
	switch t := node.(type) {
	case *ast.ArrayType:
		Len, _ := CopyAstNode(t.Len).(ast.Expr)
		Elt, _ := CopyAstNode(t.Elt).(ast.Expr)
		return &ast.ArrayType{t.Lbrack, Len, Elt}
	case *ast.AssignStmt:
		return &ast.AssignStmt{CopyExprList(t.Lhs), t.TokPos, t.Tok, CopyExprList(t.Rhs)}
	case *ast.BasicLit:
		value := make([]byte, len(t.Value))
		copy(value, t.Value)
		return &ast.BasicLit{t.ValuePos, t.Kind, value}
	case *ast.BinaryExpr:
		return &ast.BinaryExpr{CopyAstNode(t.X).(ast.Expr), t.OpPos, t.Op, CopyAstNode(t.Y).(ast.Expr)}
	case *ast.BlockStmt:
		return &ast.BlockStmt{t.Lbrace, CopyStmtList(t.List), t.Rbrace}
	case *ast.BranchStmt:
		Label, _ := CopyAstNode(t.Label).(*ast.Ident)
		return &ast.BranchStmt{t.TokPos, t.Tok, Label}
	case *ast.CallExpr:
		return &ast.CallExpr{CopyAstNode(t.Fun).(ast.Expr), t.Lparen, CopyExprList(t.Args), t.Ellipsis, t.Rparen}
	case *ast.CaseClause:
		return &ast.CaseClause{t.Case, CopyExprList(t.Values), t.Colon, CopyStmtList(t.Body)}
	case *ast.ChanType:
		return &ast.ChanType{t.Begin, t.Dir, CopyAstNode(t.Value).(ast.Expr)}
	case *ast.CommClause:
		Comm, _ := CopyAstNode(t.Comm).(ast.Stmt)
		return &ast.CommClause{t.Case, Comm, t.Colon, CopyStmtList(t.Body)}
	case *ast.Comment:
		text := make([]byte, len(t.Text))
		copy(text, t.Text)
		return &ast.Comment{t.Slash, text}
	case *ast.CommentGroup:
		if t == nil {
			return t
		}
		text := make([]*ast.Comment, len(t.List))
		copy(text, t.List)
		return &ast.CommentGroup{text}
	case *ast.CompositeLit:
		Type, _ := CopyAstNode(t.Type).(ast.Expr)
		return &ast.CompositeLit{Type, t.Lbrace, CopyExprList(t.Elts), t.Rbrace}
	case *ast.DeclStmt:
		return &ast.DeclStmt{CopyAstNode(t.Decl).(ast.Decl)}
	case *ast.DeferStmt:
		return &ast.DeferStmt{t.Defer, CopyAstNode(t.Call).(*ast.CallExpr)}
	case *ast.Ellipsis:
		Elt, _ := CopyAstNode(t.Elt).(ast.Expr)
		return &ast.Ellipsis{t.Ellipsis, Elt}
	case *ast.EmptyStmt:
		return &ast.EmptyStmt{t.Semicolon}
	case *ast.ExprStmt:
		return &ast.ExprStmt{CopyAstNode(t.X).(ast.Expr)}
	case *ast.Field:
		Doc, _ := CopyAstNode(t.Doc).(*ast.CommentGroup)
		Tag, _ := CopyAstNode(t.Tag).(*ast.BasicLit)
		Comment, _ := CopyAstNode(t.Comment).(*ast.CommentGroup)
		return &ast.Field{Doc, CopyIdentList(t.Names), CopyAstNode(t.Type).(ast.Expr), Tag, Comment}
	case *ast.FieldList:
		return &ast.FieldList{t.Opening, CopyFieldList(t.List), t.Closing}
	//case *ast.File:
	case *ast.ForStmt:
		Init, _ := CopyAstNode(t.Init).(ast.Stmt)
		Cond, _ := CopyAstNode(t.Cond).(ast.Expr)
		Post, _ := CopyAstNode(t.Post).(ast.Stmt)
		return &ast.ForStmt{t.For, Init, Cond, Post, CopyAstNode(t.Body).(*ast.BlockStmt)}
	case *ast.FuncDecl:
		Doc, _ := CopyAstNode(t.Doc).(*ast.CommentGroup)
		Recv, _ := CopyAstNode(t.Recv).(*ast.FieldList)
		Body, _ := CopyAstNode(t.Body).(*ast.BlockStmt)
		return &ast.FuncDecl{Doc, Recv, CopyAstNode(t.Name).(*ast.Ident), CopyAstNode(t.Type).(*ast.FuncType), Body}
	case *ast.FuncLit:
		return &ast.FuncLit{CopyAstNode(t.Type).(*ast.FuncType), CopyAstNode(t.Body).(*ast.BlockStmt)}
	case *ast.FuncType:
		Results, _ := CopyAstNode(t.Results).(*ast.FieldList)
		return &ast.FuncType{t.Func, CopyAstNode(t.Params).(*ast.FieldList), Results}
	case *ast.GenDecl:
		Doc, _ := CopyAstNode(t.Doc).(*ast.CommentGroup)
		return &ast.GenDecl{Doc, t.TokPos, t.Tok, t.Lparen, CopySpecList(t.Specs), t.Rparen}
	case *ast.GoStmt:
		return &ast.GoStmt{t.Go, CopyAstNode(t.Call).(*ast.CallExpr)}
	case *ast.Ident:
		return &ast.Ident{t.NamePos, t.Name + "", nil}
	case *ast.IfStmt:
		Init, _ := CopyAstNode(t.Init).(ast.Stmt)
		Else, _ := CopyAstNode(t.Else).(ast.Stmt)
		return &ast.IfStmt{t.If, Init, CopyAstNode(t.Cond).(ast.Expr), CopyAstNode(t.Body).(*ast.BlockStmt), Else}
	case *ast.ImportSpec:
		Doc, _ := CopyAstNode(t.Doc).(*ast.CommentGroup)
		Name, _ := CopyAstNode(t.Name).(*ast.Ident)
		Comment, _ := CopyAstNode(t.Comment).(*ast.CommentGroup)
		return &ast.ImportSpec{Doc, Name, CopyAstNode(t.Path).(*ast.BasicLit), Comment}
	case *ast.IncDecStmt:
		return &ast.IncDecStmt{CopyAstNode(t.X).(ast.Expr), t.TokPos, t.Tok}
	case *ast.IndexExpr:
		return &ast.IndexExpr{CopyAstNode(t.X).(ast.Expr), t.Lbrack, CopyAstNode(t.Index).(ast.Expr), t.Rbrack}
	case *ast.InterfaceType:
		return &ast.InterfaceType{t.Interface, CopyAstNode(t.Methods).(*ast.FieldList), t.Incomplete}
	case *ast.KeyValueExpr:
		return &ast.KeyValueExpr{CopyAstNode(t.Key).(ast.Expr), t.Colon, CopyAstNode(t.Value).(ast.Expr)}
	case *ast.LabeledStmt:
		return &ast.LabeledStmt{CopyAstNode(t.Label).(*ast.Ident), t.Colon, CopyAstNode(t.Stmt).(ast.Stmt)}
	case *ast.MapType:
		return &ast.MapType{t.Map, CopyAstNode(t.Key).(ast.Expr), CopyAstNode(t.Value).(ast.Expr)}
	//case *ast.Package:
	case *ast.ParenExpr:
		return &ast.ParenExpr{t.Lparen, CopyAstNode(t.X).(ast.Expr), t.Rparen}
	case *ast.RangeStmt:
		Value, _ := CopyAstNode(t.Value).(ast.Expr)
		return &ast.RangeStmt{t.For, CopyAstNode(t.Key).(ast.Expr), Value, t.TokPos, t.Tok, CopyAstNode(t.X).(ast.Expr), CopyAstNode(t.Body).(*ast.BlockStmt)}
	case *ast.ReturnStmt:
		return &ast.ReturnStmt{t.Return, CopyExprList(t.Results)}
	case *ast.SelectStmt:
		return &ast.SelectStmt{t.Select, CopyAstNode(t.Body).(*ast.BlockStmt)}
	case *ast.SelectorExpr:
		return &ast.SelectorExpr{CopyAstNode(t.X).(ast.Expr), CopyAstNode(t.Sel).(*ast.Ident)}
	case *ast.SendStmt:
		return &ast.SendStmt{CopyAstNode(t.Chan).(ast.Expr), t.Arrow, CopyAstNode(t.Value).(ast.Expr)}
	case *ast.SliceExpr:
		High, _ := CopyAstNode(t.High).(ast.Expr)
		Low, _ := CopyAstNode(t.Low).(ast.Expr)
		return &ast.SliceExpr{CopyAstNode(t.X).(ast.Expr), t.Lbrack, Low, High, t.Rbrack}
	case *ast.StarExpr:
		return &ast.StarExpr{t.Star, CopyAstNode(t.X).(ast.Expr)}
	case *ast.StructType:
		return &ast.StructType{t.Struct, CopyAstNode(t.Fields).(*ast.FieldList), t.Incomplete}
	case *ast.SwitchStmt:
		Init, _ := CopyAstNode(t.Init).(ast.Stmt)
		Tag, _ := CopyAstNode(t.Tag).(ast.Expr)
		return &ast.SwitchStmt{t.Switch, Init, Tag, CopyAstNode(t.Body).(*ast.BlockStmt)}
	case *ast.TypeAssertExpr:
		Type, _ := CopyAstNode(t.Type).(ast.Expr)
		return &ast.TypeAssertExpr{CopyAstNode(t.X).(ast.Expr), Type}
	case *ast.TypeCaseClause:
		return &ast.TypeCaseClause{t.Case, CopyExprList(t.Types), t.Colon, CopyStmtList(t.Body)}
	case *ast.TypeSpec:
		Doc, _ := CopyAstNode(t.Doc).(*ast.CommentGroup)
		Comment, _ := CopyAstNode(t.Comment).(*ast.CommentGroup)
		return &ast.TypeSpec{Doc, CopyAstNode(t.Name).(*ast.Ident), CopyAstNode(t.Type).(ast.Expr), Comment}
	case *ast.TypeSwitchStmt:
		Init, _ := CopyAstNode(t.Init).(ast.Stmt)
		return &ast.TypeSwitchStmt{t.Switch, Init, CopyAstNode(t.Assign).(ast.Stmt), CopyAstNode(t.Body).(*ast.BlockStmt)}
	case *ast.UnaryExpr:
		return &ast.UnaryExpr{t.OpPos, t.Op, CopyAstNode(t.X).(ast.Expr)}
	case *ast.ValueSpec:
		Doc, _ := CopyAstNode(t.Doc).(*ast.CommentGroup)
		Type, _ := CopyAstNode(t.Type).(ast.Expr)
		Comment, _ := CopyAstNode(t.Comment).(*ast.CommentGroup)
		return &ast.ValueSpec{Doc, CopyIdentList(t.Names), Type, CopyExprList(t.Values), Comment}
	}
	panic("can't copy node")
}

func IsNullReally(node ast.Node) bool {
	switch t := node.(type) {
	case *ast.ArrayType:
		if t == nil {
			return true
		}
	case *ast.AssignStmt:
		if t == nil {
			return true
		}
	case *ast.BasicLit:
		if t == nil {
			return true
		}
	case *ast.BinaryExpr:
		if t == nil {
			return true
		}
	case *ast.BlockStmt:
		if t == nil {
			return true
		}
	case *ast.BranchStmt:
		if t == nil {
			return true
		}
	case *ast.CallExpr:
		if t == nil {
			return true
		}
	case *ast.CaseClause:
		if t == nil {
			return true
		}
	case *ast.ChanType:
		if t == nil {
			return true
		}
	case *ast.CommClause:
		if t == nil {
			return true
		}
	case *ast.Comment:
		if t == nil {
			return true
		}
	case *ast.CommentGroup:
		if t == nil {
			return true
		}
	case *ast.CompositeLit:
		if t == nil {
			return true
		}
	case *ast.DeferStmt:
		if t == nil {
			return true
		}
	case *ast.Ellipsis:
		if t == nil {
			return true
		}
	case *ast.ExprStmt:
		if t == nil {
			return true
		}
	case *ast.Field:
		if t == nil {
			return true
		}
	case *ast.FieldList:
		if t == nil {
			return true
		}
	case *ast.ForStmt:
		if t == nil {
			return true
		}
	case *ast.FuncDecl:
		if t == nil {
			return true
		}
	case *ast.FuncLit:
		if t == nil {
			return true
		}
	case *ast.FuncType:
		if t == nil {
			return true
		}
	case *ast.GenDecl:
		if t == nil {
			return true
		}
	case *ast.GoStmt:
		if t == nil {
			return true
		}
	case *ast.Ident:
		if t == nil {
			return true
		}
	case *ast.IfStmt:
		if t == nil {
			return true
		}
	case *ast.ImportSpec:
		if t == nil {
			return true
		}
	case *ast.IncDecStmt:
		if t == nil {
			return true
		}
	case *ast.IndexExpr:
		if t == nil {
			return true
		}
	case *ast.InterfaceType:
		if t == nil {
			return true
		}
	case *ast.KeyValueExpr:
		if t == nil {
			return true
		}
	case *ast.LabeledStmt:
		if t == nil {
			return true
		}
	case *ast.MapType:
		if t == nil {
			return true
		}
	case *ast.ParenExpr:
		if t == nil {
			return true
		}
	case *ast.RangeStmt:
		if t == nil {
			return true
		}
	case *ast.ReturnStmt:
		if t == nil {
			return true
		}
	case *ast.SelectStmt:
		if t == nil {
			return true
		}
	case *ast.SelectorExpr:
		if t == nil {
			return true
		}
	case *ast.SendStmt:
		if t == nil {
			return true
		}
	case *ast.SliceExpr:
		if t == nil {
			return true
		}
	case *ast.StarExpr:
		if t == nil {
			return true
		}
	case *ast.StructType:
		if t == nil {
			return true
		}
	case *ast.SwitchStmt:
		if t == nil {
			return true
		}
	case *ast.TypeAssertExpr:
		if t == nil {
			return true
		}
	case *ast.TypeCaseClause:
		if t == nil {
			return true
		}
	case *ast.TypeSpec:
		if t == nil {
			return true
		}
	case *ast.TypeSwitchStmt:
		if t == nil {
			return true
		}
	case *ast.ValueSpec:
		if t == nil {
			return true
		}
	}
	return false
}

// refactoring project functions

func loadConfig(configPath string) []string {
	fd, err := os.Open(configPath, os.O_RDONLY, 0)
	if err != nil {
		panic("Couldn't open config \"" + configPath + "\": " + err.String())
	}
	defer fd.Close()

	res := []string{}

	reader := bufio.NewReader(fd)
	for {

		str, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		res = append(res, (str[:len(str)-1]))

	}
	//fmt.Printf("%s:\n%v\n", configPath, res)

	return res
}

func getInfo(projectDir string, pa string) (sources map[string]string, specialPackages map[string][]string, ok bool) {
	f, err := os.Open(pa, os.O_RDONLY, 0)
	if err != nil {
		panic("couldn't open goref.cfg")
	}
	d, err := ioutil.ReadAll(f)
	if err != nil {
		panic("couldn't read from goref.cfg")
	}
	data := string(d)
	//sources
	i := strings.Index(data, ".packages")
	if i == -1 {
		panic("couldn't find .package field in goref.cfg")
	}
	st := i + len(".packages") + 1

	end := strings.Index(data[st:], ".") + st
	println(end)
	if end == -1 {
		end = len(data)
	}
	end--

	packages := strings.Split(data[st:end], "\n", -1)
	sources = make(map[string]string)
	for _, pack := range packages {
		println(st, end)
		println(data[st:end])
		realpath, goPath := "", ""
		i := 0
		for ; i < len(pack) && !unicode.IsSpace(int(pack[i])); i++ {
			realpath += string(pack[i])
		}
		for ; i < len(pack) && unicode.IsSpace(int(pack[i])); i++ {

		}
		for ; i < len(pack) && !unicode.IsSpace(int(pack[i])); i++ {
			goPath += string(pack[i])
		}
		if goPath == "" || realpath == "" || i < len(pack) {
			panic(".package field has invalid format " + goPath + " " + realpath)
		}
		sources[path.Join(projectDir, realpath)] = goPath
	}

	//externSources
	i = strings.Index(data, ".externPackages")
	if i >= 0 {
		st = i + len(".externPackages") + 1

		end = strings.Index(data[st:], ".") + st
		if end == -1 {
			end = len(data)
		}
		end--

		packages = strings.Split(data[st:end], "\n", -1)
		for _, pack := range packages {
			realpath, goPath := "", ""
			i := 0
			for ; i < len(pack) && !unicode.IsSpace(int(pack[i])); i++ {
				realpath += string(pack[i])
			}
			for ; i < len(pack) && unicode.IsSpace(int(pack[i])); i++ {

			}
			for ; i < len(pack) && !unicode.IsSpace(int(pack[i])); i++ {
				goPath += string(pack[i])
			}
			if goPath == "" || realpath == "" || i < len(pack) {
				panic(".externPackages field has invalid format " + goPath + " " + realpath)
			}
			sources[realpath] = goPath
		}
	}
	//specialPackages
	specialPackages = make(map[string][]string)
	i = strings.Index(data, ".specialPackages")
	if i != -1 {

		st := i + len(".specialPackages") + 1

		end := strings.Index(data[st:], ".")
		if end == -1 {
			end = len(data)
		}
		end--
		packages := strings.Split(data[st:end], "\n", -1)
		for _, pack := range packages {
			specialPackages[pack] = loadConfig(path.Join(projectDir, pack) + ".cfg")
		}
	}

	return sources, specialPackages, true
}

func getProjectInfo(filename string) (projectDir string, sources map[string]string, specialPackages map[string][]string, ok bool) {
	projectDir, _ = path.Split(filename)
	projectDir = projectDir[:len(projectDir)-1]
	projectDir, _ = path.Split(projectDir)
	for {
		projectDir = projectDir[:len(projectDir)-1]
		if projectDir == "" {
			return "", nil, nil, false
		}
		//fmt.Println(projectDir)
		fd, _ := os.Open(projectDir, os.O_RDONLY, 0)

		list, _ := fd.Readdir(-1)

		for i := 0; i < len(list); i++ {
			d := &list[i]
			if d.Name == "goref.cfg" {
				srcs, specialPacks, ok := getInfo(projectDir, path.Join(projectDir, d.Name))
				if !ok {
					return "", nil, nil, false
				}

				return projectDir, srcs, specialPacks, true
			}
		}
		projectDir, _ = path.Split(projectDir)
		fd.Close()
	}
	return "", nil, nil, false
}

func GetProjectInfo(filename string) (projectDir string, sources map[string]string, specialPackages map[string][]string, ok bool) {
	return getProjectInfo(filename)
}

func getLineOffsets(s string) []int {
	res := []int{}
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			res = append(res, i)
		}
	}
	return res
}

func GetNodeLength(node ast.Node) (length int, lineOffsets []int) {
	b := bytes.NewBuffer([]byte{})
	printer.Fprint(b, token.NewFileSet(), node)
	s := b.String()
	lineOffsets = getLineOffsets(s)
	length = len(s)
	return
}
