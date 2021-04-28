package main

import (
	"fmt"
	"github.com/RedactableBlockChain/data"
)

func main() {
	var para [][]byte = [][]byte{[]byte("11234232"),[]byte("2342342133"),[]byte("321342343124")}
	var t,tt data.Tx;
	t,_ = data.NewBasicTx([]byte("im ztj"),[]byte("its true"),[]byte("423421341325"),para)
	err := data.Load(&t,"./test.json")
	if err != nil {fmt.Println(err)}
	tt = &data.BasicTx{}
	data.Load(&tt, "./test.json")
	fmt.Println(tt.Proof())
	// var x interface{}
	// x = new(BasicTx)
	// _,ok := x.(Tx)
	// fmt.Println(ok)
}