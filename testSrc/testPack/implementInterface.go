package testPack

import "os"
import "io"

var imi_global io.Reader

type imi_type interface {
	Error() os.Error
	Sum(a, b int) int
}

type imi_TYPE_1 int

type imi_TYPE_2 int
