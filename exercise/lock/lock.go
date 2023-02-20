package main

import (
	"fmt"
	"time"
)

func main() {
	a := 1
	for i := 0; i < 10000; i++ {
		go func() {
			a++
			fmt.Println(a)
		}()
	}
	time.Sleep(10 * time.Second)
}
