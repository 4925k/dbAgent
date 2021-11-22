Config Parameters

* name:               to identify the current config
* driver:             driver to be used 
* serverIP:           ip address of the database server
* serverPort:         port used by datbase
* database:           name of the database where data is to be fetched from
* table:              name of the table
* key:                primary key by which the rows can be ordered
* databaseUser:       user to be used to fetch data
* databasePassword:   password to the user
* logstashIP:         logstash ip to send fetched data to
* logstashPort:       port used by the logstash 
* rowCount:           default is zero. kewps track of rows fetched by the program.
* firstTime:          default is true. identify if the config is a new addition
* tts:                time frame in which the program checks for new data



----------------------------------------------------------------------------------------
drivers for different databases

Microsoft SQL serverIP          - mssql
SQL server                      - mysql
Maria DB                        - mysql
PostgreSQL                      - postgres
MongoDB                         - mongod

----------------------------------------------------------------------------------------
example single config


config:
- name: postgresql  
  driver: postgres  
  serverIP: 192.168.10.67  
  serverPort: 5432  
  database: license  
  table: licenseinfo  
  key: b01  
  databaseUser: vairav  
  databasePassword: vairav  
  logstashIP: 192.168.10.64  
  logstashPort: 1234  
  rowCount: 8  
  firstTime: false  
  tts: 1ns  


----------------------------------------------------------------------------------------
example multiple config

config:
- name: clientX  
  driver: mssql  
  serverIP: 192.168.10.67  
  serverPort: 3306  
  database: test  
  table: customers  
  key: customerID  
  databaseUser: test  
  databasePassword: Pa$$word123  
  logstashIP: 192.168.1.64  
  logstashPort: 1234  
  rowCount: 0  
  firstTime: true  
  tts: 5ns  
- name: clientY  
  driver: mysql  
  serverIP: 192.168.10.67  
  serverPort: 3306  
  database: test  
  table: tasks  
  key: id  
  databaseUser: test  
  databasePassword: Pa$$word123  
  logstashIP: 192.168.10.64  
  logstashPort: 1234  
  rowCount: 2  
  firstTime: false  
  tts: 1ns