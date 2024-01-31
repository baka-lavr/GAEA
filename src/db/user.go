package db

import (
	//"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"context"
	"time"
) 

type User struct {
	Login string
	Password string
	Roles []string
	Name string
}

func (db Database) GetUser(login string) (User,error) {
	var res User
	collection := db.connection.Database(db.name).Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.D{bson.E{"login",login}}
	err := collection.FindOne(ctx,filter).Decode(&res)
	return res,err
}

func (db Database) GetUserList() ([]User,error) {
	var res []User
	collection := db.connection.Database(db.name).Collection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.D{}
	cursor, err := collection.Find(ctx,filter)
	if err != nil {
		return nil,err
	}
	err = cursor.All(ctx,&res)
	
	return res,err
}