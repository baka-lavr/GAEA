package db

import (
	"context"
	"log"
	"time"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Database struct {
	name string
	connection *mongo.Client
}

func OpenDB(ip string,port int,name,user,pass string) (Database, error) {
	cred := options.Credential{
		Username: user,
		Password: pass,
	}
	opt := options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%d",ip,port)).SetAuth(cred)
	context, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	client,err := mongo.Connect(context, opt)
	db := Database{name,client}
	if err != nil {
		return db, err
	}
	log.Print("DataBase connected succesfully")
	err = client.Ping(context, readpref.Primary())
	return db, err
}

func (db Database) Close() {
	context, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	err := db.connection.Disconnect(context)
	if err != nil {
		log.Print(err)
	} else {
		log.Print("DataBase connection closed")
	}
}