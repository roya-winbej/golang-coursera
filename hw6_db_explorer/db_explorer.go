package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

type DbExplorer struct {
	DB *sql.DB
}

func NewDbExplorer(db *sql.DB) (*DbExplorer, error) {
	return &DbExplorer{
		DB: db,
	}, nil
}

func getAllTables(db *sql.DB) ([]string, error) {
	var tables []string

	rows, err := db.Query("SHOW TABLES;")
	defer rows.Close()

	if err != nil {
		log.Printf("%v", err)
		return nil, err
	}

	for rows.Next() {
		var table string

		if err := rows.Scan(&table); err != nil {
			log.Printf("%v", err)
			return nil, err
		}

		tables = append(tables, table)
	}

	return tables, nil
}

func isTableExists(w http.ResponseWriter, table string, db *sql.DB) bool {
	tables, err := getAllTables(db)
	if err != nil {
		panic(err)
	}

	tableStr := strings.Join(tables, " ")
	if !strings.Contains(tableStr, table) {

		encoder := json.NewEncoder(w)

		w.WriteHeader(http.StatusNotFound)

		if err := encoder.Encode(struct {
			Error string `json:"error"`
		}{
			Error: "unknown table",
		}); err != nil {
			log.Printf("%v", err)
			http.Error(w, err.Error(), 500)
		}
		return false
	}

	return true
}

func (db *DbExplorer) getTables(w http.ResponseWriter, r *http.Request) {

	tables, err := getAllTables(db.DB)
	if err != nil {
		log.Printf("%v", err)
		http.Error(w, err.Error(), 500)
		return
	}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(struct {
		Data []string `json:"data"`
	}{
		Data: tables,
	}); err != nil {
		log.Printf("%v", err)
		http.Error(w, err.Error(), 500)
	}

}

func (db *DbExplorer) getTableItems(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("%v", err)
			http.Error(w, err.(error).Error(), 500)
		}
	}()

	vars := mux.Vars(r)
	tableName := vars["table"]

	query := r.URL.Query()

	queryLimit := query.Get("limit")
	queryOffset := query.Get("offset")

	var limit = 5
	var offset = 0

	if queryLimit != "" {
		limitNumber, err := strconv.Atoi(queryLimit)
		if err == nil {
			limit = limitNumber
		}

	}

	if queryOffset != "" {
		offsetNumber, err := strconv.Atoi(queryOffset)
		if err == nil {
			offset = offsetNumber
		}
	}

	if ok := isTableExists(w, tableName, db.DB); !ok {
		return
	}

	rows, err := db.DB.Query(fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?", tableName), limit, offset)
	defer rows.Close()
	if err != nil {
		panic(err)
	}

	var items []interface{}

	columns, _ := rows.Columns()

	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(values))

	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(scanArgs...); err != nil {
			panic(err)
		}

		item := make(map[string]string)

		for i, col := range values {
			if col != nil {
				item[columns[i]] = string(col)
			}
		}

		items = append(items, item)
	}

	if rows.Err() != nil {
		panic(err)
	}

	type Data struct {
		Records []interface{} `json:"records"`
	}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(struct {
		Response Data `json:"response"`
	}{
		Response: Data{items},
	}); err != nil {
		panic(err)
	}
}

func (db *DbExplorer) getTableItem(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("%v", err)
			http.Error(w, err.(error).Error(), 500)
		}
	}()

	vars := mux.Vars(r)
	tableName := vars["table"]
	itemParam := vars["id"]

	itemID, err := strconv.Atoi(itemParam)
	if err != nil {
		log.Printf("%v", err)
		http.Error(w, "id must be a number", http.StatusBadRequest)
		return
	}

	if ok := isTableExists(w, tableName, db.DB); !ok {
		return
	}

	rows, err := db.DB.Query(fmt.Sprintf("SELECT * FROM %s WHERE id = ?", tableName), itemID)
	defer rows.Close()
	if err != nil {
		panic(err)
	}

	var items []interface{}

	columns, _ := rows.Columns()

	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(values))

	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(scanArgs...); err != nil {
			panic(err)
		}

		item := make(map[string]string)

		for i, col := range values {
			if col != nil {
				item[columns[i]] = string(col)
			}
		}

		items = append(items, item)
	}

	if rows.Err() != nil {
		panic(err)
	}

	type Data struct {
		Record interface{} `json:"record"`
	}

	encoder := json.NewEncoder(w)

	if items == nil {
		w.WriteHeader(http.StatusNotFound)

		if err := encoder.Encode(struct {
			Error string `json:"error"`
		}{
			Error: "record not found",
		}); err != nil {
			panic(err)
		}

		return
	}

	if err := encoder.Encode(struct {
		Response Data `json:"response"`
	}{
		Response: Data{items[0]},
	}); err != nil {
		panic(err)
	}
}

func (db *DbExplorer) createTable(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("%v", err)
			http.Error(w, err.(error).Error(), 500)
		}
	}()

	vars := mux.Vars(r)
	tableName := vars["table"]

	var requestBody interface{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestBody); err != nil {
		panic(err)
	}

	if ok := isTableExists(w, tableName, db.DB); !ok {
		return
	}

	type columnInfo struct {
		Field     string
		Type      string
		Null      string
		isPrimary string
	}

	var colsInfo []*columnInfo

	rows, err := db.DB.Query(fmt.Sprintf("select distinct(COLUMN_NAME), IS_NULLABLE, DATA_TYPE, COLUMN_KEY from information_schema.COLUMNS where TABLE_NAME='%s';", tableName))
	defer rows.Close()
	if err != nil {
		panic(err)
	}

	for rows.Next() {
		column := &columnInfo{}
		if err := rows.Scan(&column.Field, &column.Null, &column.Type, &column.isPrimary); err != nil {
			panic(err)
		}

		colsInfo = append(colsInfo, column)
	}

	type query struct {
		keys   []string
		values []interface{}
	}

	q := &query{}

	for _, colInfo := range colsInfo {

		for key, value := range requestBody.(map[string]interface{}) {

			if key == colInfo.Field {

				if colInfo.isPrimary != "" {
					continue
				}

				q.keys = append(q.keys, key)
				q.values = append(q.values, value)
			}
		}
	}

	placeholders := strings.Repeat("?,", len(q.keys))

	queryString := `INSERT INTO ` + tableName + ` (` + strings.Join(q.keys, ", ") + `) VALUES (` + placeholders[:len(placeholders)-1] + `)`

	result, err := db.DB.Exec(
		queryString,
		q.values...,
	)
	if err != nil {
		panic(err)
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		panic(err)
	}

	encoder := json.NewEncoder(w)

	type response struct {
		ID int64 `json:"id"`
	}

	if err := encoder.Encode(struct {
		Response response `json:"response"`
	}{
		Response: response{
			ID: lastID,
		},
	}); err != nil {
		panic(err)
	}
}

func (db *DbExplorer) updateTableItem(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("%v", err)
			http.Error(w, err.(error).Error(), 500)
		}
	}()

	vars := mux.Vars(r)
	itemParam := vars["id"]
	tableName := vars["table"]

	_, err := strconv.Atoi(itemParam)
	if err != nil {
		log.Printf("%v", err)
		http.Error(w, "id must be a number", http.StatusBadRequest)
		return
	}

	var requestBody interface{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&requestBody); err != nil {
		panic(err)
	}

	if ok := isTableExists(w, tableName, db.DB); !ok {
		return
	}

	type columnInfo struct {
		Field     string
		Type      string
		Null      string
		isPrimary string
	}

	var colsInfo []*columnInfo

	rows, err := db.DB.Query(fmt.Sprintf("select distinct(COLUMN_NAME), IS_NULLABLE, DATA_TYPE, COLUMN_KEY from information_schema.COLUMNS where TABLE_NAME='%s';", tableName))
	defer rows.Close()
	if err != nil {
		panic(err)
	}

	for rows.Next() {
		column := &columnInfo{}
		if err := rows.Scan(&column.Field, &column.Null, &column.Type, &column.isPrimary); err != nil {
			panic(err)
		}

		switch column.Type {
		case "varchar":
			column.Type = "string"
		case "text":
			column.Type = "string"
		}

		colsInfo = append(colsInfo, column)
	}

	type query struct {
		keys   []string
		values []interface{}
	}

	q := &query{}

	for _, colInfo := range colsInfo {

		for key, value := range requestBody.(map[string]interface{}) {

			if key == colInfo.Field {

				if colInfo.isPrimary != "" {

					if len(requestBody.(map[string]interface{})) == 1 {
						http.Error(w, `{"error": "field id have invalid type"}`, http.StatusBadRequest)
						return
					}

					continue
				}

				if reflect.TypeOf(value) == nil && colInfo.Null == "NO" {
					http.Error(w, `{"error": "field `+colInfo.Field+` have invalid type"}`, http.StatusBadRequest)
					return
				}

				if reflect.TypeOf(value) != nil && (reflect.TypeOf(value).Name() != colInfo.Type) {
					http.Error(w, `{"error": "field `+colInfo.Field+` have invalid type"}`, http.StatusBadRequest)
					return
				}

				q.keys = append(q.keys, key)
				q.values = append(q.values, value)
			}
		}
	}

	var querySetParams string

	for idx, key := range q.keys {
		if idx < len(q.keys)-1 {
			querySetParams += key + " = ?, "
		} else {
			querySetParams += key + " = ? "
		}
	}

	queryString := `UPDATE ` + tableName + ` SET ` + querySetParams + `WHERE id = ` + itemParam

	result, err := db.DB.Exec(
		queryString,
		q.values...,
	)
	if err != nil {
		panic(err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		panic(err)
	}

	encoder := json.NewEncoder(w)

	type response struct {
		Updated int64 `json:"updated"`
	}

	if err := encoder.Encode(struct {
		Response response `json:"response"`
	}{
		Response: response{
			Updated: affected,
		},
	}); err != nil {
		panic(err)
	}
}

func (db *DbExplorer) deleteTableItem(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("%v", err)
			http.Error(w, err.(error).Error(), 500)
		}
	}()

	vars := mux.Vars(r)
	tableName := vars["table"]
	itemParam := vars["id"]

	itemID, err := strconv.Atoi(itemParam)
	if err != nil {
		log.Printf("%v", err)
		http.Error(w, "id must be a number", http.StatusBadRequest)
		return
	}

	if ok := isTableExists(w, tableName, db.DB); !ok {
		return
	}
	res, err := db.DB.Exec(fmt.Sprintf("DELETE FROM %s WHERE id=? LIMIT 1;", tableName), itemID)
	if err != nil {
		panic(err)
	}
	rowCnt, err := res.RowsAffected()
	if err != nil {
		panic(err)
	}

	type Data struct {
		Msg int64 `json:"deleted"`
	}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(struct {
		Response Data `json:"response"`
	}{
		Response: Data{Msg: rowCnt},
	}); err != nil {
		panic(err)
	}
}
