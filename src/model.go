package main

import (
	"github.com/baka-lavr/gaea/src/db"
	"strings"
	"os"
	"fmt"
	"path/filepath"
	"io"
	"time"
	"net/http"
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	_"log"
)

func (a *Auth) LoginAttempt(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	login := r.FormValue("login")
	password := r.FormValue("password")
	usr, err := a.db.GetUser(login)
	if err != nil {
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}
	if usr.Password != password {
		http.Error(w, "Wrong password", http.StatusInternalServerError)
		return
	}
	s_token, r_token := a.NewTokens(login)
	http.SetCookie(w, &http.Cookie{
		Name: "session_token",
		Value: s_token,
		Expires: time.Now().Add(time.Minute*5),
	})
	http.SetCookie(w, &http.Cookie{
		Name: "refresh_token",
		Value: r_token,
		Expires: time.Now().Add(time.Hour),
	})
	//w.Header().Set("Location", "/")
	w.WriteHeader(200)
}

func (a *Auth) Logout(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	//var user, found = "", false
	tokenCache, err := r.Cookie("session_token")
	if err == nil {
		token := tokenCache.Value
		a.session.Delete(token)
	}
	refreshCache, err := r.Cookie("refresh_token")
	if err == nil {
		token := refreshCache.Value
		a.refresh.Delete(token)
	}

	http.SetCookie(w, &http.Cookie{
		Name: "session_token",
		Value: "",
		Expires: time.Now(),
	})
	http.SetCookie(w, &http.Cookie{
		Name: "refresh_token",
		Value: "",
		Expires: time.Now(),
	})
}

func (c *Controller) Test(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var d string
	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	res, err := json.Marshal("Got data: "+d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write(res)
}

func (c *Controller) UploadDoc(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	const MAX_FILE_SIZE = 1024*1024
	title := r.FormValue("title")
	comment := r.FormValue("comment")
	if err := r.ParseMultipartForm(MAX_FILE_SIZE); err != nil {
		http.Error(w, "Слишком большой файл", http.StatusBadRequest)
		return
	}
	file,header,err := r.FormFile("document")
	if err != nil {
		http.Error(w, "Ошибка загрузки файла", http.StatusBadRequest)
		return
	}
	defer file.Close()
	name := fmt.Sprintf("%d%s",time.Now().UnixNano(),filepath.Ext(header.Filename))
	exe,_ := os.Executable()
	ospath := filepath.Dir(exe)
	dest,err := os.Create(filepath.Join(ospath,"db","docs",name))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
		return
	}
	defer dest.Close()
	_,err = io.Copy(dest, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
		return
	}
	doc := db.Document{File:name,User:r.Context().Value("user").(db.User).Login,Title:title,Comment:comment,Date:time.Now().UTC()}
	err = c.db.CreateDoc(doc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
		os.Remove(filepath.Join(ospath,"db","docs",name))
		return
	}
	fmt.Fprintf(w,"Загрузка успешна")
}

func (c *Controller) Redirect(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	comment := r.FormValue("comment")
	doc := r.FormValue("doc")
	user := r.Context().Value("user").(db.User)
	r.ParseForm()
	users := r.Form["users"]
	if len(users) == 0 {
		http.Error(w, "Выберите хотя бы одного пользователя", http.StatusInternalServerError)
		return
	}
	err := c.db.Redirect(user.Login,doc,comment,users)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
	}
}

func (c *Controller) Send(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	to := strings.Split(strings.ReplaceAll(r.FormValue("to")," ",""),",")
	title := r.FormValue("title")
	body := r.FormValue("body")
	user := r.Context().Value("user").(db.User).Login
	
	doc,err := c.db.GetDocByID(r.FormValue("doc"), true, user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
		return
	}
	exe,_ := os.Executable()
	ospath := filepath.Dir(exe)
	path := filepath.Join(ospath,"db","docs",doc.File)
	err = c.mail.Send(title,body,path,to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
		return
	}
	err = c.db.Archiving(doc.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err)
	}
}
