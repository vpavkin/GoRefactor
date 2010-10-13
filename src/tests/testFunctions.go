package tests

import ss "strconv"

type AAAAAA struct {
	Obj         int
	OtherField1 bool
}
type BBBBBB struct {
	Obj         string
	OtherField2 bool
}
type III int
type BBB bool
func (b BBBBBB) Method3(a int){
}
func ELLTEST(a int, b ...string){
   bb := BBBBBB{};
   BBBBBB.Method3(bb,1);
}

func FFFFFF() *AAAAAA { return &AAAAAA{} }
func XXXXXX() *BBBBBB { return &BBBBBB{} }
func GGGGGG() {
	if true {
		x := FFFFFF()
		x.Obj = 10
	} else {
		x := XXXXXX()
		x.Obj = "asdfasd"
	}

	return
}
func Method1() bool{
   t:= BBB(true);
   tt := BBB(false);
   var r BBB = t && tt;
   return bool(r);
   
}
func (pppp T) Meth(){
   pppp.X = new(T)
}
func Method2() int {

   var ttt PP = PP(new(T));
   var aa AA ;
   ttt.X = P(aa[0]);
	
   println(ss.Itoa(3))
   t:= III(4);
   tt := III(5);
   var r III = t + tt;
   return int(r);
}

