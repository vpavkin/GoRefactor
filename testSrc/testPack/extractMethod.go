package testPack

import "os"

var exm_global int

func exm1() {

	var t os.Error
	a := true

	if a {
		println()
		exm_global = 0
	}
	if t == os.EOF {
		println(t.String())
	}

}
