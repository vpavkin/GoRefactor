package printerUtil

import (
	"testing"
	"go/parser"
	"go/token"
	"refactoring/program"
	"refactoring/utils"
)

var code1 string = `package testcode1

import "fmt"

type TestExtrInterfaceType int

func (a TestExtrInterfaceType) adsda() {

}

func f2(a *int) {
	*a = 10
}
func f1(b int) {
	a := new(int)
	*a = 20
	f2(a)
	fmt.Printf("%d", *a+b)

	exi1(1, true, 3, 3, 3, 3, 3, 4)
}

func exi1(aa int, bb bool, a, b, c, d, e TestExtrInterfaceType, cc int) {
	c.adsda()
}
`

func Test_as(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "aaa.go", code1, parser.ParseComments)
	if err != nil {
		t.Fatalf(err.String())
	}
	if ok, err := DeleteNode(fset, "aaa.go", file, token.Position{"aaa.go", 0, 7, 1}, token.Position{"aaa.go", 0, 9, 2}); !ok {
		t.Fatalf(err.String())
	}
}

func Test_DeleteNode(t *testing.T) {
	filename := "/home/rulerr/goRefactor/testSrc/testPack/testPack.go"
	srcDir, sources, specialPackages, _ := utils.GetProjectInfo(filename)
	program.ParseProgram(srcDir, sources,specialPackages)
}
