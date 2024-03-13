package app

import (
	"github.com/baka-lavr/gaea/src/db"
	//"fmt"
	"log"
	"encoding/json"
	"net/http"
	"github.com/julienschmidt/httprouter"
	"context"
	"os"
	//"syscall"
	"path/filepath"
	//"os/signal"
	"github.com/unrolled/render"
	//"github.com/kardianos/service"
	_"time/tzdata"
)

//var logger service.Logger

type Configuration struct {
	DBIp string
	DBPort int
	DBName string
	DBUser string
	DBPass string
	MailHost string
	MailPort int
	MailUser string
	MailPass string
}

type Controller struct {
	server http.Server
	db db.Database
	auth Auth
	render render.Render
	mail *MailConnection
}

func (c *Controller) Start() error {
	go c.run()
	return nil
}

func (c *Controller) Stop() error {
	c.db.Close()
	c.mail.Client.Close()
	c.server.Shutdown(context.Background())
	return nil
}

func (c *Controller) run() {
	exe,err := os.Executable()
	if err != nil {
		//logger.Error(err)
		log.Fatal(err)
	}
	path := filepath.Dir(exe)
	file, err := os.Open(filepath.Join(path,"conf.json"))
	defer file.Close()
	if err != nil {
		//logger.Error(err)
		log.Fatal(err)
	}
	decoder := json.NewDecoder(file)
	conf := Configuration{}
	err = decoder.Decode(&conf)
	if err != nil {
		//logger.Error(err)
		log.Fatal(err)
	}

	mail_con, err := MailConnect(conf)
	if err != nil {
		//logger.Error(err)
		log.Fatal(err)
	}
	c.mail = &mail_con

	db, err := db.OpenDB(conf.DBIp,conf.DBPort,conf.DBName,conf.DBUser,conf.DBPass)
	defer db.Close()
	if err != nil {
		//logger.Error(err)
		log.Fatal(err)
	}
	log.Print("DataBase pinged")
	//logger.Info("DataBase pinged")
	c.db = db
	
	router := httprouter.New()
	auth := Routing(router, c)

	server := http.Server {
		Addr: ":8080",
		Handler: auth,
	}
	c.server = server

	render := render.New(render.Options{
		Directory: filepath.Join(path,"front","html"),
		Layout: "layout",
	})
	c.render = *render

	if err := c.server.ListenAndServe(); err != nil {
		log.Fatal(err)
		//logger.Error(err)
	}	
}