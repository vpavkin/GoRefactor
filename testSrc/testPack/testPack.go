package testPack

import (
	"fmt"
)
func f2 (a *int) {
	*a = 10
}
func f1(b int){
	a := new(int)
	*a = 20
	f2(a)
	fmt.Printf("%d",*a + b)
}