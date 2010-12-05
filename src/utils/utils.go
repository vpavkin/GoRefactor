package utils

import (
	"os"
	"path"
	"strings"
	"go/token"
)

func GoFilter(f *os.FileInfo) bool {
	return IsGoFile(f.Name)
}

func IsGoFile(fileName string) bool {
	return (path.Ext(fileName) == ".go") && !(strings.HasSuffix(fileName, "_test.go"))
}

func ComparePosWithinFile(pos1 token.Position,pos2 token.Position) int{
	switch{
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