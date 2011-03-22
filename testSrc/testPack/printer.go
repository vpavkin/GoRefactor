package testPack

import "fmt"

func printerF1() {
	println()
	print()

	a := new(int)
	*a = 20
	f2(a)
	fmt.Printf("%d", *a)
}

var printerV1 int

var printerV2 int
