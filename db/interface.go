package db

import (
	"time"
)

type ConfigList struct {
	Config []Config `yaml:"config"`
}

type Config struct {
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

type Functions interface {
	TestConn() error                   //test connection to the db.
	GetCount() (int, error)            //get table count
	GetTable() error                   //fetch all table and send to give logstash ip:port
	GetTableOffset(newCount int) error //fetch newer data and send to logstash
}

//structConv converts the given basic config into a databse specific struct
func StructConv(config Config) Functions {
	switch config.Driver {
	case "mongod":
		return Mongod{config}
	case "postgres":
		return Postgres{config}
	case "mysql":
		return Mysql{config}
	case "mssql":
		return Mssql{config}
	}
	return nil
}
