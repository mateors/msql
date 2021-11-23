# MySQL Libaray for golang

## How do you import in your golang project?
> go get -u github.com/mateors/msql

## Import Sqlite3 thirdparty database driver according to your needs
> go get -u github.com/mattn/go-sqlite3

## or Mysql database driver
> go get -u github.com/go-sql-driver/mysql

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
	db.SetMaxOpenConns(1)
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
```

## Example code with mysql database
```go

package main

import (
  "fmt"
  "log"
  "database/sql"
  "github.com/mateors/msql"
_ "github.com/go-sql-driver/mysql"
)

var db .*sql.DB
var err error

func init(){

	// Connect to database
	db, err = sql.Open("mysql", "user:password@/dbname")
	if err != nil {
		panic(err)
	}
	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

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
```
