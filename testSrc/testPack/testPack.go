package testPack

import (
	"fmt"
)

type TestExtrInterfaceType int

func (a TestExtrInterfaceType) adsda() {

}

func f2(a *int) {
	*a = 10
}
func f1(b int) {
	a := new(int)
	*a = 20
	f2(a)
	fmt.Printf("%d", *a+b)
	
	exi1(1,true,3,3,3,3,3,4)
}

func exi1(aa int, bb bool, a, b, c, d, e TestExtrInterfaceType,cc int) {
	c.adsda()
}


