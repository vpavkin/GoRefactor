package test

type A struct {
	Obj         int
	OtherField1 bool
}
type B struct {
	Obj         string
	OtherField2 bool
}
//...
func F() *A { return &A{} }
func X() *B { return &B{} }
func G() {
	if true {
		x := F()
		x.Obj = 10
	} else {
		x := X()
		x.Obj = "asdfasd"
	}

	return
}
