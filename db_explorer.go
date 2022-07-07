package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

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

type Handler struct {
	DB *sql.DB
}

type Table []string

func (h *Handler) TableList(w http.ResponseWriter, r *http.Request) {

	ListTable := GetAllTables(h.DB)

	tables := ""

	for _, item := range ListTable {
		tables += item + " "
	}

	// send array of bytes to client
	w.Write(
		[]byte(
			"The tables in db: " + tables,
		),
	)
}

func (h *Handler) GetRow(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	table := vars["table"]
	id, err := strconv.Atoi(vars["id"])

	if err != nil {
		log.Fatal("[GetRow] GET '/table/id'. Bad converted id to int.\n Error: ", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	row := h.DB.QueryRow("USE [db_golang]; SELECT * FROM ? WHERE id = ?", table, id)

	err = row.Scan()
}

func GetAllTables(db *sql.DB) []string {

	// make request to db
	rows, err := db.Query("USE [db_golang]; SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES;")

	//auto close after returns
	defer rows.Close()

	if err != nil {
		log.Fatal("[GetAllTables] Bad query to db!\nError: ", err.Error())
		return make([]string, 0)
	}

	ListTable := make([]string, 2)

	// iteration over returned query from db and read data
	for rows.Next() {

		table := ""

		err = rows.Scan(&table)

		if err != nil {
			log.Fatal("[TableList] Bad scanned table name!\nError: ", err.Error())
			return make([]string, 0)
		}

		ListTable = append(ListTable, table)
	}

	return ListTable
}

func NewDBExplorer(db *sql.DB) (http.Handler, error) {
	// тут вы пишете код
	// обращаю ваше внимание - в этом задании запрещены глобальные переменные

	handler := &Handler{
		DB: db,
	}

	r := mux.NewRouter()

	r.HandleFunc("/", handler.TableList).Methods("GET")
	r.HandleFunc("/{table}/{id}", handler.GetRow).Methods("GET")

	// rows, err := db.Query("SELECT id, title, updated FROM items")

	// if err != nil {
	// 	panic(err)
	// }

	// for rows.Next() {
	// 	post := &Item{}
	// 	err = rows.Scan(&post.Id, &post.Title, &post.Updated)

	// 	if err != nil {
	// 		panic(err)
	// 	}

	// 	fmt.Println(post.Id, " | ", post.Title, " | ", post.Updated.String, " |")
	// }

	// rows.Close()

	return r, nil
}
