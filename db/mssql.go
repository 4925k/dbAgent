package db

import (
	"database/sql"
	"encoding/json"
	"fmt"

	lg "../logstash"
	_ "github.com/denisenkom/go-mssqldb"
)

type Mssql struct{ C Config }

func (m Mssql) TestConn() error {
	connString := fmt.Sprintf("server=%s; port=%d; user id=%s; password=%s; database=%s;", m.C.ServerIP, m.C.ServerPort, m.C.DatabaseUser, m.C.DatabasePassword, m.C.Database)

	db, err := sql.Open(m.C.Driver, connString)
	if err != nil {
		return err
	}
	//check if connection has been established
	err = db.Ping()
	if err != nil {
		return err
	}
	db.Close()
	return nil
}

func (m Mssql) GetCount() (int, error) {
	connString := fmt.Sprintf("server=%s; port=%d; user id=%s; password=%s; database=%s;", m.C.ServerIP, m.C.ServerPort, m.C.DatabaseUser, m.C.DatabasePassword, m.C.Database)

	db, err := sql.Open(m.C.Driver, connString)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", m.C.Table)
	rows, err := db.Query(query)
	if err != nil {
		return -1, err
	}
	var count int
	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return 0, err
		}
	}

	db.Close()
	return count, nil
}

func (m Mssql) GetTable() error {
	connString := fmt.Sprintf("server=%s; port=%d; user id=%s; password=%s; database=%s;", m.C.ServerIP, m.C.ServerPort, m.C.DatabaseUser, m.C.DatabasePassword, m.C.Database)

	db, err := sql.Open(m.C.Driver, connString)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("SELECT * FROM %s", m.C.Table)
	rows, err := db.Query(query)
	if err != nil {
		return err
	}

	cols, _ := rows.Columns()
	for rows.Next() {
		// Create a slice of interface{}'s to represent each column,
		// and a second slice to contain pointers to each item in the columns slice.
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		// Scan the result into the column pointers...
		if err := rows.Scan(columnPointers...); err != nil {
			return err
		}

		// Create our map, and retrieve the value for each column from the pointers slice,
		// storing it in the map with the name of the column as the key.
		y := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			y[colName] = *val
		}

		// Outputs: map[columnName:value columnName2:value2 columnName3:value3 ...]
		x, _ := json.Marshal(y)

		//Send to Logstash
		lg.Send(m.C.LogstashIP, string(x), m.C.LogstashPort)
	}

	db.Close()
	return nil
}

func (m Mssql) GetTableOffset(newCount int) error {
	connString := fmt.Sprintf("server=%s; port=%d; user id=%s; password=%s; database=%s;", m.C.ServerIP, m.C.ServerPort, m.C.DatabaseUser, m.C.DatabasePassword, m.C.Database)

	db, err := sql.Open(m.C.Driver, connString)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("SELECT * FROM %s ORDER BY (SELECT NULL) OFFSET %d ROWS", m.C.Table, m.C.RowCount)
	rows, err := db.Query(query)
	if err != nil {
		return err
	}

	cols, _ := rows.Columns()
	for rows.Next() {
		// Create a slice of interface{}'s to represent each column,
		// and a second slice to contain pointers to each item in the columns slice.
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		// Scan the result into the column pointers...
		if err := rows.Scan(columnPointers...); err != nil {
			return err
		}

		// Create our map, and retrieve the value for each column from the pointers slice,
		// storing it in the map with the name of the column as the key.
		y := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			y[colName] = *val
		}

		// Outputs: map[columnName:value columnName2:value2 columnName3:value3 ...]
		x, _ := json.Marshal(y)

		//Send to Logstash
		lg.Send(m.C.LogstashIP, string(x), m.C.LogstashPort)
	}

	db.Close()
	return nil
}
