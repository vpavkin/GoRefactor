package utils

import (
	"os"
	"path"
	"strings"
)

func GoFilter(f *os.FileInfo) bool {
	return IsGoFile(f.Name)
}

func IsGoFile(fileName string) bool {
	return (path.Ext(fileName) == ".go") && !(strings.HasSuffix(fileName, "_test.go"))
}
