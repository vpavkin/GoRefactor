package testPack

import "os"

var exm_global int

func exm1() {

	var t os.Error
	a := true

	new1(t, a)

}

func exm2() {
	println("666", "555")
}

func exm3(a, b, c bool) int {
	if (!a && !b) || (!c && a) {
		return 3
	}
	return 666
}

func exm4_pointer() {
	a := true
	b := 3

	a = false
	println(a)
	println(&b)
}
func new1(t os.Error, a bool) {

	if a {
		println()
		exm_global = 0
	}
	if t == os.EOF {
		println(t.String())
	}
}
