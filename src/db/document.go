package db

import (
	//"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"context"
	"time"
	"log"
	"errors"
) 

type Document struct {
	Id string `bson:"-"`
	ParentName string `bson:"-"`
	Objid interface{} `bson:"_id,omitempty"`
	File string
	User string
	Title string
	Comment string
	Date time.Time
	Parent string
	Child []string
	Archive bool
}

func (db Database) CreateDoc(doc Document) error {
	//doc := Document{File:path,User:user,Comment:comment,Date:time.Now()}
	collection := db.connection.Database(db.name).Collection("documents")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_,err := collection.InsertOne(ctx,doc)
	return err
}

func (db Database) GetDocs(user string, archive bool) ([]Document,error) {
	var res []Document
	collection := db.connection.Database(db.name).Collection("documents")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.D{bson.E{"user",user},bson.E{"archive",archive}}
	cursor, err := collection.Find(ctx,filter)
	if err != nil {
		return nil,err
	}
	err = cursor.All(ctx,&res)
	for i,j := range res {
		loc,err := time.LoadLocation("Europe/Moscow")
		if err == nil {
			res[i].Date = res[i].Date.In(loc)
			//logger.Info(res[i].Date.In(loc))
		}
		
		res[i].Id = j.Objid.(primitive.ObjectID).Hex()
		if j.Parent != "" {
			parent,err := db.GetDocByID(j.Parent,false,"")
			if err == nil {
				p_usr, err := db.GetUser(parent.User)
				if err == nil {
					res[i].ParentName = p_usr.Name
				}
			}
		}
	}
	return res,err
}

func (db Database) GetAllDocs(archive bool) ([]Document,error) {
	var res []Document
	collection := db.connection.Database(db.name).Collection("documents")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	filter := bson.D{bson.E{"archive",archive},bson.E{"child",nil},bson.E{"parent",bson.D{{"$ne",""}}}}
	cursor, err := collection.Find(ctx,filter)
	if err != nil {
		return nil,err
	}
	err = cursor.All(ctx,&res)
	for i,j := range res {
		loc,err := time.LoadLocation("Europe/Moscow")
		if err == nil {
			res[i].Date = res[i].Date.In(loc)
		}
		res[i].Id = j.Objid.(primitive.ObjectID).Hex()
		
		usr, err := db.GetUser(j.User)
		if err == nil {
			res[i].ParentName = usr.Name
		}
	}
	return res,err
}

func (db Database) GetDocByID(id string, protected bool, user string) (Document,error) {
	var res Document
	collection := db.connection.Database(db.name).Collection("documents")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	obj,_ := primitive.ObjectIDFromHex(id)
	filter := bson.D{bson.E{"_id",obj}}
	err := collection.FindOne(ctx,filter).Decode(&res)

	if err != nil {
		return res,err
	}

	if protected && res.User != user {
		return res, errors.New("Access denied")
	}
	loc,err := time.LoadLocation("Europe/Moscow")
	if err == nil {
		res.Date = res.Date.In(loc)
	}
	res.Id = res.Objid.(primitive.ObjectID).Hex()
	if res.Parent != "" {
		parent,err := db.GetDocByID(res.Parent,false,"")
		if err == nil {
			p_usr, err := db.GetUser(parent.User)
			if err == nil {
				res.ParentName = p_usr.Name
			}
		}
	}
	return res,nil
}

func (db Database) DeleteDocs(id ...string) error {
	for _,j := range id {
		collection := db.connection.Database(db.name).Collection("documents")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		obj,_ := primitive.ObjectIDFromHex(j)
		filter := bson.D{bson.E{"_id",obj}}
		_,err := collection.DeleteOne(ctx,filter)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db Database) Redirect(user,doc,comment string, users []string) error {
	old,err := db.GetDocByID(doc,true,user)
	log.Print(users)
	if err != nil {
		return err
	}
	var copies []string
	collection := db.connection.Database(db.name).Collection("documents")
	for _,j := range users {
		
		doc := Document{File:old.File,Title:old.Title,User:j,Comment:comment,Date:time.Now(),Parent:old.Id}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		id,err := collection.InsertOne(ctx,doc)
		copies = append(copies, id.InsertedID.(primitive.ObjectID).Hex())
		if err != nil || j == user {
			log.Print("error")
			_ = db.DeleteDocs(copies...)
			return err
		}
	}
	_,err = collection.UpdateByID(context.Background(), old.Objid,bson.D{{"$set",bson.D{{"child",copies}}}})
	if err != nil {
		return err
	}
	err = db.Archiving(old.Id)
	return err
}

func (db Database) Archiving(id string) error {
	collection := db.connection.Database(db.name).Collection("documents")
	obj,_ := primitive.ObjectIDFromHex(id)
	_,err := collection.UpdateByID(context.Background(), obj,bson.D{{"$set",bson.D{{"archive",true}}}})
	return err

}