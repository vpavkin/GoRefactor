package main

import (
	"fmt"
	"os"
	"strconv"
	//"utils"
	"refactoring/refactoring"
	//"errors"
)

const (
	INIT string = "init"
	HELP string = "help"
)
const usage string = `usage: goref <action> {arguments}.
type "goref help" to look at allowed actions.`
const renameUsage string = "usage: goref ren <filename> <line> <column> <new name>"
const extractMethodUsage string = "usage: goref exm <filename> <line> <column> <end line> <end column> <new name> [<recvLine> <recvColumn>]"
const inlineMethodUsage string = "usage: goref inm <filename> <line> <column> <end line> <end column>"
const implementInterfaceUsage string = `usage: goref imi [-p] <filename> <line> <column> <type line> <type column>

-p: implement interface for pointerType`
const extractInterfaceUsage string = "usage: goref exi <filename> <line> <column> <interface name>"
const sortUsage string = `usage: goref sort [-t|-v] [-i] <filename> [<order>]

-t:      group methods by reciever type. Methods will be sorted alphabetically by name within group.
-v:      group methods by visibility. Methods will be sorted alphabetically by name within group.
-i:      sort imports alphabetically.
<order>: defines custom order of groups of declarations. Default order string is 'cvtmf' which means 'constants, variables, types, methods, functions'
Custom order string must contain at least one character from default order string. If it's length is less than the length of default order string, other entries will be added in the default order.
Leave out order parameter to use default order.`

func printUsage() {
	println("RENAME")
	fmt.Println(renameUsage)
	println()
	println("EXTRACT METHOD")
	fmt.Println(extractMethodUsage)
	println()
	println("INLINE METHOD")
	fmt.Println(inlineMethodUsage)
	println()
	println("IMPLEMENT INTERFACE")
	fmt.Println(implementInterfaceUsage)
	println()
	println("EXTRACT INTERFACE")
	fmt.Println(extractInterfaceUsage)
	println()
	println("SORT DECLARATIONS")
	fmt.Println(sortUsage)
	println()
}

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

func getExtractInterfaceArgs() (filename string, line int, column int, interfaceName string, ok bool) {
	return getRenameArgs()
}

func getSortArgs() (filename string, groupMethodsByType bool, groupMethodsByVisibility bool, sortImports bool, order string, ok bool) {
	p := 0
	for i := 0; i < 2; i++ {
		if len(os.Args) < 3+i {
			return
		}
		switch os.Args[2+i] {
		case "-t":
			p++
			groupMethodsByType = true
		case "-v":
			p++
			groupMethodsByVisibility = true
		case "-i":
			p++
			sortImports = true
		default:
			break
		}
	}
	if groupMethodsByType && groupMethodsByVisibility {
		return
	}
	if len(os.Args) < 3+p {
		return
	}
	filename = os.Args[2+p]
	if len(os.Args) > 3+p {
		order = os.Args[3+p]
	}
	ok = true
	return
}

func main() {
	if len(os.Args) <= 1 {
		fmt.Printf("%s\n", usage)
		return
	}
	action := os.Args[1]
	switch action {
	case HELP:
		printUsage()
	case INIT:
		//goref
		fd, err := os.OpenFile("goref.cfg", os.O_CREATE|os.O_RDWR|os.O_EXCL, 0666)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}
		defer fd.Close()
		if _, err = fd.WriteString(goref_config_stub); err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}
		//os
		oss, err := os.OpenFile("os.cfg", os.O_CREATE|os.O_RDWR|os.O_EXCL, 0666)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}
		defer oss.Close()
		if _, err = oss.WriteString(os_config); err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}
		//syscall
		syscall, err := os.OpenFile("syscall.cfg", os.O_CREATE|os.O_RDWR|os.O_EXCL, 0666)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}
		defer syscall.Close()
		if _, err = syscall.WriteString(syscall_config); err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}

		//runtime
		runtime, err := os.OpenFile("runtime.cfg", os.O_CREATE|os.O_RDWR|os.O_EXCL, 0666)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}
		defer runtime.Close()
		if _, err = runtime.WriteString(runtime_config); err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}

		fmt.Printf("Initialized goref project. Now fill goref.cfg with your packages.")

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

		if ok, err := refactoring.Rename(filename, line, column, entityName); !ok {
			fmt.Println("error:", err.Message)
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
		if ok, err := refactoring.ExtractMethod(filename, line, column, endLine, endColumn, entityName, recvLine, recvColumn); !ok {
			fmt.Println("error:", err.Message)
			return
		}
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

		if ok, err := refactoring.InlineMethod(filename, line, column, endLine, endColumn); !ok {
			fmt.Println("error:", err.Message)
			return
		}
	case refactoring.EXTRACT_INTERFACE:
		filename, line, column, interfaceName, ok := getExtractInterfaceArgs()
		fmt.Println(filename, line, column, interfaceName)
		if !ok {
			fmt.Println(extractInterfaceUsage)
			return
		}
		if ok, err := refactoring.CheckExtractInterfaceParameters(filename, line, column, interfaceName); !ok {
			fmt.Println("error:", err.Message)
			return
		}
		fmt.Println("extracting interface " + interfaceName + "...")
		if ok, err := refactoring.ExtractInterface(filename, line, column, interfaceName); !ok {
			fmt.Println("error:", err.Message)
			return
		}
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
		if ok, err := refactoring.ImplementInterface(filename, line, column, typeFile, typeLine, typeColumn, asPointer); !ok {
			fmt.Println("error:", err.Message)
			return
		}
	case refactoring.SORT:
		filename, groupMethodsByType, groupMethodsByVisibility, sortImports, order, ok := getSortArgs()
		println(filename, groupMethodsByType, groupMethodsByVisibility, sortImports, order)
		if !ok {
			fmt.Println(sortUsage)
			return
		}
		if ok, err := refactoring.CheckSortParameters(filename, order); !ok {
			fmt.Println("error:", err.Message)
			return
		}
		fmt.Println("sorting file " + filename + "...")
		if ok, err := refactoring.Sort(filename, groupMethodsByType, groupMethodsByVisibility, sortImports, order); !ok {
			fmt.Println("error:", err.Message)
			return
		}
	default:
		fmt.Printf("%s\n", usage)
	}
	//fmt.Printf("%s %s %d %d %s\n", action, filename, line, column, entityName)
}
