package testPack

import "os"
import "io"

type exi_type int

func (exi_type) exi_f1(a int) {
}

func (exi_type) exi_f2() (int, os.Error) {
	return 0, nil
}

func (exi_type) exi_f3(bool) io.Reader {
	return nil
}

func exi_f(a int, b exi_type, c exi_type) {
	b.exi_f1(0)
	if true {
		b.exi_f2()
	}
	c.exi_f3(false)
}
