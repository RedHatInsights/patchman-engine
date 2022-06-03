package main

import (
	"app/base/database"
	"io/ioutil"
)

func main() {
	database.InitDB()
	database.DBWait("empty")
	query, err := ioutil.ReadFile("./dev/test_data.sql")
	if err != nil {
		panic(err)
	}
	database.Db.Exec(string(query))
}
