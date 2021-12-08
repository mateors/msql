package msql

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"
)

type stringStringScan struct {
	// cp are the column pointers
	cp []interface{}
	// row contains the final result
	row      []string
	colCount int
	colNames []string
}
type mapStringScan struct {
	// cp are the column pointers
	cp []interface{}
	// row contains the final result
	row      map[string]string
	colCount int
	colNames []string
}

func newMapStringScan(columnNames []string) *mapStringScan {
	lenCN := len(columnNames)
	s := &mapStringScan{
		cp:       make([]interface{}, lenCN),
		row:      make(map[string]string, lenCN),
		colCount: lenCN,
		colNames: columnNames,
	}
	for i := 0; i < lenCN; i++ {
		s.cp[i] = new(sql.RawBytes)
	}
	return s
}

func (s *mapStringScan) Update(rows *sql.Rows) error {
	if err := rows.Scan(s.cp...); err != nil {
		return err
	}

	for i := 0; i < s.colCount; i++ {
		if rb, ok := s.cp[i].(*sql.RawBytes); ok {
			s.row[s.colNames[i]] = string(*rb)
			*rb = nil // reset pointer to discard current value to avoid a bug
		} else {
			return fmt.Errorf("Cannot convert index %d column %s to type *sql.RawBytes", i, s.colNames[i])
		}
	}
	return nil
}

func (s *mapStringScan) Get() map[string]string {
	return s.row
}

func newStringStringScan(columnNames []string) *stringStringScan {
	lenCN := len(columnNames)
	s := &stringStringScan{
		cp:       make([]interface{}, lenCN),
		row:      make([]string, lenCN*2),
		colCount: lenCN,
		colNames: columnNames,
	}
	j := 0
	for i := 0; i < lenCN; i++ {
		s.cp[i] = new(sql.RawBytes)
		s.row[j] = s.colNames[i]
		j = j + 2
	}
	return s
}

func (s *stringStringScan) Update(rows *sql.Rows) error {
	if err := rows.Scan(s.cp...); err != nil {
		return err
	}
	j := 0
	for i := 0; i < s.colCount; i++ {
		if rb, ok := s.cp[i].(*sql.RawBytes); ok {
			s.row[j+1] = string(*rb)
			*rb = nil // reset pointer to discard current value to avoid a bug
		} else {
			return fmt.Errorf("Cannot convert index %d column %s to type *sql.RawBytes", i, s.colNames[i])
		}
		j = j + 2
	}
	return nil
}

func (s *stringStringScan) Get() []string {
	return s.row
}

// rowMapString was the first implementation but it creates for each row a new
// map and pointers and is considered as slow. see benchmark
func rowMapString(columnNames []string, rows *sql.Rows) (map[string]string, error) {
	lenCN := len(columnNames)
	ret := make(map[string]string, lenCN)

	columnPointers := make([]interface{}, lenCN)
	for i := 0; i < lenCN; i++ {
		columnPointers[i] = new(sql.RawBytes)
	}

	if err := rows.Scan(columnPointers...); err != nil {
		return nil, err
	}

	for i := 0; i < lenCN; i++ {
		if rb, ok := columnPointers[i].(*sql.RawBytes); ok {
			ret[columnNames[i]] = string(*rb)
		} else {
			return nil, fmt.Errorf("Cannot convert index %d column %s to type *sql.RawBytes", i, columnNames[i])
		}
	}

	return ret, nil
}

//InsertIntoAnyTable Insert into any table using formData and *sql.DB
func InsertIntoAnyTable(tableInfo url.Values, db *sql.DB) (primarykeyValue int64, err error) {

	table := tableInfo.Get("table")
	dbtype := tableInfo.Get("dbtype")

	var dbColList []string
	if dbtype == "sqlite3" {
		dbColList, err = ReadTable2ColumnSqlit3(table, db)
	} else {
		dbColList, err = ReadTable2Columns(table, db)
	}

	if err != nil {
		//fmt.Println("dbColList: ", dbColList, "ERROR:>>>>", err)
		return 0, err
	}

	keyList, valList := Form2KeyValueSlice(tableInfo, dbColList)
	//fmt.Println("keyList: ", keyList)
	//fmt.Println("valList: ", valList)

	sql := InsertQueryBuilder(keyList, table)
	//fmt.Println("sql: ", sql)
	//fmt.Println("valList: ", valList)
	primarykeyValue, _, err = Finsert(sql, valList, db)

	return
}

func ReadTable2ColumnSqlit3Trx(table string, trx *sql.Tx) ([]string, error) {

	sql := fmt.Sprintf("PRAGMA table_info(%s);", table)
	rows, err := trx.Query(sql)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var dflt_value *string
	var cid, name, vtype, notnull, pk string

	//cols := make(map[string]string)
	cols := []string{}

	for rows.Next() {
		err = rows.Scan(&cid, &name, &vtype, &notnull, &dflt_value, &pk)
		if err != nil {
			fmt.Println("ReadTable2Columnsqlite3:", err.Error())
		}
		cols = append(cols, name)
	}
	return cols, nil
}

//ReadTable2Columns Get table all columns as a slice of string
func ReadTable2ColumnSqlit3(table string, db *sql.DB) ([]string, error) {

	sql := fmt.Sprintf("PRAGMA table_info(%s);", table)
	rows, err := db.Query(sql)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var dflt_value *string
	var cid, name, vtype, notnull, pk string

	//cols := make(map[string]string)
	cols := []string{}

	for rows.Next() {
		err = rows.Scan(&cid, &name, &vtype, &notnull, &dflt_value, &pk)
		if err != nil {
			fmt.Println("ReadTable2Columnsqlite3:", err.Error())
		}
		cols = append(cols, name)
	}
	return cols, nil
}

//ReadTable2Columns Get table all columns as a slice of string
func ReadTable2Columns(table string, db *sql.DB) ([]string, error) {

	sql := fmt.Sprintf("SHOW COLUMNS FROM %v;", table)
	//sql := fmt.Sprintf("PRAGMA table_info(%s);", table)

	rows, err := db.Query(sql)

	//check(err, "read.query")
	if err != nil {
		// mysqlerr, _ := err.(*mysql.MySQLError)
		// if mysqlerr.Number == 1146 {
		// 	return nil, mysqlerr
		// }
		//Error 1146: Table 'gregfins_system.modules' doesn't exist
		return nil, err
	}

	defer rows.Close()

	//sql.NullString
	var vfield, vtype, vnull, vkey, vextra string
	var vdefault *string

	//cols := make(map[string]string)
	cols := []string{}

	for rows.Next() {
		err = rows.Scan(&vfield, &vtype, &vnull, &vkey, &vdefault, &vextra)
		if err != nil {
			fmt.Println("ReadTable2Columns:", err.Error())
		}
		//data := fmt.Sprintf("%v,%v,%v,%v,%v,%v = %T", vfield, vtype, vnull, vkey, vdefault, vextra, vdefault)
		//fmt.Println(data)

		//if vkey != "PRI" { //primary key are generally auto incremented value so we ignore
		cols = append(cols, vfield)
		//}

	}

	return cols, nil

}

//Finsert Insert using sql query, return LastInsertId,RowsAffected, Error
func FinsertTrx(sql string, valAray []string, trx *sql.Tx) (int64, int64, error) {

	stmt, err := trx.Prepare(sql)
	if err != nil {
		return 0, 0, err
	}

	defer stmt.Close()
	v := make([]interface{}, len(valAray))
	for i, val := range valAray {
		v[i] = val
	}

	res, err := stmt.Exec(v...) //"Inventory", "1", "1"
	if err != nil {
		return 0, 0, err
	}

	err = stmt.Close()
	if err != nil {
		return 0, 0, err
	}

	lrid, _ := res.LastInsertId()
	lcount, _ := res.RowsAffected()
	return lrid, lcount, nil
}

//Finsert Insert using sql query, return LastInsertId,RowsAffected, Error
func Finsert(sql string, valAray []string, db *sql.DB) (int64, int64, error) {

	stmt, err := db.Prepare(sql)
	if err != nil {

		return 0, 0, err
	}
	//check(err, "db.Prepare")
	defer stmt.Close()

	//mr := []string{"Account", "1", "1"}

	//fmt.Fprintf("%v\n", mr)

	//v := []interface{}{"Inventory", "1", "1"}
	// var names []interface{}
	// names = append(names, "Report")
	// names = append(names, "1")

	//vals := []string{"Report", "1", "1"}
	v := make([]interface{}, len(valAray))
	for i, val := range valAray {
		v[i] = val
	}

	res, err := stmt.Exec(v...) //"Inventory", "1", "1"
	if err != nil {

		//log.Fatal(err)
		//panic(err)
		//mysqlerr, _ := err.(*mysql.MySQLError)
		//fmt.Fprintln(w, "EROR::", mysqlerr.Number, mysqlerr.Message) //Error 1062: Duplicate entry 'Report' for key 'module_name'
		return 0, 0, err
	}

	lrid, _ := res.LastInsertId()
	lcount, _ := res.RowsAffected()

	return lrid, lcount, nil
}

//UpdateByValAray ...
func UpdateByValAray(sql string, valAray []string, db *sql.DB) (rowsAfftected int64, err error) {

	stmt, err := db.Prepare(sql)
	if err != nil {
		return 0, err
	}

	defer stmt.Close()
	//mr := []string{"Account", "1", "1"}
	//fmt.Fprintf("%v\n", mr)
	//v := []interface{}{"Inventory", "1", "1"}
	// var names []interface{}
	// names = append(names, "Report")
	// names = append(names, "1")
	//vals := []string{"Report", "1", "1"}

	v := make([]interface{}, len(valAray))
	for i, val := range valAray {
		v[i] = val
	}

	res, err := stmt.Exec(v...) //"Inventory", "1", "1"
	if err != nil {
		return 0, err
	}

	//lrid, _ := res.LastInsertId()
	rowsAfftected, _ = res.RowsAffected()

	return

}

//Form2KeyValueSlice Set form value and Get keyList, valueList separately
func Form2KeyValueSlice(form map[string][]string, colList []string) (keyList []string, valList []string) {

	fmap := make(map[string]string)
	for key, valAray := range form {
		val := valAray[0]
		fmap[key] = val
	}

	for _, colName := range colList {

		//fmt.Printf("%v \n", colName)
		var cval = ""
		if colval, ok := fmap[colName]; ok {
			//fmt.Printf("%v-> %v exist value = %v\n", i, colName, colval)
			cval = colval
		} else {
			//fmt.Printf("%v-> %v NOT IN MAP => %v\n", i, colName, colval)
		}

		if cval != "" {
			keyList = append(keyList, colName)
			valList = append(valList, cval)
		}

	}
	return
}

//InsertQueryBuilder Get raw sql query using key value pair and table name
func InsertQueryBuilder(keyVal []string, tableName string) string {

	sb := &strings.Builder{}
	//fmt.Fprintf(sb, "INSERT INTO `%v` (", tableName)
	fields := ""
	vals := ""

	//ignoring slice 0 index value which is primary key auto incremented
	for _, v := range keyVal {

		if v == "NULL" {
			//fields += fmt.Sprintf("%v, ", v)
			fields += "NULL, "
		} else {
			fields += fmt.Sprintf("`%v`, ", v)
		}

		vals += "?, "
	}

	fmt.Fprintf(sb, "INSERT INTO `%v` (%v) VALUES(%v);", tableName, strings.TrimRight(fields, ", "), strings.TrimRight(vals, ", "))
	return sb.String()

}

//UpdateQueryBuilder ...
func UpdateQueryBuilder(keyVal []string, tableName string, whereCondition string) (sql string) {

	sb := &strings.Builder{}
	var fields string

	for _, v := range keyVal {

		fields += fmt.Sprintf("`%v`=?, ", v)
	}

	fmt.Fprintf(sb, "UPDATE `%v` SET %v WHERE %v;", tableName, strings.TrimRight(fields, ", "), whereCondition)

	sql = sb.String()
	return
}

//FieldByValue Get one field_value using where clause
func FieldByValue(table, fieldName, where string, db *sql.DB) string {

	sql := fmt.Sprintf("SELECT %v FROM `%v` WHERE %v;", fieldName, table, where)
	rows := db.QueryRow(sql)

	var vfield string
	err := rows.Scan(&vfield)
	if err != nil {
		return vfield
	}

	//data := fmt.Sprintf("%v = %T", vcnt, vcnt)
	//fmt.Println(data)
	return vfield
}

//RawSQL Update using raq sql query,if query executed return true, otherwise false
func RawSQL(sql string, db *sql.DB) bool {

	stmt, err := db.Prepare(sql)
	if err != nil {
		fmt.Printf("Invalid Query @prepare: %v", err.Error())
		return false
	}

	defer stmt.Close()

	r, err := stmt.Exec()
	if err != nil {
		fmt.Printf("Fail execution @RawSql %v\n", err.Error())
		return false
	}

	n, err := r.RowsAffected()
	if err != nil {
		return false
	}

	if n > 0 {
		return true
	}

	return false

}

//CheckCount Get row count using where condition
func CheckCount(table, where string, db *sql.DB) (count int64) {

	sql := fmt.Sprintf("SELECT count(*)as cnt FROM %v WHERE %v;", table, where)
	rows := db.QueryRow(sql)

	//var count int64
	err := rows.Scan(&count)
	if err != nil {
		fmt.Println("CheckCount:", err.Error())
	}

	return
}

//GetAllRowsByQuery Get all table rows using raw sql query
func GetAllRowsByQuery(sql string, db *sql.DB) ([]map[string]interface{}, error) {

	rows, err := db.Query(sql)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	columnNames, err := rows.Columns()

	rc := newMapStringScan(columnNames)
	tableData := make([]map[string]interface{}, 0)
	//fmt.Println(rc.row)

	//var i = 0
	for rows.Next() {

		err := rc.Update(rows)
		//check(err, "rc.Update")
		if err != nil {
			break
		}

		cv := rc.Get()
		dd := make(map[string]interface{})
		for _, col := range columnNames {

			//fmt.Println(col)
			dd[col] = cv[col]

		}

		//fmt.Println(dd)
		tableData = append(tableData, dd)
		//i++

	}

	//fmt.Println(len(tableData))
	//fmt.Println(tableData)

	return tableData, nil

}
