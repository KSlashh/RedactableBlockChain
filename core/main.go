package main

import (
	"fmt"
)

func main() {
	var para [][]byte = [][]byte{[]byte("11234232"),[]byte("2342342133"),[]byte("321342343124")}
	t,_ := NewBasicTx([]byte("im ztj"),"its true",[]byte("423421341325"),para)
	err := t.Write("./test.json")
	if err != nil {fmt.Println(err)}
	tt := &BasicTx{}
	tt.Load("./test.json")
	fmt.Println(tt.Proof())
	// var x interface{}
	// x = new(BasicTx)
	// _,ok := x.(Tx)
	// fmt.Println(ok)
}