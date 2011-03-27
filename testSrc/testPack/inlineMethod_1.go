package testPack

import "strings"
import "fmt"
import "os"
import "go/token"

func inm1_1(a string) {
	i := strings.Index(a, "a")
	fmt.Printf("%d", i)
	println(os.EOF)
	println(token.ASSIGN)
	if token.ASSIGN != token.AND {
		println("!!!!")
	}
}
