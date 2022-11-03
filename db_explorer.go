package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var TypeInt = "int"
var TypeText = "text"
var TypeVarchar = "varchar(255)"

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

type Columns struct {
	Field   string
	Type    string
	Null    string
	Key     string
	Default interface{}
	Extra   string
}

// Хендлер для списка всех таблиц. Вызывается по эндпоинту "/" [GET]
func (h *Handler) TableList(w http.ResponseWriter) {

	tables := make([]string, 0)

	for _, table := range h.Table {
		tables = append(tables, table.Name)
	}

	log.Println(tables)

	result, err := json.Marshal(
		map[string]interface{}{
			"response": map[string][]string{
				"tables": tables,
			},
		},
	)

	if err != nil {
		log.Println("Bad packed json:", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	// send array of bytes to client
	_, err = w.Write(result)

	if err != nil {
		log.Println("Bad request:", err.Error())
	}
}

// Хендлер для получния записи / записей по id. Вызывается по эндпоинту "/{table}/{id}" [GET]
func (h *Handler) SelectRecordByID(w http.ResponseWriter, r *http.Request) {

	url := r.URL.Path

	params := strings.Split(url, "/")

	table := params[1]

	cond, idx, err := contains(h.Table, table)

	if !cond {

		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		_, err2 := w.Write(
			[]byte(`{"error" : "unknown table"}`),
		)

		if err2 != nil {
			log.Println("Bad packed json:", err2.Error())
		}

		return
	}

	if err != nil {
		return
	}

	id, err := strconv.Atoi(params[2])

	if err != nil {
		log.Printf("[GetRecordById] GET '/%v/%v'. Bad converted id to int.\n Error: %v",
			table, id, err.Error())

		w.WriteHeader(http.StatusBadRequest)
		return
	}

	values := ColumnsType(h.Table[idx])

	columns := GetColumnsTable(h.Table[idx], false)

	err = h.DB.
		QueryRow(
			fmt.Sprintf("SELECT %s FROM %s WHERE %s = %v",
				columns, h.Table[idx].Name, h.Table[idx].ID, id),
		).
		Scan(values...)

	if err != nil {
		log.Printf("[GetRecordById] GET '/%v/%v'. Bad scanned to table %v. Error: %v",
			h.Table[idx].Name, id, table, err.Error())

		result, err2 := json.Marshal(
			map[string]string{
				"error": "record not found",
			})

		if err2 != nil {
			log.Println("Bad packed json:", err.Error())
		}

		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(result)

		if err != nil {
			log.Println("Bad request: ", err.Error())
		}

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

	if err != nil {
		log.Println("Bad packed json")
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "appliсation/json")
	_, err = w.Write(result)

	if err != nil {
		log.Println("Bad request")
	}
}

// Хендлер для полуения записений из таблицы с лимитом и оффсетом.
// Вызывается по эндпоинту "/{table}&offset=a&limit=b" [GET]
func (h *Handler) SelectRecord(w http.ResponseWriter, r *http.Request) {

	url := r.URL.Path

	table := strings.Split(url, "/")[1]

	cond, idx, err := contains(h.Table, table)

	if !cond {

		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		_, err2 := w.Write(
			[]byte(`{"error" : "unknown table"}`),
		)

		if err2 != nil {
			log.Println("Bad packed json:", err2.Error())
		}

		return
	}

	if err != nil {
		return
	}

	limit := r.URL.Query().Get("limit")

	if limit == "" {
		limit = "5"
	}

	lim, err := strconv.Atoi(limit)
	if err != nil {
		lim = 5
	}

	offset := r.URL.Query().Get("offset")

	if offset == "" {
		offset = "0"
	}

	off, err := strconv.Atoi(offset)
	if err != nil {
		off = 0
	}

	values := ColumnsType(h.Table[idx])

	columns := GetColumnsTable(h.Table[idx], false)

	log.Println("columns: ", columns)

	query := fmt.Sprintf(
		"SELECT %s FROM %v ORDER BY %v LIMIT %d, %d",
		columns, h.Table[idx].Name, h.Table[idx].ID, off, lim,
	)

	log.Println(query)

	rows, err := h.DB.Query(query)

	defer rows.Close() //nolint:staticcheck

	if err != nil {
		log.Printf("[TableContain] GET '/%v?limit=%v&offfset=%v'. Bad query to table %v.\n Error: %v",
			table, lim, off, table, err.Error())
		w.WriteHeader(500)
		return
	}

	data := make([]interface{}, 0)

	for rows.Next() {

		err = rows.Scan(values...)

		if err != nil {
			log.Printf("[TableContain] GET '/%v?limit=%v&offfset=%v'. Bad scanned to table %v.\n Error: %v",
				table, limit, offset, table, err.Error())
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
		log.Printf("[TableContain] GET '/%v?limit=%v&offfset=%v'. Bad json marshal.\n Error: %v",
			table, limit, offset, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Println(data)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(result)

	if err != nil {
		log.Println("Bad request")
	}
}

// Хендлер для создания новой записи. Параметры передаются в теле.
// Вызывается по эндпоинту "/{table}" [PUT]
func (h *Handler) CreateRecord(w http.ResponseWriter, r *http.Request) {

	url := r.URL.Path

	params := strings.Split(url, "/")

	table := params[1]

	cond, idx, err := contains(h.Table, table)

	if !cond {

		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		_, err2 := w.Write(
			[]byte(`{"error" : "unknown table"}`),
		)

		if err2 != nil {
			log.Println("Bad packed json:", err2.Error())
		}

		return
	}

	if err != nil {
		return
	}

	item, placeholder, err := MakeContainerInsert(h.Table[idx], r)

	if err != nil {
		w.WriteHeader(500)
		_, err2 := w.Write([]byte(`{"error": "bad unpacked json"}`))

		if err2 != nil {
			log.Println("Bad request:", err2.Error())
		}

		return
	}

	columns := GetColumnsTable(h.Table[idx], true)

	query := fmt.Sprintf(
		"INSERT INTO %v (%v) VALUES (%v);",
		h.Table[idx].Name, columns, placeholder,
	)

	log.Println(query)

	res, err := h.DB.Exec(query, item...)

	if err != nil {
		log.Printf("[CreateRecord] Bad Execute query! Error: %v", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	LastID, err := res.LastInsertId()

	if err != nil {
		log.Printf("[CreateRecord] Bad called RowsAffected()! Error: %v",
			err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	NameID := GetIDColumnName(h.Table[idx])

	result, err := json.Marshal(
		map[string]interface{}{
			"response": map[string]int{
				NameID: int(LastID),
			},
		},
	)

	if err != nil {
		log.Println("Bad packed json:", err.Error())
	}

	_, err = w.Write(result)

	if err != nil {
		log.Println("Bad request:", err.Error())
	}

	log.Println("Inserted ID:", LastID)
}

// Хендлер для обновлени существующей записи по ID. Параметры передаются в теле.
// Вызывается по эндпоинту "/{table}/{id}". [POST]
func (h *Handler) UpdateRecord(w http.ResponseWriter, r *http.Request) {

	url := r.URL.Path

	params := strings.Split(url, "/")

	table := params[1]

	cond, idx, err := contains(h.Table, table)

	if !cond {

		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		_, err2 := w.Write(
			[]byte(`{"error" : "unknown table"}`),
		)

		if err2 != nil {
			log.Println("Bad packed json:", err2.Error())
		}

		return
	}

	if err != nil {
		return
	}

	id, err := strconv.Atoi(params[2])

	if err != nil {
		log.Printf("[UpdateRecord] POST '/%v/%v'. Bad converted id to int. Error: %v",
			table, id, err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	columnIDName, placeholder, item, err := CheckParamsAndTypes(h.Table[idx], r)

	if err != nil {
		log.Printf("Cant update %s", columnIDName)

		w.WriteHeader(http.StatusBadRequest)
		_, err2 := w.Write(
			[]byte(
				fmt.Sprintf(`{"error" : "field %s have invalid type"}`, columnIDName),
			),
		)

		if err2 != nil {
			log.Println("Bad request:", err2.Error())
		}
		return
	}

	if strings.Contains(placeholder, columnIDName) {

		log.Printf("Cant update %s", columnIDName)

		w.WriteHeader(http.StatusBadRequest)
		_, err2 := w.Write(
			[]byte(
				fmt.Sprintf(`{"error" : "field %s have invalid type"}`, columnIDName),
			),
		)

		if err2 != nil {
			log.Println("Bad request:", err2.Error())
		}
		return
	}

	query := fmt.Sprintf(
		"UPDATE %v SET %v WHERE %v = %d",
		h.Table[idx].Name, placeholder, columnIDName, id,
	)

	log.Println(query)

	res, err := h.DB.Exec(query, item...)

	if err != nil {
		log.Printf("[UpdateRecord] Bad Execute query! Error: %v",
			err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	affected, err := res.RowsAffected()

	if err != nil {
		log.Printf("[UpdateRecord] Bad called RowsAffected()! Error: %v",
			err.Error())
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

	if err != nil {
		log.Println("Bad packed json:", err.Error())
	}

	_, err = w.Write(result)

	if err != nil {
		log.Println("Bad request:", err.Error())
	}

	log.Println("Row affected:", affected)
}

// Хендлер для удлаения записи по ID. Вызывается по эндпоинту "/{table}/{id}". [DELETE]
func (h *Handler) DeleteRecord(w http.ResponseWriter, r *http.Request) {

	url := r.URL.Path

	params := strings.Split(url, "/")

	table := params[1]

	cond, idx, err := contains(h.Table, table)

	if !cond {

		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		_, err2 := w.Write(
			[]byte(`{"error" : "unknown table"}`),
		)

		if err2 != nil {
			log.Println("Bad packed json:", err2.Error())
		}

		return
	}

	if err != nil {
		return
	}

	id, err := strconv.Atoi(params[2])

	if err != nil {
		log.Printf("[DeleteRecord] DELETE '/%v/%v'. Bad converted id to int. Error: %v",
			table, id, err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	NameID := GetIDColumnName(h.Table[idx])

	query := fmt.Sprintf(
		"DELETE FROM %v WHERE %v = %d", h.Table[idx].Name, NameID, id,
	)

	log.Println(query)

	res, err := h.DB.Exec(query)

	if err != nil {
		log.Printf("[DeleteRecord] Bad Execute query! Error: %v", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	affected, err := res.RowsAffected()

	if err != nil {
		log.Printf("[DeleteRecord] Bad called RowsAffected()! \nError: %v", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result, err := json.Marshal(
		map[string]interface{}{
			"response": map[string]int{
				"deleted": int(affected),
			},
		},
	)

	if err != nil {
		log.Println("Bad packed json:", err.Error())
	}

	log.Println(result)

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(result)

	if err != nil {
		log.Println("Bad request:", err.Error())
	}
}

// Возвращаем имя столбца, который явл. ID
func GetIDColumnName(table TableInfo) string {

	for _, field := range table.Fields {

		if field.IsKey {
			return field.Name
		}
	}

	return ""
}

// Проверка наличия конкретной таблицы
func contains(s []TableInfo, table string) (bool, int, error) {
	for idx, v := range s {
		if v.Name == table {
			return true, idx, nil
		}
	}

	return false, -1, fmt.Errorf(
		fmt.Sprintf("The database not contain table %v", table),
	)
}

// Смотрим имена столбцов в таблице
func GetColumnsTable(table TableInfo, key bool) string {

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
func MakeContainerInsert(table TableInfo, r *http.Request) ([]interface{}, string, error) {

	item := make([]interface{}, 0)
	placeholder := make([]string, 0)

	decoder := json.NewDecoder(r.Body)
	param := make(map[string]interface{}, len(table.Fields))
	err := decoder.Decode(&param)

	if err != nil {
		log.Println("Bad decode json data: ", err.Error())
		return make([]interface{}, 0), "", err
	}

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
				case TypeInt:
					item = append(item, 0)
				case TypeVarchar, TypeText:
					item = append(item, "")
				}
			}
		}

		placeholder = append(placeholder, "?")
	}

	return item, strings.Join(placeholder, ","), nil
}

// Проверка типов параметров, которые пришли в реквесте. Делаем placeholders
func CheckParamsAndTypes(table TableInfo, r *http.Request) (string, string, []interface{}, error) {

	var columnIDName string

	item := make([]interface{}, 0)
	placeholder := make([]string, 0)

	decoder := json.NewDecoder(r.Body)
	param := make(map[string]interface{}, len(table.Fields))
	err := decoder.Decode(&param)

	if err != nil {
		log.Println("Bad decode json data")
		return "", "", make([]interface{}, 0), err
	}

	for _, field := range table.Fields {

		if field.IsKey {
			columnIDName = field.Name
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

					if field.ColumnType != TypeVarchar && field.ColumnType != TypeText {
						return field.Name, "", make([]interface{}, 0), fmt.Errorf("%s column cant use this type", field.Name)
					}

				case float64:
					if field.ColumnType != TypeInt {
						return field.Name, "", make([]interface{}, 0), fmt.Errorf("%s column cant use this type", field.Name)
					}
				}
			}
		}

		item = append(item, val)

		placeholder = append(placeholder, fmt.Sprintf("%v = ?", key))
	}

	return columnIDName, strings.Join(placeholder, ","), item, nil
}

// Возвращаем интерфейс с подготовленными типами для
// чтения записи из таблицы table
func ColumnsType(table TableInfo) []interface{} {

	values := make([]interface{}, len(table.Fields))

	for i, field := range table.Fields {
		switch field.ColumnType {
		case TypeInt:
			values[i] = new(sql.NullInt64)
		case TypeVarchar, TypeText:
			values[i] = new(sql.NullString)
		}
	}

	return values
}

// Делаем каст
func CastType(record []interface{}, table TableInfo) map[string]interface{} {

	item := make(map[string]interface{}, len(record))

	for idx, value := range record {

		if v, ok := value.(*sql.NullString); ok {
			if v.Valid {
				item[table.Fields[idx].Name] = v.String
			} else {
				item[table.Fields[idx].Name] = nil
			}
		}

		if v, ok := value.(*sql.NullInt64); ok {
			if v.Valid {
				item[table.Fields[idx].Name] = v.Int64
			} else {
				item[table.Fields[idx].Name] = nil
			}
		}

	}
	return item
}

// Информация о все таблицах в БД
func GetAllTables(db *sql.DB) ([]string, error) {

	tables := make([]string, 0)

	// make request to db. Get all tables name
	rows, err := db.Query(`SHOW TABLES;`)

	// auto close after returns
	defer rows.Close() //nolint:staticcheck

	if err != nil {
		return tables, err
	}

	// iteration over returned query from db and read data
	for rows.Next() {

		table := ""

		err = rows.Scan(&table)

		if err != nil {
			return make([]string, 0), err
		}

		tables = append(tables, table)
	}

	return tables, nil
}

// Получение информации о всех столбцах таблицы
func GetTablesInfo(db *sql.DB) ([]TableInfo, error) {

	tableInfo := []TableInfo{}

	tables, err := GetAllTables(db)

	if err != nil {
		return nil, err
	}

	for _, table := range tables {

		fieldInfo := []FieldInfo{}

		var nameID string

		rows, err := db.Query(
			fmt.Sprintf(`SHOW COLUMNS FROM %s`, table),
		)

		if err != nil {
			return nil, err
		}

		col := Columns{}

		for rows.Next() {

			var null bool
			var isKey bool

			err = rows.Scan(&col.Field, &col.Type, &col.Null, &col.Key, &col.Default, &col.Extra)

			if err != nil {
				return nil, err
			}

			if col.Key == "PRI" {
				isKey = true
				nameID = col.Field
			}

			if col.Null == "YES" {
				null = true
			}

			fieldInfo = append(
				fieldInfo,
				FieldInfo{
					Name:       col.Field,
					ColumnType: col.Type,
					IsKey:      isKey,
					CouldNull:  null,
				},
			)
		}

		tableInfo = append(
			tableInfo,
			TableInfo{
				Name:   table,
				ID:     nameID,
				Fields: fieldInfo,
			},
		)
	}

	log.Println(tableInfo)

	return tableInfo, nil
}

func URLLength(url string) int {

	count := 0

	if url == "/" {
		count = 0
	} else {

		split := strings.Split(url, "/")

		for _, item := range split {
			if item != "" {
				count++
			}
		}
	}

	return count
}

func (h *Handler) mainHandler(w http.ResponseWriter, r *http.Request) {

	url := r.URL.Path
	lenurl := URLLength(url)

	switch r.Method {

	case "GET":
		switch lenurl {

		case 0:
			h.TableList(w)
		case 1:
			h.SelectRecord(w, r)
		case 2:
			h.SelectRecordByID(w, r)
		default:
			w.WriteHeader(http.StatusBadGateway)
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"error" : "error way"}`))

			if err != nil {
				log.Println("Bad request:", err.Error())
			}
		}

	case "PUT":
		h.CreateRecord(w, r)
	case "POST":
		h.UpdateRecord(w, r)
	case "DELETE":
		h.DeleteRecord(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"error" : "error method"}`))

		if err != nil {
			log.Println("Bad request:", err.Error())
		}
	}
}

func NewDBExplorer(db *sql.DB) (http.Handler, error) {

	handler := &Handler{
		DB: db,
	}

	tableInfo, err := GetTablesInfo(db)

	if err != nil {
		return nil, err
	}

	handler.Table = tableInfo

	mux := http.NewServeMux()

	mux.HandleFunc("/", handler.mainHandler)

	return mux, nil
}
