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

## MySQL Transaction
```go
func CreateOrder(ctx context.Context, albumID, quantity, custID int) (orderID int64, err error) {

    // Create a helper function for preparing failure results.
    fail := func(err error) (int64, error) {
        return fmt.Errorf("CreateOrder: %v", err)
    }

    // Get a Tx for making transaction requests.
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return fail(err)
    }
    // Defer a rollback in case anything fails.
    defer tx.Rollback()

    // Confirm that album inventory is enough for the order.
    var enough bool
    if err = tx.QueryRowContext(ctx, "SELECT (quantity >= ?) from album where id = ?", quantity, albumID).Scan(&enough); err != nil {
        if err == sql.ErrNoRows {
            return fail(fmt.Errorf("no such album"))
        }
        return fail(err)
    }
    if !enough {
        return fail(fmt.Errorf("not enough inventory"))
    }

    // Update the album inventory to remove the quantity in the order.
    _, err = tx.ExecContext(ctx, "UPDATE album SET quantity = quantity - ? WHERE id = ?",
        quantity, albumID)
    if err != nil {
        return fail(err)
    }

    // Create a new row in the album_order table.
    result, err := tx.ExecContext(ctx, "INSERT INTO album_order (album_id, cust_id, quantity, date) VALUES (?, ?, ?, ?)",
        albumID, custID, quantity, time.Now())
    if err != nil {
        return fail(err)
    }
    // Get the ID of the order item just created.
    orderID, err := result.LastInsertId()
    if err != nil {
        return fail(err)
    }

    // Commit the transaction.
    if err = tx.Commit(); err != nil {
        return fail(err)
    }

    // Return the order ID.
    return orderID, nil
}
```
