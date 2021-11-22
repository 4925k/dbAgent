package main

import (
	"io/ioutil"
	"log"
	"os"
	"time"

	db "./db"
	"gopkg.in/yaml.v2"
)

//logging is setup here
func init() {
	//set up logging
	logFile, err := os.OpenFile("log/mssqlAgent.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("ERROR opening logging file: ", err)
	}
	log.SetOutput(logFile)
	log.Print("INFO logging started")

	//did not close logFile because it keeps running
	//logFile.Close()
}

type configList db.ConfigList

func main() {
	var c configList                       //config list struct
	var exit = make(chan bool)             //channel to wait for go routine
	configLocation := "config/config.yaml" //path to config file

	//reads config file and stores to to vairable c
	c.getConf(configLocation)

	//looping to get each config
	for i, config := range c.Config {
		//starts fetching routine for each config
		go startRoutine(i, config, c)
	}

	//empty channel to let the go routine running
	<-exit
}

//getConf fetches config from the given location into a struct
func (c *configList) getConf(location string) {

	yamlFile, err := ioutil.ReadFile(location)
	if err != nil {
		log.Fatalf("ERROR : opening config file : %s", err)
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}
}

//updateConfig marshals data into yaml and updates the file in given location
func updateConfig(data configList) error {
	newConfig, _ := yaml.Marshal(data)

	err := ioutil.WriteFile("config/config.yaml", newConfig, 0644)
	if err != nil {
		return err
	}
	return nil
}

//startRoutine starts fetching data for config
func startRoutine(index int, config db.Config, fullConfig configList) {
	dbType := db.StructConv(config)
	//if db cant connect, dont continue.
	//add: keep count? if n tries, stop trying?
	for {
		err := dbType.TestConn()
		if err != nil {
			log.Printf("ERROR %s : database connection : %s", config.Name, err)
			time.Sleep(5 * time.Minute)
			continue
		}
		log.Printf("INFO %s : connection test successfull", config.Name)
		break
	}
	for {
		//if its the first time running
		if config.FirstTime {
			//get whole table
			err := dbType.GetTable()
			if err != nil {
				log.Printf("ERROR %s : fetching all data : %s", config.Name, err)
				time.Sleep(5 * time.Minute)
				continue
			}
			log.Printf("INFO %s : all data sent", config.Name)
			config.FirstTime = false
			//update rowCount, firstTime boolean and update config
			config.RowCount, err = dbType.GetCount()
			if err != nil {
				log.Printf("ERROR %s : fetching count : %s", config.Name, err)
				time.Sleep(5 * time.Minute)
				continue
			}
			fullConfig.Config[index].FirstTime = false
			fullConfig.Config[index].RowCount = config.RowCount
			updateConfig(fullConfig)
			log.Printf("INFO %s : config updated", config.Name)
		} else {
			newCount, err := dbType.GetCount()
			if err != nil {
				log.Printf("ERROR %s : fetching rowCount : %s", config.Name, err)
				time.Sleep(5 * time.Minute)
				continue
			}
			if newCount > config.RowCount {
				log.Printf("INFO %s : new data found", config.Name)
				dbType.GetTableOffset(newCount)
				log.Printf("INFO %s : new data sent", config.Name)
			} else {
				log.Printf("INFO %s : data up to date", config.Name)
			}

			config.RowCount = newCount
			fullConfig.Config[index].RowCount = newCount
			updateConfig(fullConfig)
			log.Printf("INFO %s : config updated", config.Name)
		}
		log.Printf("INFO %s : sleeping", config.Name)
		time.Sleep(config.TimeToSleep * time.Minute)

	}
}
