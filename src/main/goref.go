package main

import (
	"fmt"
	"flag"
	"os"
	"path"
	//"utils"
	"refactoring"
	//"errors"
	"program"
)

var filename, varFile, entityName string
var line, column int
var endLine, endColumn int
var varLine, varColumn int
var action string


const (
	INIT string = "init"
)

func init() {

	flag.StringVar(&action, "a", "", "usage: -a <refactoring action>")
	flag.StringVar(&filename, "f", "", "usage: -f <filename>")
	flag.IntVar(&line, "l", -1, "usage: -l <line number>")
	flag.IntVar(&column, "c", -1, "usage: -c <column number>")
	flag.IntVar(&endLine, "el", -1, "usage: -el <line number>")
	flag.IntVar(&endColumn, "ec", -1, "usage: -ec <column number>")
	flag.StringVar(&varFile, "vf", "", "usage: -vf <filename>")
	flag.IntVar(&varLine, "vl", -1, "usage: -vl <line number>")
	flag.IntVar(&varColumn, "vc", -1, "usage: -vc <column number>")
	flag.StringVar(&entityName, "e", "", "usage: -e <entity name>")
	flag.Parse()

}

func getInitedDir(filename string) (string, bool) {
	srcDir, _ := path.Split(filename)
	srcDir = srcDir[:len(srcDir)-1]
	srcDir, _ = path.Split(srcDir)
	for {
		srcDir = srcDir[:len(srcDir)-1]
		if srcDir == "" {
			return "", false
		}
		//fmt.Println(srcDir)
		fd, _ := os.Open(srcDir, os.O_RDONLY, 0)

		list, _ := fd.Readdir(-1)

		for i := 0; i < len(list); i++ {
			d := &list[i]
			if d.Name == "goref.cfg" {
				return srcDir, true
			}
		}
		srcDir, _ = path.Split(srcDir)
		fd.Close()
	}
	return "", false
}

//TODO: add extern sources support
func main() {
	switch action {
	case INIT:
		fd, err := os.Open("goref.cfg", os.O_CREATE, 0666)
		if err != nil {
			fmt.Printf("error: %v\n", err)
		} else {
			defer fd.Close()
			fmt.Printf("inited\n", action)
		}
	case refactoring.RENAME:
		if ok, err := refactoring.CheckRenameParameters(filename, line, column, entityName); !ok {
			fmt.Println("error:", err.Message)
			return
		}
		fmt.Println("renaming symbol to ", entityName+"...")
		srcDir, _ := getInitedDir(filename)
		p := program.ParseProgram(srcDir, nil)
		if ok, count, err := refactoring.Rename(p, filename, line, column, entityName); !ok {
			fmt.Println("error:", err.Message)
		} else {
			fmt.Println(count, "occurences renamed")
			p.Save()
		}
	case refactoring.EXTRACT_METHOD:
		if ok, err := refactoring.CheckExtractMethodParameters(filename, line, column, endLine, endColumn, entityName, varLine, varColumn); !ok {
			fmt.Println("error:", err.Message)
			return
		}
		fmt.Println("extracting code to method ", entityName+"...")
		srcDir, _ := getInitedDir(filename)
		p := program.ParseProgram(srcDir, nil)
		if ok, err := refactoring.ExtractMethod(p, filename, line, column, endLine, endColumn, entityName, varLine, varColumn); !ok {
			fmt.Println("error:", err.Message)
			return
		}
		p.SaveFile(filename)
	case refactoring.INLINE_METHOD:
		if ok, err := refactoring.CheckInlineMethodParameters(filename, line, column, endLine, endColumn); !ok {
			fmt.Println("error:", err.Message)
			return
		}
		fmt.Println("inlining call...")
		srcDir, _ := getInitedDir(filename)
		p := program.ParseProgram(srcDir, nil)
		if ok, err := refactoring.InlineMethod(p, filename, line, column, endLine, endColumn); !ok {
			fmt.Println("error:", err.Message)
			return
		}
		p.SaveFile(filename)
	case refactoring.EXTRACT_INTERFACE:
		fmt.Println("this feature is not implemented yet")
	case refactoring.IMPLEMENT_INTERFACE:
		if ok, err := refactoring.CheckImplementInterfaceParameters(filename, line, column, varFile, varLine, varColumn); !ok {
			fmt.Println("error:", err.Message)
			return
		}
		fmt.Println("implementing interface...")
		srcDir, _ := getInitedDir(filename)
		p := program.ParseProgram(srcDir, nil)
		if ok, err := refactoring.ImplementInterface(p, filename, line, column, varFile, varLine, varColumn); !ok {
			fmt.Println("error:", err.Message)
			return
		}
		p.SaveFile(varFile)
	case refactoring.SORT:
		fmt.Println("this feature is not implemented yet")
	}
	//fmt.Printf("%s %s %d %d %s\n", action, filename, line, column, entityName)
}
