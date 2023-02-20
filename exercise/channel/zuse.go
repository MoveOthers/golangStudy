package main

import (
	"fmt"
	"time"
)

func main() {
	ch := make(chan struct{}, 1)
	ch <- struct{}{}

	go func() {
		time.Sleep(time.Second * 5)

		//<-ch
		fmt.Println("第一次取出管道内的数据")
	}()
	fmt.Println("------1----")
	ch <- struct{}{}

	fmt.Println("-----2-----")
	go func() {
		time.Sleep(time.Second * 2)
		fmt.Println("写入阻塞11")
		ch <- struct{}{}
		fmt.Println("写入阻塞")
	}()

	time.Sleep(3 * time.Second)
	fmt.Println("-----结束-------")
}
