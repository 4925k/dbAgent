----------------------------------------------------------------------------------------
how to install

1. make a folder with:
	<main folder>
		- dbAgent.exe
		- config
		- log
	*config and log are folders

2. make a config as mentioned below and put it inside config with the name => "config.yaml"

3. run the dbAgent program in background.

----------------------------------------------------------------------------------------

how to add another database.
1. make newDB.go inside db directory. make required functions. base format given below.
    package db
    
    type NewDB struct { C Config }
    
    func (x NewDB) TestConn() error{
      return nil
    }
    
    func	(x NewDB) GetCount() (int, error) {
      return 0, nil
    }
    
    func (x NewDB) GetTable() error {
      return nil
    }
    
    func (x NewDB) GetTableOffset(newCount int) error {
      return nil
    }
    
    * fill functions with required code  

2. add in func structConv under interface.go
    case "NewDB":
        return NewDB{config}

3. test it out

----------------------------------------------------------------------------------------