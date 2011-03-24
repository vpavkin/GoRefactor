package printerUtil

import (
	"testing"
	"go/token"
	"refactoring/program"
	"refactoring/utils"
	"refactoring/errors"
)

func Test_Delete(t *testing.T) {
	filename := "/home/rulerr/goRefactor/testSrc/testPack/testPack.go"
	srcDir, sources, specialPackages, _ := utils.GetProjectInfo(filename)
	p := program.ParseProgram(srcDir, sources, specialPackages)
	pack, file := p.FindPackageAndFileByFilename(filename)
	if pack == nil || file == nil {
		t.Fatalf(errors.ArgumentError("filename", "Program packages don't contain file '"+filename+"'").String())
	}
	if ok, err := DeleteNode(pack.FileSet, filename, file, token.Position{filename, 0, 19, 1}, token.Position{filename, 0, 20, 7}); !ok {
		t.Fatalf(err.String())
	}
	p.SaveFile(filename)
}

func test_getNLines(t *testing.T) {
	for i := 1; i < 2000; i++ {
		if l := len(getNLines(i)); l != i {
			t.Fatalf("expected %d, got %d", i, l)
		}
	}
}

func test_reparseFile(t *testing.T) {
	filename := "/home/rulerr/goRefactor/testSrc/testPack/printer.go"
	srcDir, sources, specialPackages, _ := utils.GetProjectInfo(filename)
	p := program.ParseProgram(srcDir, sources, specialPackages)
	pack, file := p.FindPackageAndFileByFilename(filename)
	if pack == nil || file == nil {
		t.Fatalf(errors.ArgumentError("filename", "Program packages don't contain file '"+filename+"'").String())
	}
	N := 1000
	fset, _ := reparseFile(file, filename, N, p.IdentMap)
	oldTF := getFileFromFileSet(pack.FileSet, filename)
	newTF := getFileFromFileSet(fset, filename)
	if newTF.Size()-oldTF.Size() != N {
		t.Fatalf("expected %d, got %d", N, newTF.Size()-oldTF.Size())
	}
}

func test_ReplaceNode(t *testing.T) {
	// filename := "/home/rulerr/goRefactor/testSrc/testPack/testPack.go"
	filename := "/home/rulerr/goRefactor/testSrc/testPack/printer.go"
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
	// 	if ok, err := ReplaceNode(pack.FileSet, filename, file, token.Position{filename, 0, 9, 1}, token.Position{filename, 0, 11, 2},
	// 		pack.FileSet, filename, file, token.Position{filename, 0, 17, 1}, token.Position{filename, 0, 25, 2}); !ok {
	// 		t.Fatalf(err.String())
	// 	}
	if ok, fset, newF, err := ReplaceNode(pack.FileSet, filename, file, token.Position{filename, 0, 15, 1}, token.Position{filename, 0, 15, 18},
		pack.FileSet, filename, file, token.Position{filename, 0, 5, 1}, token.Position{filename, 0, 13, 2}, p.IdentMap); !ok {
		t.Fatalf(err.String())
	} else {
		p.SaveFileExplicit(filename, fset, newF)
	}
}

func test_AddDecl(t *testing.T) {
	filename := "/home/rulerr/goRefactor/testSrc/testPack/printer.go"
	srcDir, sources, specialPackages, _ := utils.GetProjectInfo(filename)
	p := program.ParseProgram(srcDir, sources, specialPackages)
	pack, file := p.FindPackageAndFileByFilename(filename)
	if pack == nil || file == nil {
		t.Fatalf(errors.ArgumentError("filename", "Program packages don't contain file '"+filename+"'").String())
	}
	if ok, fset, newF, err := AddDecl(pack.FileSet, filename, file, pack.FileSet, filename, file, token.Position{filename, 0, 5, 1}, token.Position{filename, 0, 13, 2}, p.IdentMap); !ok {
		t.Fatalf(err.String())
	} else {
		p.SaveFileExplicit(filename, fset, newF)
	}
}
