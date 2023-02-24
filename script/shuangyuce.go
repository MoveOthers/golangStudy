package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

type suangseqiu struct {
	Number int `json:"number"`
	Id     int `json:"id"`
	Red1   int `json:"red1"`
	Red2   int `json:"red2"`
	Red3   int `json:"red3"`
	Red4   int `json:"red4"`
	Red5   int `json:"red5"`
	Red6   int `json:"red6"`
	Blue   int `json:"blue"`
}

func main() {
	//链接数据库
	db, err := sql.Open("mysql", "root:root@tcp(192.3.11.116:3306)/qianxin")
	if err != nil {
		fmt.Println("---err2--", err.Error())
		panic(any("---dddd"))
	}
	//获取最大的期数
	str := "select id,red1,red2,red3,red4,red5,red6,blue,number from suangseqiu;"
	query, err := db.Query(str)
	if err != nil {
		fmt.Println("---err3--", err.Error())
		panic(any(err))
	}
	list := make(map[int]map[string]interface{}, 0)
	for query.Next() {
		var cc suangseqiu
		err = query.Scan(&cc.Id, &cc.Red1, &cc.Red2, &cc.Red3, &cc.Red4, &cc.Red5, &cc.Red6, &cc.Blue, &cc.Number)
		if err != nil {
			fmt.Println("---err4--", err.Error())
			panic(any(err))
		}
		info := make(map[string]interface{}, 0)
		info["id"] = cc.Id
		info["red1"] = cc.Red1
		info["red2"] = cc.Red2
		info["red3"] = cc.Red3
		info["red4"] = cc.Red4
		info["red5"] = cc.Red5
		info["red6"] = cc.Red6
		info["blue"] = cc.Blue
		info["number"] = cc.Number
		list[cc.Id] = info
	}
	//杀号失败的期 和对应的杀号失败的球
	//errList := make(map[int]map[string]interface{},0)
	//排除红色号码  杀号
	for k, v := range list {
		//用34减去上一期的六个号码 获取对称码号
		//对称码切片
		symmetry := make(map[string]int, 0)
		symmetry["red1"] = 34 - v["red1"].(int)
		symmetry["red2"] = 34 - v["red2"].(int)
		symmetry["red3"] = 34 - v["red3"].(int)
		symmetry["red4"] = 34 - v["red4"].(int)
		symmetry["red5"] = 34 - v["red5"].(int)
		symmetry["red6"] = 34 - v["red6"].(int)
		//上一期的第六颗红球 减去第一位红球
		//杀号球
		Kill := []int{}
		Kill = append(Kill, v["red6"].(int)-v["red1"].(int))
		//第三位对称码 加上 7
		Kill = append(Kill, symmetry["red3"]-3)
		//第一位红+第二 +第三 减去33
		Kill = append(Kill, v["red2"].(int)+v["red1"].(int)+v["red3"].(int)-33)
		//蓝色球+2
		Kill = append(Kill, v["blue"].(int)+2)

		//上一期红球的切片
		previous := []int{}
		previous = append(previous, list[k+1]["red1"].(int))
		previous = append(previous, list[k+1]["red2"].(int))
		previous = append(previous, list[k+1]["red3"].(int))
		previous = append(previous, list[k+1]["red4"].(int))
		previous = append(previous, list[k+1]["red5"].(int))
		previous = append(previous, list[k+1]["red6"].(int))
		fmt.Println("-------", v)
		fmt.Println("-------", previous)
		//for _,vv:=range Kill{
		//
		//}

		return
	}
}
