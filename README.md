# MySQL Libaray for golang

## How do you import in your golang project?
> go get github.com/mateors/msql

## Example code with sqlite3 database
```go

package main

import (
  "fmt"
  "log"
	"database/sql"
 	"github.com/mateors/msql"
	_ "github.com/mattn/go-sqlite3"
)
var db .*sql.DB
var err error

func init(){

	// Connect to database
	db, err = sql.Open("sqlite3", "./dbfile")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Println("db connection successful")

}

func main(){

	rows, err := msql.GetAllRowsByQuery("SELECT * FROM request", db)
	if err != nil {
		log.Fatal(err)
	}

	for i, row := range rows {
		fmt.Println(i, row)
	}
  
}
