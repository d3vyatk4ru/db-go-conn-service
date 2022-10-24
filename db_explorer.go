package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

type FieldInfo struct {
	Name       string
	ColumnType string
	IsKey      bool
	CouldNull  bool
}

type TableInfo struct {
	Name   string
	ID     string
	Fields []FieldInfo
}

type Handler struct {
	DB    *sql.DB
	Table []TableInfo
}

// Хендлер для списка всех таблиц. Вызывается по эндпоинту "/" [GET]
func (h *Handler) TableList(w http.ResponseWriter, r *http.Request) {

	ListTable, err := GetAllTables(h.DB)

	if err != nil {
		log.Fatal("[TableList] GET '/'.\n Error: ", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tables := []string{}

	for _, item := range ListTable {
		tables = append(tables, item)
	}

	log.Println(tables)

	result, err := json.Marshal(
		map[string]interface{}{
			"response": map[string][]string{
				"tables": tables,
			},
		},
	)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	// send array of bytes to client
	w.Write(result)
}

// Хендлер для получния записи / записей по id. Вызывается по эндпоинту "/{table}/{id}" [GET]
func (h *Handler) SelectRecordByID(w http.ResponseWriter, r *http.Request) {

	table := mux.Vars(r)["table"]

	idx, err := CheckTableExist(w, h.Table, table, "GetRecordById")

	if err != nil {
		return
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])

	if err != nil {
		log.Println(
			fmt.Sprintf("[GetRecordById] GET '/%v/%v'. Bad converted id to int.\n Error: %v",
				table, mux.Vars(r)["id"], err.Error(),
			))

		w.WriteHeader(http.StatusBadRequest)

		return
	}

	values := ColumnsType(h.Table[idx])

	columns := GetColumnsTable(h.Table[idx], r, false)

	err = h.DB.
		QueryRow(
			fmt.Sprintf("SELECT %s FROM %s WHERE %s = %v",
				columns, h.Table[idx].Name, h.Table[idx].ID, id),
		).
		Scan(values...)

	if err != nil {
		log.Println(
			fmt.Sprintf("[GetRecordById] GET '/%v/%v'. Bad scanned to table %v. Error: %v",
				h.Table[idx].Name, mux.Vars(r)["id"], table, err.Error(),
			))

		result, _ := json.Marshal(
			map[string]string{
				"error": "record not found",
			})

		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		w.Write(result)

		return
	}

	data := CastType(values, h.Table[idx])

	result, err := json.Marshal(
		map[string]interface{}{
			"response": map[string]interface{}{
				"record": data,
			},
		},
	)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "appliсation/json")
	w.Write(result)
}

// Хендлер для полуения записений из таблицы с лимитом и оффсетом.
// Вызывается по эндпоинту "/{table}&offset=a&limit=b" [GET]
func (h *Handler) SelectRecord(w http.ResponseWriter, r *http.Request) {

	table := mux.Vars(r)["table"]

	idx, err := CheckTableExist(w, h.Table, table, "GetRecordById")

	if err != nil {
		return
	}

	limit := r.FormValue("limit")

	if limit == "" {
		limit = "5"
	}

	lim, err := strconv.Atoi(limit)
	if err != nil {
		lim = 5
	}

	offset := r.FormValue("offset")

	if offset == "" {
		offset = "0"
	}

	off, err := strconv.Atoi(offset)
	if err != nil {
		off = 0
	}

	values := ColumnsType(h.Table[idx])

	columns := GetColumnsTable(h.Table[idx], r, false)

	log.Println("columns: ", columns)

	query := fmt.Sprintf(
		"SELECT %s FROM %v ORDER BY %v LIMIT %d, %d",
		columns, h.Table[idx].Name, h.Table[idx].ID, off, lim,
	)

	log.Println(query)

	rows, err := h.DB.Query(query)

	defer rows.Close()

	if err != nil {
		log.Println(
			fmt.Sprintf("[TableContain] GET '/%v?limit=%v&offfset=%v'. Bad query to table %v.\n Error: %v",
				table, lim, off, table, err.Error(),
			))
		w.WriteHeader(500)
		return
	}

	data := make([]interface{}, 0)

	for rows.Next() {

		err = rows.Scan(values...)

		if err != nil {
			log.Println(
				fmt.Sprintf("[TableContain] GET '/%v?limit=%v&offfset=%v'. Bad scanned to table %v.\n Error: %v",
					table, limit, offset, table, err.Error(),
				))
			w.WriteHeader(500)
			return
		}

		data = append(data, CastType(values, h.Table[idx]))
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
			fmt.Sprintf("[TableContain] GET '/%v?limit=%v&offfset=%v'. Bad json marshal.\n Error: %v",
				table, limit, offset, err.Error(),
			))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Println(data)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(result)
}

// Хендлер для создания новой записи. Параметры передаются в теле.
// Вызывается по эндпоинту "/{table}" [PUT]
func (h *Handler) CreateRecord(w http.ResponseWriter, r *http.Request) {

	table := mux.Vars(r)["table"]

	idx, err := CheckTableExist(w, h.Table, table, "CreateRecord")

	if err != nil {
		return
	}

	item, placeholder := MakeContainerInsert(h.Table[idx], r)
	columns := GetColumnsTable(h.Table[idx], r, true)

	query := fmt.Sprintf(
		"INSERT INTO %v (%v) VALUES (%v);",
		h.Table[idx].Name, columns, placeholder,
	)

	log.Println(query)

	res, err := h.DB.Exec(query, item...)

	if err != nil {
		log.Println(
			fmt.Sprintf("[CreateRecord] Bad Execute query! Error: %v",
				err.Error(),
			))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	LastID, err := res.LastInsertId()

	if err != nil {
		log.Println(
			fmt.Sprintf("[CreateRecord] Bad called RowsAffected()! Error: %v",
				err.Error(),
			))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	NameID := GetIdColumnName(h.Table[idx])

	result, err := json.Marshal(
		map[string]interface{}{
			"response": map[string]int{
				fmt.Sprintf("%s", NameID): int(LastID),
			},
		},
	)

	w.Write(result)

	log.Println("Inserted ID:", LastID)
}

// Хендлер для обновлени существующей записи по ID. Параметры передаются в теле.
// Вызывается по эндпоинту "/{table}/{id}". [POST]
func (h *Handler) UpdateRecord(w http.ResponseWriter, r *http.Request) {

	table := mux.Vars(r)["table"]

	idx, err := CheckTableExist(w, h.Table, table, "UpdateRecord")

	if err != nil {
		return
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])

	if err != nil {
		log.Println(
			fmt.Sprintf("[UpdateRecord] POST '/%v/%v'. Bad converted id to int. Error: %v",
				table, mux.Vars(r)["id"], err.Error(),
			))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ColumnIdName, placeholder, item, err := CheckParamsAndTypes(h.Table[idx], r)

	if err != nil {
		log.Println(
			fmt.Sprintf("Cant update %s", ColumnIdName),
		)

		result, _ := json.Marshal(
			map[string]string{
				"error": fmt.Sprintf("field %s have invalid type", ColumnIdName),
			},
		)

		w.WriteHeader(http.StatusBadRequest)
		w.Write(result)
		return
	}

	log.Println("placeholder:", placeholder)

	if strings.Contains(placeholder, ColumnIdName) {

		log.Println(
			fmt.Sprintf("Cant update %s", ColumnIdName),
		)

		result, _ := json.Marshal(
			map[string]string{
				"error": fmt.Sprintf("field %s have invalid type", ColumnIdName),
			},
		)

		w.WriteHeader(http.StatusBadRequest)
		w.Write(result)
		return
	}

	query := fmt.Sprintf(
		"UPDATE %v SET %v WHERE %v = %d",
		h.Table[idx].Name, placeholder, ColumnIdName, id,
	)

	log.Println(query)

	res, err := h.DB.Exec(query, item...)

	if err != nil {
		log.Println(
			fmt.Sprintf("[UpdateRecord] Bad Execute query! Error: %v",
				err.Error(),
			))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	affected, err := res.RowsAffected()

	if err != nil {
		log.Println(
			fmt.Sprintf("[UpdateRecord] Bad called RowsAffected()! Error: %v",
				err.Error(),
			))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	result, err := json.Marshal(
		map[string]interface{}{
			"response": map[string]int{
				"updated": int(affected),
			},
		},
	)

	w.Write(result)

	log.Println("Row affected:", affected)
}

// Хендлер для удлаения записи по ID. Вызывается по эндпоинту "/{table}/{id}". [DELETE]
func (h *Handler) DeleteRecord(w http.ResponseWriter, r *http.Request) {

	table := mux.Vars(r)["table"]

	idx, err := CheckTableExist(w, h.Table, table, "DeleteRecord")

	if err != nil {
		return
	}

	id, err := strconv.Atoi(mux.Vars(r)["id"])

	if err != nil {
		log.Println(
			fmt.Sprintf("[DeleteRecord] DELETE '/%v/%v'. Bad converted id to int. Error: %v",
				table, mux.Vars(r)["id"], err.Error(),
			))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	NameID := GetIdColumnName(h.Table[idx])

	query := fmt.Sprintf(
		"DELETE FROM %v WHERE %v = %d", h.Table[idx].Name, NameID, id,
	)

	log.Println(query)

	res, err := h.DB.Exec(query)

	if err != nil {
		log.Println(
			fmt.Sprintf("[DeleteRecord] Bad Execute query! Error: %v", err.Error()),
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	affected, err := res.RowsAffected()

	if err != nil {
		log.Println(
			fmt.Sprintf("[DeleteRecord] Bad called RowsAffected()! \nError: %v", err.Error()),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result, _ := json.Marshal(
		map[string]interface{}{
			"response": map[string]int{
				"deleted": int(affected),
			},
		},
	)

	log.Println(result)

	w.Write(result)
}

// Возвращаем имя столбца, который явл. ID
func GetIdColumnName(table TableInfo) string {

	for _, field := range table.Fields {

		if field.IsKey {
			return field.Name
		}
	}

	return ""
}

// Проверка наличия таблицы в БД и отправка ошибки
func CheckTableExist(w http.ResponseWriter, tableInfo []TableInfo, table string, funcName string) (int, error) {

	cond, idx, err := contains(tableInfo, table)

	if !cond {

		result, _ := json.Marshal(
			map[string]string{
				"error": "unknown table",
			},
		)

		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		w.Write(result)

		log.Println(
			fmt.Sprintf("[%s] Bad table in endpoint! Table %v is not exist!\nError: %v",
				funcName, table, err.Error(),
			))

		return -1, err
	}

	return idx, nil
}

// Проверка наличия конкретной таблицы
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

// Смотрим имена столбцов в таблице
func GetColumnsTable(table TableInfo, r *http.Request, key bool) string {

	columns := make([]string, 0)

	for _, field := range table.Fields {

		if key {
			if field.IsKey {
				continue
			}
		}

		columns = append(columns, field.Name)
	}

	return strings.Join(columns, ",")
}

// Создаем контейнер для записи значений из БД и плейсхолдер
func MakeContainerInsert(table TableInfo, r *http.Request) ([]interface{}, string) {

	item := make([]interface{}, 0)
	placeholder := make([]string, 0)

	decoder := json.NewDecoder(r.Body)
	param := make(map[string]interface{}, len(table.Fields))
	decoder.Decode(&param)

	for _, field := range table.Fields {

		if field.IsKey {
			continue
		}

		if param[field.Name] != nil {
			item = append(item, param[field.Name])
		} else {
			if field.CouldNull {
				item = append(item, nil)
			} else {
				switch field.ColumnType {
				case "int":
					item = append(item, 0)
				case "varchar", "text":
					item = append(item, "")
				}
			}
		}

		placeholder = append(placeholder, "?")
	}

	return item, strings.Join(placeholder, ",")
}

// Проверка типов параметров, которые пришли в реквесте. Делаем placeholders
func CheckParamsAndTypes(table TableInfo, r *http.Request) (string, string, []interface{}, error) {

	var columnIdName string

	item := make([]interface{}, 0)
	placeholder := make([]string, 0)

	decoder := json.NewDecoder(r.Body)
	param := make(map[string]interface{}, len(table.Fields))
	decoder.Decode(&param)

	for _, field := range table.Fields {

		if field.IsKey {
			columnIdName = field.Name
			break
		}
	}

	for key, val := range param {

		for _, field := range table.Fields {

			if field.Name == key {

				if val == nil && !field.CouldNull {
					return field.Name, "", make([]interface{}, 0), fmt.Errorf("%s column cant use this type", field.Name)
				}

				switch val.(type) {

				case string:

					if field.ColumnType != "varchar" && field.ColumnType != "text" {
						return field.Name, "", make([]interface{}, 0), fmt.Errorf("%s column cant use this type", field.Name)
					}

				case float64:
					if field.ColumnType != "int" {
						return field.Name, "", make([]interface{}, 0), fmt.Errorf("%s column cant use this type", field.Name)
					}
				}
			}
		}

		item = append(item, val)

		placeholder = append(placeholder, fmt.Sprintf("%v = ?", key))
	}

	return columnIdName, strings.Join(placeholder, ","), item, nil
}

// Возвращаем интерфейс с подготовленными типами для
// чтения записи из таблицы table
func ColumnsType(table TableInfo) []interface{} {

	values := make([]interface{}, len(table.Fields))

	for i, field := range table.Fields {
		switch field.ColumnType {
		case "int":
			values[i] = new(sql.NullInt64)
		case "varchar", "text":
			values[i] = new(sql.NullString)
		}
	}

	return values
}

// Делаем каст к гошным типам
func CastType(record []interface{}, table TableInfo) map[string]interface{} {

	item := make(map[string]interface{}, len(record))

	for idx, value := range record {

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

// Информация о все таблицах в БД
func GetAllTables(db *sql.DB) ([]string, error) {

	ListTable := make([]string, 0)

	// make request to db. Get all tables name
	rows, err := db.Query(`
		SELECT
			Table_name
		FROM
			information_schema.tables
		WHERE
			table_schema = 'golang';`)

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

// Получение информации о всех столбцах таблицы
func GetTablesInfo(db *sql.DB) ([]TableInfo, error) {

	tableInfo := []TableInfo{}

	ListTable, err := GetAllTables(db)

	if err != nil {
		log.Println(
			fmt.Sprintf("[GetTableInfo]. Bad calling from [GetAllTables] function! Error: %v",
				err.Error(),
			))

		return nil, err
	}

	for _, table := range ListTable {

		fieldInfo := []FieldInfo{}

		var nameId string

		row := db.QueryRow(
			fmt.Sprintf(
				`SELECT
				   	COLUMN_NAME 
				 FROM
				   	INFORMATION_SCHEMA.COLUMNS
				 WHERE
				 	TABLE_SCHEMA = 'golang' AND
					TABLE_NAME = '%s' AND
					COLUMN_KEY = 'PRI'`,
				table,
			),
		)

		err := row.Scan(&nameId)

		if err != nil {
			log.Println(
				fmt.Sprintf("[GetTableInfo] Bad scanned column id from table %v. Error: %v",
					table, err.Error(),
				))
			return nil, err
		}

		// make query for getting column name and column type into $table
		rows, err := db.Query(
			fmt.Sprintf(
				`SELECT
					COLUMN_NAME, DATA_TYPE , IS_NULLABLE
		  		 FROM
					INFORMATION_SCHEMA.COLUMNS 
		  		 WHERE
					TABLE_SCHEMA = 'golang' AND
					TABLE_NAME = '%s'`,
				table,
			),
		)

		if err != nil {
			log.Println(
				fmt.Sprintf("[GetTableInfo] Bad query to table %v. COLUMN_NAME, DATA_TYPE, IS_NULLABLE! Error: %v",
					table, err.Error(),
				))

			return nil, err
		}

		for rows.Next() {

			var fieldName string = ""
			var fieldType string = ""
			var fieldNull string = ""
			var isKey bool = false

			err = rows.Scan(&fieldName, &fieldType, &fieldNull)

			if err != nil {
				log.Println(
					fmt.Sprintf("[GetTableInfo] Bad scanned COLUMN_NAME and DATA_TYPE from %v! Error: %v",
						table, err.Error(),
					))

				return nil, err
			}

			if fieldName == nameId {
				isKey = true
			}

			var null bool

			if fieldNull == "YES" {
				null = true
			}

			fieldInfo = append(
				fieldInfo,
				FieldInfo{
					Name:       fieldName,
					ColumnType: fieldType,
					IsKey:      isKey,
					CouldNull:  null,
				},
			)
		}

		tableInfo = append(
			tableInfo,
			TableInfo{
				Name:   table,
				ID:     nameId,
				Fields: fieldInfo,
			},
		)
	}

	log.Println(tableInfo)

	return tableInfo, nil
}

func NewDBExplorer(db *sql.DB) (http.Handler, error) {

	handler := &Handler{
		DB: db,
	}

	tableInfo, err := GetTablesInfo(db)

	if err != nil {
		log.Println(
			fmt.Sprintf("[NewDBExplorer] Bad calling from [GetTablesInfo] function! Error: %v",
				err.Error(),
			))

		return nil, err
	}

	handler.Table = tableInfo

	r := mux.NewRouter()

	r.HandleFunc("/", handler.TableList).Methods("GET")
	r.HandleFunc("/{table}/{id:[0-9]+}", handler.SelectRecordByID).Methods("GET")
	r.HandleFunc("/{table}/", handler.CreateRecord).Methods("PUT")
	r.HandleFunc("/{table}", handler.SelectRecord).Methods("GET")
	r.HandleFunc("/{table}/{id:[0-9]+}", handler.UpdateRecord).Methods("POST")
	r.HandleFunc("/{table}/{id:[0-9]+}", handler.DeleteRecord).Methods("DELETE")

	return r, nil
}
