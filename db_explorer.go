package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Table []string

type FieldInfo struct {
	Name  string
	Type  string
	IsKey bool
}

type TableInfo struct {
	Name   string
	Id     string
	Fields []FieldInfo
}

type Handler struct {
	DB    *sql.DB
	Table []TableInfo
}

func (h *Handler) TableList(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	ListTable, err := GetAllTables(h.DB)

	if err != nil {
		log.Fatal("[GetRow] GET '/'.\n Error: ", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tables := ""

	for _, item := range ListTable {
		tables += item + " "
	}

	// send array of bytes to client
	w.Write(
		[]byte(
			fmt.Sprintf("The tables in db: %v", tables),
		),
	)
}

func (h *Handler) GetRowById(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	vars := mux.Vars(r)

	table := vars["table"]

	cond, idx, err := contains(h.Table, table)

	if !cond {
		log.Println(
			fmt.Sprintf("[GetRow] Bad table in endpoint! Table %v is not exist!\nError: %v", table, err.Error()),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, err := strconv.Atoi(vars["id"])

	if err != nil {
		log.Println(
			fmt.Sprintf("[GetRow] GET '/%v/%v'. Bad converted id to int.\n Error: %v", table, vars["id"], err.Error()),
		)
		http.Error(w, err.Error(), 500)
		return
	}

	values := getColumnInfo(h.Table[idx])

	err = h.
		DB.
		QueryRow(
			fmt.Sprintf("USE [db_golang]; SELECT * FROM %s WHERE %s = %v", table, h.Table[idx].Id, id),
		).
		Scan(values...)

	if err != nil {
		log.Println(
			fmt.Sprintf("[GetRow] GET '/%v/%v'. Bad scanned to table %v.\n Error: %v", table, vars["id"], table, err.Error()),
		)
		http.Error(w, err.Error(), 500)
		return
	}

	data := tranformQueryResult(values, h.Table[idx])

	result, err := json.Marshal(
		map[string]interface{}{
			"response": map[string]interface{}{
				"records": data,
			},
		},
	)

	w.Write(result)
}

func (h *Handler) GetRows(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	table := mux.Vars(r)["table"]

	cond, idx, err := contains(h.Table, table)

	if !cond {
		log.Println(
			fmt.Sprintf("[GetRows] Bad table in endpoint! Table %v is not exist!\nError: %v", table, err.Error()),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	limit := r.FormValue("limit")
	offset := r.FormValue("offset")

	l, err := strconv.Atoi(limit)

	if err != nil {
		log.Println(
			fmt.Sprintf("[GetRow] GET '/%v?limit=%v&offfset=%v'. Bad converted offset to int.\n Error: %v", table, limit, offset, err.Error()),
		)
		http.Error(w, err.Error(), 500)
		return
	}

	o, err := strconv.Atoi(offset)

	if err != nil {
		log.Println(
			fmt.Sprintf("[GetRow] GET '/%v?limit=%v&offfset=%v'. Bad converted limit to int.\n Error: %v", table, l, o, err.Error()),
		)
		http.Error(w, err.Error(), 500)
		return
	}

	values := getColumnInfo(h.Table[idx])

	rows, err := h.DB.Query(
		fmt.Sprintf("USE [db_golang]; SELECT * FROM %v ORDER BY %v OFFSET %v ROWS", table, h.Table[idx].Id, offset),
	)

	if err != nil {
		log.Println(
			fmt.Sprintf("[GetRows] GET '/%v?limit=%v&offfset=%v'. Bad query to table %v.\n Error: %v", table, limit, offset, table, err.Error()),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0)

	for rows.Next() {

		err = rows.Scan(values...)

		if err != nil {
			log.Println(
				fmt.Sprintf("[GetRows] GET '/%v?limit=%v&offfset=%v'. Bad scanned to table %v.\n Error: %v", table, limit, offset, table, err.Error()),
			)
			http.Error(w, err.Error(), 500)
			return
		}

		data = append(data, tranformQueryResult(values, h.Table[idx]))
	}

	result, err := json.Marshal(
		map[string]interface{}{
			"response": map[string]interface{}{
				"records": data,
			},
		},
	)

	if err != nil {
		log.Println(
			fmt.Sprintf("[GetRows] GET '/%v?limit=%v&offfset=%v'. Bad json marshal.\n Error: %v", table, limit, offset, err.Error()),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(result)
}

func getColumnInfo(table TableInfo) []interface{} {

	values := make([]interface{}, len(table.Fields))

	for i, field := range table.Fields {
		switch field.Type {
		case "int":
			values[i] = new(sql.NullInt64)
		case "nvarchar", "text":
			values[i] = new(sql.NullString)
		}
	}

	return values
}

func tranformQueryResult(row []interface{}, table TableInfo) map[string]interface{} {

	item := make(map[string]interface{}, len(row))

	for idx, value := range row {

		switch value.(type) {

		case *sql.NullString:
			if v, ok := value.(*sql.NullString); ok {
				if v.Valid {
					item[table.Fields[idx].Name] = v.String
				} else {
					item[table.Fields[idx].Name] = nil
				}
			}

		case *sql.NullInt64:
			if v, ok := value.(*sql.NullInt64); ok {
				if v.Valid {
					item[table.Fields[idx].Name] = v.Int64
				} else {
					item[table.Fields[idx].Name] = nil
				}
			}
		}
	}
	return item
}

func contains(s []TableInfo, table string) (bool, int, error) {
	for idx, v := range s {
		if v.Name == table {
			return true, idx, nil
		}
	}

	return false, -1, errors.New(
		fmt.Sprintf("The database not contain table %v", table),
	)
}

func GetAllTables(db *sql.DB) ([]string, error) {

	ListTable := make([]string, 0)

	// make request to db. Get all tables name
	rows, err := db.Query("USE [db_golang]; SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES;")

	//auto close after returns
	defer rows.Close()

	if err != nil {
		log.Fatal("[GetAllTables] Bad query to db!\nError: ", err.Error())
		return ListTable, err
	}

	// iteration over returned query from db and read data
	for rows.Next() {

		table := ""

		err = rows.Scan(&table)

		if err != nil {
			log.Fatal("[TableList] Bad scanned table name!\nError: ", err.Error())
			return make([]string, 0), err
		}

		ListTable = append(ListTable, table)
	}

	return ListTable, nil
}

func GetTablesInfo(db *sql.DB) ([]TableInfo, error) {

	tableInfo := []TableInfo{}

	ListTable, err := GetAllTables(db)

	if err != nil {
		log.Println(
			fmt.Sprintf("[GetTableInfo]. Bad calling from [GetAllTables] function!\nError: %v", err.Error()),
		)

		return nil, err
	}

	for _, table := range ListTable {

		fieldInfo := []FieldInfo{}

		var nameId string

		// query to getting name of PK column
		row := db.QueryRow(
			"SELECT K.COLUMN_NAME FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS T JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE K ON K.CONSTRAINT_NAME=T.CONSTRAINT_NAME WHERE K.TABLE_NAME=@table AND T.CONSTRAINT_TYPE='PRIMARY KEY';",
			sql.Named("table", table),
		)

		err := row.Scan(&nameId)

		if err != nil {
			log.Println(
				fmt.Sprintf("[GetTableInfo] Bad scanned column id from table %v .\nError: %v", table, err.Error()),
			)
			return nil, err
		}

		// make query for getting column name and column type into $table
		rows, err := db.Query(
			"SELECT COLUMN_NAME, DATA_TYPE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = @table",
			sql.Named("table", table),
		)

		if err != nil {
			log.Println(
				fmt.Sprintf("[GetTableInfo] Bad query to table %v. COLUMN_NAME and DATA_TYPE!\nError: %v", table, err.Error()),
			)

			return nil, err
		}

		for rows.Next() {

			var fieldName string = ""
			var fieldType string = ""
			var isKey bool = false

			err = rows.Scan(&fieldName, &fieldType)

			if err != nil {
				log.Println(
					fmt.Sprintf("[GetTableInfo] Bad scanned COLUMN_NAME and DATA_TYPE from %v!\nError: %v", table, err.Error()),
				)

				return nil, err
			}

			if fieldName == nameId {
				isKey = true
			}

			fieldInfo = append(
				fieldInfo,
				FieldInfo{
					Name:  fieldName,
					Type:  fieldType,
					IsKey: isKey,
				},
			)
		}

		tableInfo = append(
			tableInfo,
			TableInfo{
				Name:   table,
				Id:     nameId,
				Fields: fieldInfo,
			},
		)
	}

	return tableInfo, nil
}

func NewDBExplorer(db *sql.DB) (http.Handler, error) {
	// тут вы пишете код
	// обращаю ваше внимание - в этом задании запрещены глобальные переменные

	handler := &Handler{
		DB: db,
	}

	tableInfo, err := GetTablesInfo(db)

	if err != nil {
		log.Println(
			fmt.Sprintf("[NewDBExplorer] Bad calling from [GetTablesInfo] function!\nError: %v", err.Error()),
		)

		return nil, err
	}

	handler.Table = tableInfo

	r := mux.NewRouter()

	r.HandleFunc("/", handler.TableList).Methods("GET")
	r.HandleFunc("/{table}/{id}", handler.GetRowById).Methods("GET")
	r.HandleFunc("/{table}", handler.GetRows).Methods("GET")

	return r, nil
}