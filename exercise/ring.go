package main

import (
	ring2 "container/ring"
	"fmt"
)

func main() {
	next := 10
	ring := ring2.New(next)

	for i := 0; i < 100; i++ {
		ring.Value = i
		ring = ring.Next()
	}

	sum := 0
	ring.Do(func(i interface{}) {
		num := i.(int)
		sum += num
	})
	fmt.Println("----------------", sum/next)
}
