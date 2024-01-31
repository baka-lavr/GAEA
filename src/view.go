package main

import (
	"github.com/baka-lavr/gaea/src/db"
	"log"
	_"io/ioutil"
	"fmt"
	_"time"
	"net/http"
	"github.com/julienschmidt/httprouter"
	_"github.com/unrolled/render"
	"html/template"
)

type Filler struct {
	User db.User
	Content interface{}
}

func (c *Controller) View(w http.ResponseWriter, r *http.Request, page string, content interface{}) {
	user := r.Context().Value("user").(db.User)
	user.Password = ""
	filler := Filler{User: user, Content: content}
	c.render.HTML(w,http.StatusOK,page,filler)
}

func (c *Controller) Main(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	c.View(w,r,"home",nil)
}

func (c *Controller) DocPage(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("doc")
	user := r.Context().Value("user").(db.User)
	doc,err := c.db.GetDocByID(id,true,user.Login)
	if err != nil {
		w.Header().Set("Location", "/")
		w.WriteHeader(303)
		return
	}
	c.View(w,r,"doc",doc)
}

func (c *Controller) DocPageProtected(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("doc")
	doc,err := c.db.GetDocByID(id,false,"")
	log.Print(doc.Id)
	if err != nil {
		w.Header().Set("Location", "/")
		w.WriteHeader(303)
		return
	}
	c.View(w,r,"doc_protected",doc)
}


func (c *Controller) Login(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	//w.Header().Set("LOGIN","1")
	c.render.HTML(w,http.StatusOK,"login",nil)
}

func (c *Controller) ActualDocs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	docs, err := c.db.GetDocs(r.Context().Value("user").(db.User).Login,false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
	}
	tmpl, err := template.ParseFiles("front/html/elements/actual_docs.tmpl")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
	}
	tmpl.Execute(w,docs)
}

func (c *Controller) HeadDocs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	docs, err := c.db.GetAllDocs(false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
	}
	tmpl, err := template.ParseFiles("front/html/elements/head_docs.tmpl")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
	}
	tmpl.Execute(w,docs)
}

func (c *Controller) HeadArchive(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	docs, err := c.db.GetAllDocs(true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
	}
	tmpl, err := template.ParseFiles("front/html/elements/head_docs.tmpl")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
	}
	tmpl.Execute(w,docs)
}

func (c *Controller) FormRedirect(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	users, err := c.db.GetUserList()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
	}
	tmpl, err := template.ParseFiles("front/html/elements/redirect.tmpl")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
	}
	tmpl.Execute(w,users)
}

func (c *Controller) FormSend(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	tmpl, err := template.ParseFiles("front/html/elements/send.tmpl")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
	}
	tmpl.Execute(w,nil)
}

func (c *Controller) GetFile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	user := r.Context().Value("user").(db.User)
	doc := r.URL.Query().Get("doc")
	admin := false
	for _,j := range user.Roles {
		if j == "admin" {
			admin = true
		}
	}
	var path db.Document
	var err error
	if admin {
		path,err = c.db.GetDocByID(doc,false,"")
	} else {
		path,err = c.db.GetDocByID(doc,true,user.Login)
	}
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
	}
	//w.Header().Set("filename", path.File)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s",path.File))
	http.ServeFile(w,r,fmt.Sprintf("./db/docs/%s",path.File))
	//file, err := ioutil.ReadFile(fmt.Sprintf("./db/docs/%s",path.File))
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//}
	//w.Header().Set("Content-Type","application/octet-stream")
	//w.Write(file)
}