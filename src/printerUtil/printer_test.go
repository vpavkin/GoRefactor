package printerUtil

import (
	"testing"
	"go/token"
	"refactoring/program"
	"refactoring/utils"
	"refactoring/errors"
)

func test_DeleteNode(t *testing.T) {
	filename := "/home/rulerr/goRefactor/testSrc/testPack/testPack.go"
	srcDir, sources, specialPackages, _ := utils.GetProjectInfo(filename)
	p := program.ParseProgram(srcDir, sources, specialPackages)
	pack, file := p.FindPackageAndFileByFilename(filename)
	if pack == nil || file == nil {
		t.Fatalf(errors.ArgumentError("filename", "Program packages don't contain file '"+filename+"'").String())
	}
	if ok, err := DeleteNode(pack.FileSet, filename, file, token.Position{filename, 0, 9, 1}, token.Position{filename, 0, 11, 2}); !ok {
		t.Fatalf(err.String())
	}
	p.SaveFile(filename)
}

func Test_ReplaceNode(t *testing.T) {
	filename := "/home/rulerr/goRefactor/testSrc/testPack/testPack.go"
// 	filename := "/home/rulerr/goRefactor/testSrc/testPack/printer.go"
	srcDir, sources, specialPackages, _ := utils.GetProjectInfo(filename)
	p := program.ParseProgram(srcDir, sources, specialPackages)
	pack, file := p.FindPackageAndFileByFilename(filename)
	if pack == nil || file == nil {
		t.Fatalf(errors.ArgumentError("filename", "Program packages don't contain file '"+filename+"'").String())
	}
	// 	if ok, err := ReplaceNode(pack.FileSet, filename, file, token.Position{filename, 0, 15, 2}, token.Position{filename, 0, 15, 9},
	// 		pack.FileSet, filename, file,token.Position{filename, 0, 20, 2}, token.Position{filename, 0, 20, 7}); !ok {
	// 		t.Fatalf(err.String())
	// 	}
		if ok, err := ReplaceNode(pack.FileSet, filename, file, token.Position{filename, 0, 9, 1}, token.Position{filename, 0, 11, 2},
			pack.FileSet, filename, file, token.Position{filename, 0, 17, 1}, token.Position{filename, 0, 25, 2}); !ok {
			t.Fatalf(err.String())
		}
// 	if ok, err := ReplaceNode(pack.FileSet, filename, file, token.Position{filename, 0, 8, 1}, token.Position{filename, 0, 8, 18},
// 		pack.FileSet, filename, file, token.Position{filename, 0, 3, 1}, token.Position{filename, 0, 6, 2}); !ok {
// 		t.Fatalf(err.String())
// 	}
	p.SaveFile(filename)
}
