package main

import (
	"fmt"
	"os"
	"path"
	"strconv"
	//"utils"
	"refactoring"
	//"errors"
	"program"
)

const (
	INIT string = "init"
)
const renameUsage string = "usage: goref ren <filename> <line> <column> <new name>"
const extractMethodUsage string = "usage: goref exm <filename> <line> <column> <end line> <end column> <new name> [<recvLine> <recvColumn>]"
const inlineMethodUsage string = "usage: goref inm <filename> <line> <column> <end line> <end column>"
const implementInterfaceUsage string = "usage: goref imi [-p] <filename> <line> <column> <type line> <type column>\n\t-p: implement interface for pointerType"

func getRenameArgs() (filename string, line int, column int, entityName string, ok bool) {
	var err os.Error
	if len(os.Args) < 6 {
		return
	}
	filename = os.Args[2]
	line, err = strconv.Atoi(os.Args[3])
	if err != nil {
		return
	}
	column, err = strconv.Atoi(os.Args[4])
	if err != nil {
		return
	}
	entityName = os.Args[5]
	ok = true
	return
}

func getExtractMethodArgs() (filename string, line int, column int, endLine int, endColumn int, entityName string, recvLine int, recvColumn int, ok bool) {
	var err os.Error
	if len(os.Args) < 8 {
		return
	}
	filename = os.Args[2]
	line, err = strconv.Atoi(os.Args[3])
	if err != nil {
		return
	}
	column, err = strconv.Atoi(os.Args[4])
	if err != nil {
		return
	}
	endLine, err = strconv.Atoi(os.Args[5])
	if err != nil {
		return
	}
	endColumn, err = strconv.Atoi(os.Args[6])
	if err != nil {
		return
	}
	entityName = os.Args[7]
	recvLine = -1
	recvColumn = -1
	if len(os.Args) >= 10 {
		recvLine, err = strconv.Atoi(os.Args[8])
		if err != nil {
			return
		}
		recvColumn, err = strconv.Atoi(os.Args[9])
		if err != nil {
			return
		}
	}
	ok = true
	return
}

func getInlineMethodArgs() (filename string, line int, column int, endLine int, endColumn int, ok bool) {
	var err os.Error
	if len(os.Args) < 7 {
		return
	}
	filename = os.Args[2]
	line, err = strconv.Atoi(os.Args[3])
	if err != nil {
		return
	}
	column, err = strconv.Atoi(os.Args[4])
	if err != nil {
		return
	}
	endLine, err = strconv.Atoi(os.Args[5])
	if err != nil {
		return
	}
	endColumn, err = strconv.Atoi(os.Args[6])
	if err != nil {
		return
	}
	ok = true
	return
}

func getImplementInterfaceArgs() (filename string, line int, column int, typeFile string, typeLine int, typeColumn int, asPointer bool, ok bool) {
	var err os.Error
	p := 0
	if len(os.Args) < 8 {
		return
	}
	if os.Args[2] == "-p" {
		if len(os.Args) < 9 {
			return
		}
		asPointer = true
		p++
	}
	filename = os.Args[2+p]
	line, err = strconv.Atoi(os.Args[3+p])
	if err != nil {
		return
	}
	column, err = strconv.Atoi(os.Args[4+p])
	if err != nil {
		return
	}
	typeFile = os.Args[5+p]
	typeLine, err = strconv.Atoi(os.Args[6+p])
	if err != nil {
		return
	}
	typeColumn, err = strconv.Atoi(os.Args[7+p])
	if err != nil {
		return
	}
	ok = true
	return
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
	action := os.Args[1]
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
		filename, line, column, entityName, ok := getRenameArgs()
		if !ok {
			fmt.Println(renameUsage)
			return
		}
		if ok, err := refactoring.CheckRenameParameters(filename, line, column, entityName); !ok {
			fmt.Println("error:", err.Message)
			fmt.Println(renameUsage)
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
		filename, line, column, endLine, endColumn, entityName, recvLine, recvColumn, ok := getExtractMethodArgs()
		if !ok {
			fmt.Println(extractMethodUsage)
			return
		}
		if ok, err := refactoring.CheckExtractMethodParameters(filename, line, column, endLine, endColumn, entityName, recvLine, recvColumn); !ok {
			fmt.Println("error:", err.Message)
			return
		}
		fmt.Println("extracting code to method ", entityName+"...")
		srcDir, _ := getInitedDir(filename)
		p := program.ParseProgram(srcDir, nil)
		if ok, err := refactoring.ExtractMethod(p, filename, line, column, endLine, endColumn, entityName, recvLine, recvColumn); !ok {
			fmt.Println("error:", err.Message)
			return
		}
		p.SaveFile(filename)
	case refactoring.INLINE_METHOD:
		filename, line, column, endLine, endColumn, ok := getInlineMethodArgs()
		if !ok {
			fmt.Println(inlineMethodUsage)
			return
		}
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
		filename, line, column, typeFile, typeLine, typeColumn, asPointer, ok := getImplementInterfaceArgs()
		if !ok {
			fmt.Println(implementInterfaceUsage)
			return
		}
		if ok, err := refactoring.CheckImplementInterfaceParameters(filename, line, column, typeFile, typeLine, typeColumn); !ok {
			fmt.Println("error:", err.Message)
			return
		}
		fmt.Println("implementing interface...")
		srcDir, _ := getInitedDir(filename)
		p := program.ParseProgram(srcDir, nil)
		if ok, err := refactoring.ImplementInterface(p, filename, line, column, typeFile, typeLine, typeColumn, asPointer); !ok {
			fmt.Println("error:", err.Message)
			return
		}
		p.SaveFile(typeFile)
	case refactoring.SORT:
		fmt.Println("this feature is not implemented yet")
	}
	//fmt.Printf("%s %s %d %d %s\n", action, filename, line, column, entityName)
}
