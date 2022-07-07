package main

import (
	"database/sql"
	"fmt"
	"net/http"

	_ "github.com/microsoft/go-mssqldb"
)

// var (
// // docker run -p 3306:3306 -v $(PWD):/docker-entrypoint-initdb.d -e MYSQL_ROOT_PASSWORD=1234 -e MYSQL_DATABASE=golang -d mysql
// // DSN = "root:pwd@tcp(localhost:1433)/golang?charset=utf8"
// // DSN = "coursera:5QPbAUufx7@tcp(localhost:3306)/coursera?charset=utf8"
// )

var (
	server   = "localhost"
	user     = "sa"
	port     = 1433
	password = "super_PWD_go"
	database = "master"
)

// DSN это соединение с базой
var DSN = fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s;",
	server, user, password, port, database)

func main() {

	db, err := sql.Open("sqlserver", DSN)

	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(10)

	err = db.Ping() // тут будет первое подключение к бд

	if err != nil {
		panic(err)
	}

	handler, err := NewDBExplorer(db) //nolint:typecheck

	if err != nil {
		panic(err)
	}

	fmt.Println("starting server at :8082")

	err = http.ListenAndServe(":8082", handler)

	if err != nil {
		panic(err)
	}
}
