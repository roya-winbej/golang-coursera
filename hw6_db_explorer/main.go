// тут лежит тестовый код
// менять вам может потребоваться только коннект к базе
package main

import (
	"database/sql"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

var (
	// DSN это соединение с базой
	// вы можете изменить этот на тот который вам нужен
	// docker run -p 3306:3306 -v $(PWD):/docker-entrypoint-initdb.d -e MYSQL_ROOT_PASSWORD=1234 -e MYSQL_DATABASE=golang -d mysql
	DSN = "root:root@tcp(localhost:3306)/golang_coursera?charset=utf8"
	// DSN = "coursera:5QPbAUufx7@tcp(localhost:3306)/coursera?charset=utf8"
)

func main() {
	db, err := sql.Open("mysql", DSN)
	db.SetMaxOpenConns(10)
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	defer db.Close()

	r := mux.NewRouter()

	dbExplorer, err := NewDbExplorer(db)
	if err != nil {
		panic(err)
	}

	r.Use(JSONHeaders)

	r.HandleFunc("/", dbExplorer.getTables).Methods("GET")
	r.HandleFunc("/{table}", dbExplorer.getTableItems).Methods("GET")
	r.HandleFunc("/{table}/{id}", dbExplorer.getTableItem).Methods("GET")
	r.HandleFunc("/{table}", dbExplorer.createTable).Methods("POST")
	r.HandleFunc("/{table}/{id}", dbExplorer.updateTableItem).Methods("PUT")
	r.HandleFunc("/{table}/{id}", dbExplorer.deleteTableItem).Methods("DELETE")

	fmt.Println("starting server at :8082")
	fmt.Println(http.ListenAndServe(":8082", r))
}
