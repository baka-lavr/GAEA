package app

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
	"path/filepath"
	"os"
	"strconv"
)

type Filter struct {
	Tags []db.Tag
	URL string
	Id string
}

type Filler struct {
	User db.User
	Content interface{}
}

func (c *Controller) ParseElement(name string) (*template.Template, error) {
	exe,_ := os.Executable()
	os := filepath.Dir(exe)
	tmpl, err := template.ParseFiles(filepath.Join(os,"front","html","elements",name))
	return tmpl, err
}

func (c *Controller) View(w http.ResponseWriter, r *http.Request, page string, content interface{}) {
	user := r.Context().Value("user").(db.User)
	user.Password = ""
	filler := Filler{User: user, Content: content}
	c.render.HTML(w,http.StatusOK,page,filler)
}

func (c *Controller) Main(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	tags, err := c.db.GetTags()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	c.View(w,r,"home",tags)
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
	filter := r.URL.Query()["filter"]
	docs, err := c.db.GetDocs(r.Context().Value("user").(db.User).Login,false,filter)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Print(err)
		//logger.Error(err)
	}
	tmpl, err := c.ParseElement("actual_docs.tmpl")
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Print(err)
		//logger.Error(err)
	}
	tmpl.Execute(w,docs)
}

func (c *Controller) HeadDocs(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	docs, err := c.db.GetAllDocs(false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Print(err)
		//logger.Error(err)
	}
	tmpl, err := c.ParseElement("head_docs.tmpl")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Print(err)
		//logger.Error(err)
	}
	tmpl.Execute(w,docs)
}

func (c *Controller) HeadArchive(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	docs, err := c.db.GetAllDocs(true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Print(err)
		//logger.Error(err)
	}
	tmpl, err := c.ParseElement("head_docs.tmpl")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Print(err)
		//logger.Error(err)
	}
	tmpl.Execute(w,docs)
}

func (c *Controller) FormRedirect(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	users, err := c.db.GetUserList()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Print(err)
		//logger.Error(err)
	}
	tmpl, err := c.ParseElement("redirect.tmpl")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Print(err)
		//logger.Error(err)
	}
	tmpl.Execute(w,users)
}

func (c *Controller) FormSend(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	tmpl, err := c.ParseElement("send.tmpl")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Print(err)
		//logger.Error(err)
	}
	tmpl.Execute(w,nil)
}

func (c *Controller) GetFile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	user := r.Context().Value("user").(db.User)
	doc := r.URL.Query().Get("doc")
	number,_ := strconv.Atoi(r.URL.Query().Get("number"))
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
		log.Print(err)
		//logger.Error(err)
	}

	exe,_ := os.Executable()
	os := filepath.Dir(exe)

	//w.Header().Set("filename", path.File)
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s",path.Files[number]))
	http.ServeFile(w,r,filepath.Join(os,"db","docs",path.Files[number]))
	//file, err := ioutil.ReadFile(fmt.Sprintf("./db/docs/%s",path.File))
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//}
	//w.Header().Set("Content-Type","application/octet-stream")
	//w.Write(file)
}

func (c *Controller) TagList(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := r.URL.Query()
	tags, err := c.db.GetTags()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Print(err)
	}
	tmpl, err := c.ParseElement("tags.tmpl")
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Print(err)
		//logger.Error(err)
	}
	filter := Filter {
		Tags: tags,
		URL: query["url"][0],
		Id: query["id"][0],
	}
	tmpl.Execute(w,filter)
}