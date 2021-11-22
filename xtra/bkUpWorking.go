package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v2"
)

type configList struct {
	Config []config `yaml:"config"`
}

type config struct {
	Name             string        `yaml:"name"`
	Driver           string        `yaml:"driver"`
	ServerIP         string        `yaml:"serverIP"`
	ServerPort       int           `yaml:"serverPort"`
	Database         string        `yaml:"database"`
	Table            string        `yaml:"table"`
	Key              string        `yaml:"key"`
	DatabaseUser     string        `yaml:"databaseUser"`
	DatabasePassword string        `yaml:"databasePassword"`
	LogstashIP       string        `yaml:"logstashIP"`
	LogstashPort     int           `yaml:"logstashPort"`
	RowCount         int           `yaml:"rowCount"`
	FirstTime        bool          `yaml:"firstTime"`
	TimeToSleep      time.Duration `yaml:"tts"`
}

type routine interface {
	getConn() *sql.DB
	getCount(*sql.DB) (int, error)
	getTableAll(db *sql.DB)
	getTableOffset(db *sql.DB, count int)
}

type mssql struct {
	c config
}

type mysql struct {
	c config
}

type postgresql struct {
	c config
}

func main() {
	var c configList                       //config list struct
	var err error                          //error
	var db *sql.DB                         //databse connection
	var exit = make(chan bool)             //channel to wait for go routine
	configLocation := "config/config.yaml" //path to config file
	logLocation := "log/mssqlAgent.log"    //path to log file

	//set up logging
	logFile, err := os.OpenFile(logLocation, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("ERROR opening logging file: ", err)
	}
	log.SetOutput(logFile)
	log.Print("INFO logging started")
	defer logFile.Close()

	//fetching configuration
	c.getConf(configLocation)

	for i, config := range c.Config {

		//mongoDB
		if config.Driver == "mongodb" {
			go func(i int, c configList) {
				for {
					err := mongoCheckConn(c.Config[i])
					if err != nil {
						log.Printf("ERROR %s : database connection : %s", c.Config[i].Name, err)
						continue
					}
					log.Printf("INFO %s : connection established", c.Config[i].Name)
					break
				}
				for {
					if c.Config[i].FirstTime {
						//get all table
						mongoGetTable(c.Config[i])
						log.Printf("INFO %s : all data fetched", c.Config[i].Name)
						//getCount
						count, err := mongoGetCount(c.Config[i])
						if err != nil {
							log.Printf("ERROR %s : fetching count : %s", c.Config[i].Name, err)
						}
						//update firstTime boolean and rowCount
						c.Config[i].FirstTime = false
						c.Config[i].RowCount = count
						updateConfig(c, configLocation)
						log.Printf("INFO %s : config updated", c.Config[i].Name)
					} else {
						//getCount
						count, err := mongoGetCount(c.Config[i])
						if err != nil {
							log.Printf("ERROR %s : fetching count : %s", c.Config[i].Name, err)
						}

						if count > c.Config[i].RowCount {
							log.Printf("INFO %s : new data found", c.Config[i].Name)
							mongoGetTableNew(c.Config[i], c.Config[i].RowCount)
							log.Printf("INFO %s : new data sent", c.Config[i].Name)
						} else {
							log.Printf("INFO %s : data up to date", c.Config[i].Name)
						}

						c.Config[i].RowCount = count
						updateConfig(c, configLocation)
						log.Printf("INFO %s : config updated", c.Config[i].Name)
					}

					log.Printf("INFO %s : sleeping", c.Config[i].Name)
					time.Sleep(c.Config[i].TimeToSleep * time.Minute)
				}

			}(i, c)
			continue
		}

		//to convert into needed structure
		conf := structConv(config)
		go func(i int, conf routine) {
			for {
				//get connection
				log.Printf("INFO %s : initializing connection", c.Config[i].Name)
				db = conf.getConn()
				log.Printf("INFO %s : connection established", c.Config[i].Name)

				//firstRun
				if c.Config[i].FirstTime {
					log.Printf("INFO %s : first time running", c.Config[i].Name)
					//get all table data
					conf.getTableAll(db)
					log.Printf("INFO %s : all data fetched", c.Config[i].Name)

					//fetch row count
					count, err := conf.getCount(db)
					if err != nil {
						log.Printf("ERROR %s : %s", c.Config[i].Name, err)
					}

					//update firstTime boolean and rowCount
					c.Config[i].FirstTime = false
					c.Config[i].RowCount = count
					updateConfig(c, configLocation)
					log.Printf("INFO %s : config updated", c.Config[i].Name)

				} else { //checking counts and getting new data
					count, err := conf.getCount(db)
					if err != nil {
						log.Printf("ERROR %s : %s", c.Config[i].Name, err)
					}

					if count > c.Config[i].RowCount {
						log.Printf("INFO %s : new data found", c.Config[i].Name)
						conf.getTableOffset(db, count)
						log.Printf("INFO %s : new data sent", c.Config[i].Name)
					} else {
						log.Printf("INFO %s : data up to date", c.Config[i].Name)
					}

					c.Config[i].RowCount = count
					updateConfig(c, configLocation)
					log.Printf("INFO %s : config updated", c.Config[i].Name)
				}

				db.Close()
				log.Printf("INFO %s : connection closed", c.Config[i].Name)
				log.Printf("INFO %s : sleeping", c.Config[i].Name)
				time.Sleep(c.Config[i].TimeToSleep * time.Minute)
			}

		}(i, conf)

	}
	//empty channel to let the go routine running
	<-exit
}

//getConf fetches config from the given location into a struct
func (c *configList) getConf(location string) *configList {

	yamlFile, err := ioutil.ReadFile(location)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	return c
}

//updateConfig marshals data into yaml and updates the file in given location
func updateConfig(data configList, location string) error {
	newConfig, _ := yaml.Marshal(data)

	err := ioutil.WriteFile(location, newConfig, 0644)
	if err != nil {
		return err
	}
	return nil
}

//structConv converts the given basic config into a databse specific struct
func structConv(conf config) routine {
	if conf.Driver == "mssql" {
		return mssql{conf}
	} else if conf.Driver == "mysql" {
		return mysql{conf}
	} else if conf.Driver == "postgres" {
		return postgresql{conf}
	}
	return nil
}

//sends to logstash ip
func logstash(ip, data string, port int) {
	address := fmt.Sprintf("%v:%v", ip, port)
	conn, err := net.Dial("udp", address)
	if err != nil {
		log.Printf("sending to logstash error %v", err)
		return
	}
	fmt.Fprintf(conn, data)
	conn.Close()
}

//MSSQL FUNCTIONS
//getConn returns a conection to the database
func (m mssql) getConn() *sql.DB {
	for {
		tts := time.Minute * 5
		connString := fmt.Sprintf("server=%s; port=%d; user id=%s; password=%s; database=%s;", m.c.ServerIP, m.c.ServerPort, m.c.DatabaseUser, m.c.DatabasePassword, m.c.Database)

		db, err := sql.Open(m.c.Driver, connString)
		if err != nil {
			log.Printf("ERROR %s : database connection : %s", m.c.Name, err)
			time.Sleep(tts)
			continue
		}
		//check if connection has been established
		err = db.Ping()
		if err != nil {
			log.Printf("ERROR %s : database connection : %s", m.c.Name, err)
			time.Sleep(tts)
			continue
		}
		return db

	}
}

//getCount fetches the row count
func (m mssql) getCount(db *sql.DB) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", m.c.Table)
	rows, err := db.Query(query)
	if err != nil {
		return -1, err
	}
	var count int
	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return -1, err
		}
	}
	return count, nil
}

//getTableAll sends all the data from the table to logstash
func (m mssql) getTableAll(db *sql.DB) {
	query := fmt.Sprintf("SELECT * FROM %s", m.c.Table)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("ERROR %s : %s", m.c.Name, err)
		return
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
			log.Printf("ERROR %s : %s", m.c.Name, err)
			return
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
		logstash(m.c.LogstashIP, string(x), m.c.LogstashPort)
	}
	return
}

//getTableOffset sends data from given count to logstash
func (m mssql) getTableOffset(db *sql.DB, count int) {
	query := fmt.Sprintf("SELECT * FROM %s ORDER BY (SELECT NULL) OFFSET %d ROWS", m.c.Table, m.c.RowCount)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("ERROR %s : %s", m.c.Name, err)
		return
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
			log.Printf("ERROR %s : %s", m.c.Name, err)
			return
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
		logstash(m.c.LogstashIP, string(x), m.c.LogstashPort)
	}
	return
}

//MYSQL FUNCTIONS
//getConn returns a conection to the database
func (m mysql) getConn() *sql.DB {
	for {
		tts := time.Minute * 5
		connString := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", m.c.DatabaseUser, m.c.DatabasePassword, m.c.ServerIP, m.c.ServerPort, m.c.Database)

		db, err := sql.Open(m.c.Driver, connString)
		if err != nil {
			log.Printf("ERROR %s : database connection : %s", m.c.Name, err)
			time.Sleep(tts)
			continue
		}
		//check if connection has been established
		err = db.Ping()
		if err != nil {
			log.Printf("ERROR %s : database connection : %s", m.c.Name, err)
			time.Sleep(tts)
			continue
		}
		return db

	}
}

//getCount fetches the row count
func (m mysql) getCount(db *sql.DB) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", m.c.Table)
	results, err := db.Query(query)
	if err != nil {
		log.Printf("ERROR %s getting count: %s", m.c.Name, err)
		return 0, nil
	}
	var count int
	for results.Next() {
		err = results.Scan(&count)
		if err != nil {
			log.Printf("ERROR %s getting count: %s", m.c.Name, err)
			return 0, nil
		}
	}
	return count, nil
}

//getTableAll sends all the data from the table to logstash
func (m mysql) getTableAll(db *sql.DB) {
	query := fmt.Sprintf("SELECT * FROM %s", m.c.Table)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("ERROR %s getting all table data: %s", m.c.Name, err)
		return
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
			log.Printf("ERROR %s : %s", m.c.Name, err)
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
					log.Printf("ERROR %s : %s", m.c.Name, err)
				}
				mm[i] = string(ints)
			}

		}

		// Outputs: map[columnName:value columnName2:value2 columnName3:value3 ...]
		x, _ := json.Marshal(mm)

		//Send to Logstash
		logstash(m.c.LogstashIP, string(x), m.c.LogstashPort)
	}
	return
}

//getTableOffset sends data from given count to logstash
func (m mysql) getTableOffset(db *sql.DB, count int) {
	row := count - m.c.RowCount
	query := fmt.Sprintf("SELECT * FROM ( SELECT * FROM %s ORDER BY %s DESC LIMIT %d ) sub ORDER BY %s ASC;", m.c.Table, m.c.Key, row, m.c.Key)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("ERROR %s getting all table data: %s", m.c.Name, err)
		return
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
			log.Printf("ERROR %s : %s", m.c.Name, err)
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
					log.Printf("ERROR %s : %s", m.c.Name, err)
				}
				mm[i] = string(ints)
			}

		}

		// Outputs: map[columnName:value columnName2:value2 columnName3:value3 ...]
		x, _ := json.Marshal(mm)

		//Send to Logstash
		logstash(m.c.LogstashIP, string(x), m.c.LogstashPort)
	}
	return
}

//POSTGRESQL FUNCTIONS
//getConn returns a conection to the database
func (m postgresql) getConn() *sql.DB {
	for {
		tts := time.Minute * 5
		connString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", m.c.ServerIP, m.c.ServerPort, m.c.DatabaseUser, m.c.DatabasePassword, m.c.Database)

		db, err := sql.Open(m.c.Driver, connString)
		if err != nil {
			log.Printf("ERROR %s : database connection : %s", m.c.Name, err)
			time.Sleep(tts)
			continue
		}
		//check if connection has been established
		err = db.Ping()
		if err != nil {
			log.Printf("ERROR %s : database connection : %s", m.c.Name, err)
			time.Sleep(tts)
			continue
		}
		return db

	}
}

//getCount fetches the row count
func (m postgresql) getCount(db *sql.DB) (int, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", m.c.Table)
	results, err := db.Query(query)
	if err != nil {
		log.Printf("ERROR %s getting count: %s", m.c.Name, err)
		return 0, nil
	}
	var count int
	for results.Next() {
		err = results.Scan(&count)
		if err != nil {
			log.Printf("ERROR %s getting count: %s", m.c.Name, err)
			return 0, nil
		}
	}
	return count, nil
}

//getTableAll sends all the data from the table to logstash
func (m postgresql) getTableAll(db *sql.DB) {
	query := fmt.Sprintf("SELECT * FROM %s", m.c.Table)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("ERROR %s : %s", m.c.Name, err)
		return
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
			log.Printf("ERROR %s : %s", m.c.Name, err)
			return
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
		logstash(m.c.LogstashIP, string(x), m.c.LogstashPort)
	}
	return
}

//getTableOffset sends data from given count to logstash
func (m postgresql) getTableOffset(db *sql.DB, count int) {
	query := fmt.Sprintf("SELECT * FROM %s OFFSET %d ROWS", m.c.Table, m.c.RowCount)
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("ERROR %s getting all table data: %s", m.c.Name, err)
		return
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
			log.Printf("ERROR %s : %s", m.c.Name, err)
			return
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
		logstash(m.c.LogstashIP, string(x), m.c.LogstashPort)
	}
	return
}

//MONGOD FUNCTIONS

func mongoCheckConn(c config) error {
	credential := options.Credential{
		Username: c.DatabaseUser,
		Password: c.DatabasePassword,
	}
	// Declare Context type object for managing multiple API requests
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connString := fmt.Sprintf("mongodb://%s:%d/", c.ServerIP, c.ServerPort)
	clientOptions := options.Client().ApplyURI(connString).SetAuth(credential)

	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return err
	}

	//CLOSE CONNECTION
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Printf("ERROR closing collection %s : %s", c.Name, err)
		}
	}()

	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		return err
	}
	return nil
}

func mongoGetTable(c config) error {
	credential := options.Credential{
		Username: c.DatabaseUser,
		Password: c.DatabasePassword,
	}
	// Declare Context type object for managing multiple API requests
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connString := fmt.Sprintf("mongodb://%s:%d/", c.ServerIP, c.ServerPort)
	clientOptions := options.Client().ApplyURI(connString).SetAuth(credential)

	// Connect to the MongoDB and return Client instance
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return err
	}
	//CLOSE CONNECTION
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Printf("ERROR closing collection %s : %s", c.Name, err)
		}
	}()

	// Access a MongoDB collection through a database
	col := client.Database("myNewDB").Collection("myNewCollection1")
	// Declare an empty array to store documents returned

	// Call the collection's Find() method to return Cursor obj
	// with all of the col's documents
	cursor, err := col.Find(context.TODO(), bson.D{})

	// Find() method raised an error
	if err != nil {
		defer cursor.Close(ctx)
		return err
		// If the API call was a success
	} else {
		// iterate over docs using Next()
		for cursor.Next(ctx) {

			// declare a result BSON object
			var result bson.M
			err := cursor.Decode(&result)

			// If there is a cursor.Decode error
			if err != nil {
				return err

				// If there are no cursor.Decode errors
			} else {
				x, _ := json.Marshal(result)

				//Send to Logstash
				logstash(c.LogstashIP, string(x), c.LogstashPort)
			}
		}
	}
	return nil
}

func mongoGetCount(c config) (int, error) {
	credential := options.Credential{
		Username: c.DatabaseUser,
		Password: c.DatabasePassword,
	}
	// Declare Context type object for managing multiple API requests
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connString := fmt.Sprintf("mongodb://%s:%d/", c.ServerIP, c.ServerPort)
	clientOptions := options.Client().ApplyURI(connString).SetAuth(credential)

	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return 0, err
	}

	//CLOSE CONNECTION
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	count, err := client.Database("myNewDB").Collection("myNewCollection1").CountDocuments(context.Background(), bson.D{})
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func mongoGetTableNew(c config, n int) error {
	credential := options.Credential{
		Username: c.DatabaseUser,
		Password: c.DatabasePassword,
	}
	// Declare Context type object for managing multiple API requests
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connString := fmt.Sprintf("mongodb://%s:%d/", c.ServerIP, c.ServerPort)
	clientOptions := options.Client().ApplyURI(connString).SetAuth(credential)

	// Connect to the MongoDB and return Client instance
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return err
	}
	//CLOSE CONNECTION
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Printf("ERROR closing collection %s : %s", c.Name, err)
		}
	}()

	// Access a MongoDB collection through a database
	col := client.Database("myNewDB").Collection("myNewCollection1")
	findOptions := options.Find()
	findOptions.SetSkip(int64(n))
	// Declare an empty array to store documents returned

	// Call the collection's Find() method to return Cursor obj
	// with all of the col's documents
	cursor, err := col.Find(context.TODO(), bson.D{}, findOptions)

	// Find() method raised an error
	if err != nil {
		defer cursor.Close(ctx)
		return err
		// If the API call was a success
	} else {
		// iterate over docs using Next()
		for cursor.Next(ctx) {

			// declare a result BSON object
			var result bson.M
			err := cursor.Decode(&result)

			// If there is a cursor.Decode error
			if err != nil {
				return err

				// If there are no cursor.Decode errors
			} else {
				x, _ := json.Marshal(result)
				//Send to Logstash
				logstash(c.LogstashIP, string(x), c.LogstashPort)
			}
		}
	}
	return nil
}
