package main

import (
	"fmt"
	"net/http"
	"net/rpc"
)

func main() {
	rect := new(Test)
	err := rpc.Register(rect)
	if err != nil {
		panic(any("-------结束-----"))
	}
	rpc.HandleHTTP()
	if err := http.ListenAndServe(":8099", nil); err != nil {
		fmt.Println(err)
	}
}

type Test struct {
}

type Params struct {
	A, B int
}

func (t *Test) Sum(p Params) (r int, err error) {
	r = p.A + p.B
	return r, nil
}

func (t *Test) Poor(p Params) (r int, err error) {
	r = p.A - p.B
	return r, nil
}
