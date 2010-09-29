package main

import (
	"fmt"
	"go/printer"
	"flag"
	"os"
)
import . "symbolTable"


var filename, newName string
var line, column int

func init() {
	flag.StringVar(&filename, "f", "", "usage: -f <filename>")
	flag.IntVar(&line, "l", -1, "usage: -l <line number>")
	flag.IntVar(&column, "c", -1, "usage: -c <column number>")
	flag.StringVar(&newName, "n", "", "usage: -n <new symbol name>")
	flag.Parse()
}

func main() {

	if filename == "" || line == -1 || column == -1 || newName == "" {
		fmt.Printf("%v %v %v \n", filename, line, column)
		fmt.Printf("Usage: goref -f <filename> -l <line number> -c <column number> -n <new symbol name>\n")
		return
	}
	newPack, count := RenameIdent(".", filename, line, column, newName)
	if newPack != nil {
		fmt.Printf("res : %v\n", count)
		for fname, f := range newPack.Files {
			if nf, err := os.Open("refgo_out_"+fname, os.O_CREAT|os.O_EXCL|os.O_RDWR, 0666); err != nil {
				fmt.Printf("file %v already exists,cannot finish output\n", "refgo_out_"+fname)
				return
			} else {

				printer.Fprint(nf, f)
				nf.Close()
			}
		}
	} else {
		fmt.Printf("error : %v\n", count)
	}

}
