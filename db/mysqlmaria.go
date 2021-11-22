package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	lg "../logstash"

	_ "github.com/go-sql-driver/mysql"
)

type Mysql struct{ C Config }

func (m Mysql) TestConn() error {
	connString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", m.C.DatabaseUser, m.C.DatabasePassword, m.C.ServerIP, m.C.ServerPort, m.C.Database)

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

func (m Mysql) GetCount() (int, error) {
	connString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", m.C.DatabaseUser, m.C.DatabasePassword, m.C.ServerIP, m.C.ServerPort, m.C.Database)

	db, err := sql.Open(m.C.Driver, connString)
	if err != nil {
		return 0, err
	}
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", m.C.Table)
	results, err := db.Query(query)
	if err != nil {
		return 0, err
	}
	var count int
	for results.Next() {
		err = results.Scan(&count)
		if err != nil {
			return 0, err
		}
	}

	db.Close()
	return count, nil
}

func (m Mysql) GetTable() error {
	connString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", m.C.DatabaseUser, m.C.DatabasePassword, m.C.ServerIP, m.C.ServerPort, m.C.Database)

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
		mm := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			mm[colName] = *val
		}

		//the results art in ascii so we convert them to string
		for i, v := range mm {
			if v != nil {
				str := fmt.Sprintf("%v", v)
				str = strings.ReplaceAll(str, " ", ",")

				var ints []byte
				err := json.Unmarshal([]byte(str), &ints)
				if err != nil {
					return err
				}
				mm[i] = string(ints)
			}

		}

		// Outputs: map[columnName:value columnName2:value2 columnName3:value3 ...]
		x, _ := json.Marshal(mm)

		//Send to Logstash
		lg.Send(m.C.LogstashIP, string(x), m.C.LogstashPort)
	}

	db.Close()
	return nil
}

func (m Mysql) GetTableOffset(newCount int) error {
	connString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", m.C.DatabaseUser, m.C.DatabasePassword, m.C.ServerIP, m.C.ServerPort, m.C.Database)

	db, err := sql.Open(m.C.Driver, connString)
	if err != nil {
		return err
	}

	row := newCount - m.C.RowCount
	query := fmt.Sprintf("SELECT * FROM ( SELECT * FROM %s ORDER BY %s DESC LIMIT %d ) sub ORDER BY %s ASC;", m.C.Table, m.C.Key, row, m.C.Key)
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
		mm := make(map[string]interface{})
		for i, colName := range cols {
			val := columnPointers[i].(*interface{})
			mm[colName] = *val
		}

		//the results art in ascii so we convert them to string
		for i, v := range mm {
			if v != nil {
				str := fmt.Sprintf("%v", v)
				str = strings.ReplaceAll(str, " ", ",")

				var ints []byte
				err := json.Unmarshal([]byte(str), &ints)
				if err != nil {
					return err
				}
				mm[i] = string(ints)
			}

		}

		// Outputs: map[columnName:value columnName2:value2 columnName3:value3 ...]
		x, _ := json.Marshal(mm)

		//Send to Logstash
		lg.Send(m.C.LogstashIP, string(x), m.C.LogstashPort)
	}

	db.Close()
	return nil
}
