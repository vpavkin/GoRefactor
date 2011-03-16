package printerUtil

import (
	"testing"
	"go/token"
	"refactoring/program"
	"refactoring/utils"
	"refactoring/errors"
)

func Test_DeleteNode(t *testing.T) {
	filename := "/home/rulerr/goRefactor/testSrc/testPack/testPack.go"
	srcDir, sources, specialPackages, _ := utils.GetProjectInfo(filename)
	p := program.ParseProgram(srcDir, sources, specialPackages)
	pack, file := p.FindPackageAndFileByFilename(filename)
	if pack == nil || file == nil {
		t.Fatalf(errors.ArgumentError("filename", "Program packages don't contain file '"+filename+"'").String())
	}
	if ok, err := DeleteNode(pack.FileSet, filename, file, token.Position{filename, 0, 15, 2}, token.Position{filename, 0, 15, 9}); !ok {
		t.Fatalf(err.String())
	}
	p.SaveFile(filename)
}
