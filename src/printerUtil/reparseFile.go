package printerUtil

import (
	"go/ast"
	"go/parser"
	"go/token"
	"refactoring/st"
	"os"
)

type restoreIMSourceVisitor struct {
	comm chan *ast.Ident
}

type restoreIMDestVisitor struct {
	identMap st.IdentifierMap
	comm     chan *ast.Ident
}

func (vis *restoreIMSourceVisitor) Visit(node ast.Node) ast.Visitor {
	switch t := node.(type) {
	case *ast.Ident:
		vis.comm <- t
	}
	return vis
}

func (vis *restoreIMDestVisitor) Visit(node ast.Node) ast.Visitor {
	switch t := node.(type) {
	case *ast.Ident:
		id := <-vis.comm
		if s, ok := vis.identMap.GetSymbolSafe(id); ok {
			vis.identMap.AddIdent(t, s)
		}
	}
	return vis
}

func restoreIdentMapping(oldFile, newFile *ast.File, identMap st.IdentifierMap) {
	comm := make(chan *ast.Ident)
	source := &restoreIMSourceVisitor{comm}
	dest := &restoreIMDestVisitor{identMap, comm}
	go ast.Walk(source, oldFile)
	ast.Walk(dest, newFile)
}

func getNLines(n int) string {
	res := ""
	t := "\n"
	for n > 0 {
		if n%2 == 1 {
			res += t
		}
		t += t
		n = int(n / 2)
	}
	return res
}

func appendFile(filename string, add int) {
	fd, err := os.Open(filename, os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		panic("couldn't open file " + filename + ": " + err.String())
	}
	defer fd.Close()
	if _, err := fd.Seek(0, 2); err != nil { //end
		panic("couldn't write to file " + filename + ": " + err.String())
	}
	s := getNLines(add)

	n, err := fd.WriteString(s)
	if n != add || err != nil {
		panic("couldn't write to file " + filename + ".")
	}
	if err := fd.Sync(); err != nil {
		panic("couldn't write to file " + filename + ": " + err.String())
	}
}

func reparseFile(oldFile *ast.File, filename string, add int, identMap st.IdentifierMap) (fset *token.FileSet, file *ast.File) {
	appendFile(filename, add)

	var err os.Error
	fset = token.NewFileSet()
	file, err = parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		panic("couldn't reparse file " + filename + ": " + err.String())
	}
	restoreIdentMapping(oldFile, file, identMap)
	return
}
