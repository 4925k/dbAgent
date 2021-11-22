package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	lg "../logstash"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Mongod struct{ C Config }

func (m Mongod) TestConn() error {
	credential := options.Credential{
		Username: m.C.DatabaseUser,
		Password: m.C.DatabasePassword,
	}
	// Declare Context type object for managing multiple API requests
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connString := fmt.Sprintf("mongodb://%s:%d/", m.C.ServerIP, m.C.ServerPort)
	clientOptions := options.Client().ApplyURI(connString).SetAuth(credential)

	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return err
	}

	//CLOSE CONNECTION
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Printf("ERROR closing collection %s : %s", m.C.Name, err)
		}
	}()

	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		return err
	}
	return nil
}

func (m Mongod) GetCount() (int, error) {
	credential := options.Credential{
		Username: m.C.DatabaseUser,
		Password: m.C.DatabasePassword,
	}
	// Declare Context type object for managing multiple API requests
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connString := fmt.Sprintf("mongodb://%s:%d/", m.C.ServerIP, m.C.ServerPort)
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

func (m Mongod) GetTable() error {
	credential := options.Credential{
		Username: m.C.DatabaseUser,
		Password: m.C.DatabasePassword,
	}
	// Declare Context type object for managing multiple API requests
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connString := fmt.Sprintf("mongodb://%s:%d/", m.C.ServerIP, m.C.ServerPort)
	clientOptions := options.Client().ApplyURI(connString).SetAuth(credential)

	// Connect to the MongoDB and return Client instance
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return err
	}
	//CLOSE CONNECTION
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Printf("ERROR closing collection %s : %s", m.C.Name, err)
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
				lg.Send(m.C.LogstashIP, string(x), m.C.LogstashPort)
			}
		}
	}
	return nil
}

func (m Mongod) GetTableOffset(newCount int) error {
	credential := options.Credential{
		Username: m.C.DatabaseUser,
		Password: m.C.DatabasePassword,
	}
	// Declare Context type object for managing multiple API requests
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connString := fmt.Sprintf("mongodb://%s:%d/", m.C.ServerIP, m.C.ServerPort)
	clientOptions := options.Client().ApplyURI(connString).SetAuth(credential)

	// Connect to the MongoDB and return Client instance
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		return err
	}
	//CLOSE CONNECTION
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Printf("ERROR closing collection %s : %s", m.C.Name, err)
		}
	}()

	// Access a MongoDB collection through a database
	col := client.Database("myNewDB").Collection("myNewCollection1")
	findOptions := options.Find()
	findOptions.SetSkip(int64(m.C.RowCount))
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
				lg.Send(m.C.LogstashIP, string(x), m.C.LogstashPort)
			}
		}
	}
	return nil
}
