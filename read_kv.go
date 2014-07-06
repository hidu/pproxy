package main

import (
	"./serve"
	"fmt"
)

func main() {
	mydb := serve.NewTieDb("./data/")
	mydb.RequestTable.ForAll(func(id uint64, doc map[string]interface{}) bool {
		fmt.Println(id, doc["user"], doc["host"], doc["cookie"])
		return true
	})
}
