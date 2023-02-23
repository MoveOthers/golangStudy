package main

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"time"
)

func main() {
	Cron := cron.New()
	i := 1
	addFunc, err := Cron.AddFunc("*/1 * * * *", func() {
		fmt.Println("------每分钟执行一次---", time.Now(), i)
		i++
	})
	if err != nil {
		fmt.Println("-----err-----", err)
		return
	}
	fmt.Println(addFunc, time.Now(), err)
	Cron.Start()

	defer Cron.Stop()
	time.Sleep(time.Minute * 5)
}
