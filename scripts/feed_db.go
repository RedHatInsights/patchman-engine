package main

import (
	"app/base/database"
	"io/ioutil"
)

func main() {
	database.InitDB()
	database.DBWait("full")
	query, err := ioutil.ReadFile("./dev/test_data.sql")
	if err != nil {
		panic(err)
	}
	err = database.Db.Exec(string(query)).Error
	if err != nil {
		panic(err)
	}
}
