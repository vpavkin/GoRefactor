package tests

import ss "fmt"

type (
    B bool
    I int32
    A [10]P
    T struct {
        X P
    }
    AA A
    P *T
    PP P
    R *R
    F func(A) I
    Y interface {
        f(A) I
    }
    S []P
    M map[I]F
    C chan<- I
)

type (

    t *t

    U V
    V W
    W *U

    P1 *S1
    P2 P1

    S1 struct {
        a, b, c int
        u, v float
    }


    L1 []L1
    L2 []int

    A1 [10]int

    A4 [10]A1

    F1 func()
    F2 func(x, y, z float)
    F3 func(x, y float)
    F4 func() (x, y float)
    F5 func(x int) (x float)

    I1 interface{}
    I2 interface {
        m1()
    }
    I3 interface {
        m1()
    }
    I4 interface {
        m1(x, y float)
        m2() (x, y float)
        m3(x int) (x float)
    }
    I5 interface {
        m1(I5)
    }

    C1 chan int
    C2 <-chan int
    C3 chan<- C3

    M1 map[Last]string
    M2 map[string]M2

    Last int
)

func ABCD(){
	ss.Printf("\n");
}
