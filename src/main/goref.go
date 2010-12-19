package main

import (
	"fmt"
	"flag"
	"os"
	"path"
	"utils"
	"refactoring"
	"errors"
	"program"
)

var filename, entityName string
var line, column int
var endLine, endColumn int
var action string


const (
	INIT                string = "init"
	RENAME              string = "ren"
	EXTRACT_METHOD      = "exm"
	INLINE_METHOD       = "inm"
	EXTRACT_INTERFACE   = "exi"
	IMPLEMENT_INTERFACE = "imi"
	SORT                = "sort"
)

func init() {

	flag.StringVar(&action, "a", "", "usage: -a <refactoring action>")
	flag.StringVar(&filename, "f", "", "usage: -f <filename>")
	flag.IntVar(&line, "l", -1, "usage: -l <line number>")
	flag.IntVar(&column, "c", -1, "usage: -c <column number>")
	flag.IntVar(&endLine, "el", -1, "usage: -el <line number>")
	flag.IntVar(&endColumn, "ec", -1, "usage: -ec <column number>")
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
		fmt.Println(srcDir)
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
			fmt.Printf("%v\n", err)
		}
		defer fd.Close()

		fmt.Printf("inited\n", action)
	case RENAME:
		var err *errors.GoRefactorError
		switch {
		case filename == "" || !utils.IsGoFile(filename):
			err = errors.ArgumentError("filename", "It's not a valid go file name")

		case line < 1:
			err = errors.ArgumentError("line", "Must be > 1")
		case column < 1:
			err = errors.ArgumentError("column", "Must be > 1")
		case !refactoring.IsGoIdent(entityName):
			err = errors.ArgumentError("newName", "It's not a valid go identifier")
		}
		if err != nil {
			fmt.Println(err.Message)
			return
		}
		srcDir, _ := getInitedDir(filename)
		fmt.Printf("dir %s\n", srcDir)
		p := program.ParseProgram(srcDir, nil)
		if ok, count, err := refactoring.Rename(p, filename, line, column, entityName); !ok {
			fmt.Println("error:", err.Message)
		} else {
			fmt.Println(count, "occurences renamed")
		}
	case EXTRACT_METHOD:
		fmt.Println("this feature is not implemented yet")
	case INLINE_METHOD:
		fmt.Println("this feature is not implemented yet")
	case EXTRACT_INTERFACE:
		fmt.Println("this feature is not implemented yet")
	case IMPLEMENT_INTERFACE:
		fmt.Println("this feature is not implemented yet")
	case SORT:
		fmt.Println("this feature is not implemented yet")
	}
	fmt.Printf("%s %s %d %d %s\n", action, filename, line, column, entityName)
}
