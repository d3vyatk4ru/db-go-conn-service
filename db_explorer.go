package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

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
	Tmpl  *template.Template
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

	resp := fmt.Sprintf("The tables in db: %v", tables)

	log.Println(resp)

	// send array of bytes to client
	w.Write(
		[]byte(resp),
	)
}

func (h *Handler) GetRecordById(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	vars := mux.Vars(r)

	table := vars["table"]

	cond, idx, err := contains(h.Table, table)

	if !cond {
		log.Println(
			fmt.Sprintf("[GetRowById] Bad table in endpoint! Table %v is not exist!\nError: %v", table, err.Error()),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, err := strconv.Atoi(vars["id"])

	if err != nil {
		log.Println(
			fmt.Sprintf("[GetRowById] GET '/%v/%v'. Bad converted id to int.\n Error: %v", table, vars["id"], err.Error()),
		)
		http.Error(w, err.Error(), 500)
		return
	}

	values := getColumnInfo(h.Table[idx])

	err = h.
		DB.
		QueryRow(
			fmt.Sprintf("USE [db_golang]; SELECT * FROM %s WHERE %s = %v", h.Table[idx].Name, h.Table[idx].Id, id),
		).
		Scan(values...)

	if err != nil {
		log.Println(
			fmt.Sprintf("[GetRowById] GET '/%v/%v'. Bad scanned to table %v.\n Error: %v", h.Table[idx].Name, vars["id"], table, err.Error()),
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

	log.Println(result)

	w.Write(result)
}

func (h *Handler) tableHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == "GET" {

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

		if limit == "" && offset == "" {

			limit = "5"
			offset = "0"

		}

		l, err := strconv.Atoi(limit)

		if err != nil {
			log.Println(
				fmt.Sprintf("[GetRows] GET '/%v?limit=%v&offfset=%v'. Bad converted offset to int.\n Error: %v", table, limit, offset, err.Error()),
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

		query := fmt.Sprintf(
			"USE [db_golang]; SELECT * FROM %v ORDER BY %v OFFSET %d ROWS FETCH NEXT %d ROWS ONLY",
			h.Table[idx].Name, h.Table[idx].Id, o, l,
		)

		log.Println(query)

		rows, err := h.DB.Query(query)

		defer rows.Close()

		if err != nil {
			log.Println(
				fmt.Sprintf("[GetRows] GET '/%v?limit=%v&offfset=%v'. Bad query to table %v.\n Error: %v", table, l, o, table, err.Error()),
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

		log.Println(data)

		w.Write(result)
	}
}

func (h *Handler) CreateRecord(w http.ResponseWriter, r *http.Request) {

	if r.Method != "PUT" {
		log.Println(
			fmt.Sprintf("[CreateRecord] Bad HTTP Method. Need PUT, got %v", r.Method),
		)
		http.Error(w, http.StatusText(405), 405)
	}

	table := mux.Vars(r)["table"]

	cond, idx, err := contains(h.Table, table)

	if !cond {
		log.Println(
			fmt.Sprintf("[CreateRecord] Bad table in endpoint! Table %v is not exist!\nError: %v", table, err.Error()),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	item, columnForQuery, placeholder := preprareInsertData(h.Table[idx], r)

	query := fmt.Sprintf(
		"SET IDENTITY_INSERT %v ON; INSERT INTO %v (%v) VALUES (%v); SET IDENTITY_INSERT %v OFF;",
		h.Table[idx].Name, h.Table[idx].Name, columnForQuery, placeholder, h.Table[idx].Name,
	)

	log.Println(query)

	result, err := h.DB.Exec(
		query,
		item...,
	)

	if err != nil {
		log.Println(
			fmt.Sprintf("[CreateRecord] Bad Execute query! \nError: %v", err.Error()),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	affected, err := result.RowsAffected()

	if err != nil {
		log.Println(
			fmt.Sprintf("[CreateRecord] Bad called RowsAffected()! \nError: %v", err.Error()),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(
		[]byte(
			fmt.Sprintf("Rows affected %v", affected),
		),
	)

	log.Println(
		"Inserted: ", item,
	)

	return
}

func (h *Handler) UpdateRecord(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {

		log.Println(
			fmt.Sprintf("[UpdateRecord] Bad HTTP Method. Need POST, got %v", r.Method),
		)

		http.Error(w, http.StatusText(405), 405)
	}

	vars := mux.Vars(r)

	table := vars["table"]

	cond, idx, err := contains(h.Table, table)

	if !cond {
		log.Println(
			fmt.Sprintf("[UpdateRecord] Bad table in endpoint! Table %v is not exist!\nError: %v", table, err.Error()),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, err := strconv.Atoi(vars["id"])

	if err != nil {
		log.Println(
			fmt.Sprintf("[UpdateRecord] POST '/%v/%v'. Bad converted id to int.\n Error: %v", table, vars["id"], err.Error()),
		)
		http.Error(w, err.Error(), 500)
		return
	}

	nameId, updQuery, item := preprareUpdateQuery(h.Table[idx], r)

	query := fmt.Sprintf(
		"UPDATE %v SET %v WHERE %v = %d", h.Table[idx].Name, updQuery, nameId, id,
	)

	log.Println(
		query,
	)

	result, err := h.DB.Exec(
		query,
		item...,
	)

	if err != nil {
		log.Println(
			fmt.Sprintf("[UpdateRecord] Bad Execute query! \nError: %v", err.Error()),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	affected, err := result.RowsAffected()

	if err != nil {
		log.Println(
			fmt.Sprintf("[UpdateRecord] Bad called RowsAffected()! \nError: %v", err.Error()),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := "Row affected " + fmt.Sprint(affected) + "\n" +
		"Record with id = " + fmt.Sprint(id) + "was updated"

	log.Println(resp)

	w.Write(
		[]byte(resp),
	)
}

func (h *Handler) DeleteRecordById(w http.ResponseWriter, r *http.Request) {

	if r.Method != "DELETE" {
		log.Println(
			fmt.Sprintf("[DeleteRecordById] Bad HTTP Method. Need DELETE, got %v", r.Method),
		)

		http.Error(w, http.StatusText(405), 405)
	}

	vars := mux.Vars(r)

	table := vars["table"]

	cond, idx, err := contains(h.Table, table)

	if !cond {
		log.Println(
			fmt.Sprintf("[DeleteRecordById] Bad table in endpoint! Table %v is not exist!\nError: %v", table, err.Error()),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, err := strconv.Atoi(vars["id"])

	if err != nil {
		log.Println(
			fmt.Sprintf("[DeleteRecordById] DELETE '/%v/%v'. Bad converted id to int.\n Error: %v", table, vars["id"], err.Error()),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var NameID string = ""

	for _, field := range h.Table[idx].Fields {

		if field.IsKey {
			NameID = field.Name
		}

	}

	query := fmt.Sprintf(
		"DELETE FROM %v WHERE %v = %d", h.Table[idx].Name, NameID, id,
	)

	log.Println(query)

	result, err := h.DB.Exec(query)

	if err != nil {
		log.Println(
			fmt.Sprintf("[DeleteRecordById] Bad Execute query! \nError: %v", err.Error()),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	affected, err := result.RowsAffected()

	if err != nil {
		log.Println(
			fmt.Sprintf("[DeleteRecordById] Bad called RowsAffected()! \nError: %v", err.Error()),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := "Row affected " + fmt.Sprint(affected) + "\n" +
		"Record with id = " + fmt.Sprint(id) + "was deleted"

	log.Println(resp)

	w.Write(
		[]byte(resp),
	)
}

func preprareInsertData(table TableInfo, r *http.Request) ([]interface{}, string, string) {

	item := make([]interface{}, len(table.Fields))
	columnForQuery := make([]string, len(table.Fields))
	placeholder := make([]string, len(table.Fields))

	for i, field := range table.Fields {

		param := r.FormValue(field.Name)

		if param != "" {
			item[i] = param
		} else {
			switch field.Type {
			case "int":
				item[i] = 0
			case "nvarchar", "text":
				item[i] = ""
			}
		}

		columnForQuery[i] = field.Name
		placeholder[i] = fmt.Sprintf("@p%d", i+1)
	}

	return item, strings.Join(columnForQuery, ","), strings.Join(placeholder, ",")
}

func preprareUpdateQuery(table TableInfo, r *http.Request) (string, string, []interface{}) {

	columnQuery := make([]string, 0)
	item := make([]interface{}, 0)

	nameId := ""

	var i int64 = 0

	for _, field := range table.Fields {

		param := r.FormValue(field.Name)

		if field.IsKey {
			nameId = field.Name
			continue
		}

		if param != "" {
			item = append(item, param)
		} else {
			switch field.Type {
			case "int":
				item = append(item, 0)
			case "nvarchar", "text":
				item = append(item, "")
			}
		}

		columnQuery = append(columnQuery, fmt.Sprintf("%v = @p%d", field.Name, i+1))

		i += 1
	}

	return nameId, strings.Join(columnQuery, ","), item
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

	handler := &Handler{
		DB:   db,
		Tmpl: template.Must(template.ParseGlob("templates/*")),
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
	r.HandleFunc("/{table}/{id:[0-9]+}", handler.GetRecordById).Methods("GET")
	r.HandleFunc("/{table}", handler.tableHandler).Methods("GET")
	r.HandleFunc("/{table}", handler.CreateRecord).Methods("PUT")
	r.HandleFunc("/{table}/{id:[0-9]+}", handler.UpdateRecord).Methods("POST")
	r.HandleFunc("/{table}/{id:[0-9]+}", handler.DeleteRecordById).Methods("DELETE")

	return r, nil
}
