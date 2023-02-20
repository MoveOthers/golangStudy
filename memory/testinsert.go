package main

import "fmt"

func main() {

	a := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 0, 0, 1, 2, 12, 12, 2, 112, 2, 3, 1, 2222, 312}
	fmt.Printf("%p\n", a)
	b := a[0:5]
	fmt.Printf("%p\n", b)
}
