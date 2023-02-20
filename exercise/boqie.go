package main

import "fmt"

func main() {
	fmt.Println(fibonaci(6))
}

func fibonaci(n int) int {
	if n == 1 || n == 2 {
		return n - 1
	}
	a := make([]int, n+1)
	a[0] = 0
	a[1] = 1
	for i := 2; i < n; i++ {
		a[i] = a[i-1] + a[i-2]
	}
	return a[n-1]
}
