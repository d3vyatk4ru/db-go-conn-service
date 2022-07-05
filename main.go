package main

import (
	"database/sql"
	"fmt"
	"log"
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

type Item struct {
	Id          int64
	Title       string
	Description string
	Updated     sql.NullString
}

type Users struct {
	User_id  int
	Login    string
	Password string
	Email    string
	Info     string
	Updated  sql.NullString
}

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

	rows, err := db.Query("SELECT id, title, updated FROM items")

	if err != nil {
		panic(err)
	}

	for rows.Next() {
		post := &Item{}
		err = rows.Scan(&post.Id, &post.Title, &post.Updated)

		if err != nil {
			panic(err)
		}

		fmt.Println(post.Id, " | ", post.Title, " | ", post.Updated.String, " |")
	}

	rows.Close()

	handler, err := NewDBExplorer(db) //nolint:typecheck

	if err != nil {
		panic(err)
	}

	fmt.Println("starting server at :8082")
	if err := http.ListenAndServe(":8082", handler); err != nil {
		log.Printf("error listenAndServer: %v", err)
	}
}
