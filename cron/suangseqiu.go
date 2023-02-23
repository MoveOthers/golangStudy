package main

import (
	"database/sql"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
	"strconv"
)

type number struct {
	Number int `json:"number"`
}

func main() {
	//链接数据库
	db, err := sql.Open("mysql", "root:root@tcp(192.3.11.116:3306)/qianxin")
	if err != nil {
		fmt.Println("---err2--", err.Error())
		panic(any("---dddd"))
	}
	//获取最大的期数
	str := "select number from suangseqiu order  by id desc  limit 1;"
	query, err := db.Query(str)
	if err != nil {
		fmt.Println("---err3--", err.Error())
		panic(any(err))
	}
	num := 0
	for query.Next() {
		var cc number
		err = query.Scan(&cc.Number)
		if err != nil {
			fmt.Println("---err4--", err.Error())
			panic(any(err))
		}
		num = cc.Number

	}

	resp, err := http.Get("http://datachart.500.com/ssq/history/newinc/history.php?start=" + strconv.Itoa(num))
	//resp, err := http.Get("http://datachart.500.com/ssq/history/newinc/history.php?start=23018" )
	if err != nil {
		log.Fatalln("get request failed!")
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatalln("failed to parse response body!")
	}

	doc.Find(".t_tr1").Each(func(i int, selection *goquery.Selection) {

		// 将每一条数据，从td标签中取出
		trSelection := selection.Find("td")
		//fmt.Println(selection.Find("td").Text())
		// 获取期数
		lotteryNum, _ := trSelection.Eq(0).Html()

		// 获取开奖时间
		lotteryTime, _ := trSelection.Eq(15).Html()
		// 获取红色球开奖号码
		lotteryRed1, _ := trSelection.Eq(1).Html()
		lotteryRed2, _ := trSelection.Eq(2).Html()
		lotteryRed3, _ := trSelection.Eq(3).Html()
		lotteryRed4, _ := trSelection.Eq(4).Html()
		lotteryRed5, _ := trSelection.Eq(5).Html()
		lotteryRed6, _ := trSelection.Eq(6).Html()
		// 获取蓝色球开奖号码
		lotteryBlue, _ := trSelection.Eq(7).Html()
		// 获取奖金池金额
		lotteryMoney, _ := trSelection.Eq(9).Html()

		if lotteryNum != strconv.Itoa(num) {
			sqlStr := "INSERT INTO `suangseqiu` ( `red1`, `red2`, `red3`, `red4`, `red5`, `red6`, `blue`, `bonus_pool`, `drawing_time`, `number`) VALUES (" + lotteryRed1 + ", " + lotteryRed2 + ", " + lotteryRed3 + ", " + lotteryRed4 + ", " + lotteryRed5 + ", " + lotteryRed6 + ", " + lotteryBlue + ", '" + lotteryMoney + "', '" + lotteryTime + "', " + lotteryNum + ");"
			query, err := db.Query(sqlStr)
			if err != nil {
				fmt.Printf("insert data error: %v\n", err)
			}
			var result int
			query.Scan(&result)
			query.Close()
		}
	})

	fmt.Println("----3--")

}
