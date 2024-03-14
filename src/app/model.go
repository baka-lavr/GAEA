package app

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
	"log"
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
	a.NewTokens(login,w,r)
	//w.Header().Set("Location", "/")
	w.WriteHeader(200)
}

func (a *Auth) Logout(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	//var user, found = "", false
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
	files := make([]string,0)
	tags := r.Form["tags"]
	if err := r.ParseMultipartForm(MAX_FILE_SIZE); err != nil {
		http.Error(w, "Слишком большой файл", http.StatusBadRequest)
		return
	}

	fs := r.MultipartForm.File["document"]
	for _,header := range fs {
		file, err := header.Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Print(err)
			//logger.Error(err)
			return
		}
		defer file.Close()
		//logger.Info(header.Filename)
		name := fmt.Sprintf("%s%d%s",header.Filename[:len(header.Filename)-len(filepath.Ext(header.Filename))],time.Now().UnixNano(),filepath.Ext(header.Filename))
		exe,_ := os.Executable()
		ospath := filepath.Dir(exe)
		dest,err := os.Create(filepath.Join(ospath,"db","docs",name))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Print(err)
			//ogger.Error(err)
			return
		}
		defer dest.Close()
		_,err = io.Copy(dest, file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Print(err)
			//logger.Error(err)
			return
		}
		files = append(files, name)
	}
	

	//file,header,err := r.FormFile("document")
	//if err != nil {
	//	http.Error(w, "Ошибка загрузки файла", http.StatusBadRequest)
	//	return
	//}
	//defer file.Close()
	//name := fmt.Sprintf("%d%s",time.Now().UnixNano(),filepath.Ext(header.Filename))
	//exe,_ := os.Executable()
	//ospath := filepath.Dir(exe)
	//dest,err := os.Create(filepath.Join(ospath,"db","docs",name))
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	logger.Error(err)
	//	return
	//}
	//defer dest.Close()
	//_,err = io.Copy(dest, file)
	//if err != nil {
	//	http.Error(w, err.Error(), http.StatusInternalServerError)
	//	logger.Error(err)
	//	return
	//}
	exe,_ := os.Executable()
	ospath := filepath.Dir(exe)
	doc := db.Document{Files:files,User:r.Context().Value("user").(db.User).Login,Title:title,Comment:comment,Date:time.Now().UTC(),Tags:tags}
	err := c.db.CreateDoc(doc)
	if err != nil {
		for _,name := range files {
			os.Remove(filepath.Join(ospath,"db","docs",name))
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Print(err)
		//logger.Error(err)
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
		log.Print(err)
		//logger.Error(err)
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
		log.Print(err)
		//logger.Error(err)
		return
	}
	exe,_ := os.Executable()
	ospath := filepath.Dir(exe)
	paths := make([]string,0)
	for _,j := range doc.Files{
		paths = append(paths,filepath.Join(ospath,"db","docs",j))
	}
	err = c.mail.Send(title,body,paths,to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Print(err)
		//logger.Error(err)
		return
	}
	err = c.db.Archiving(doc.Id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Print(err)
		//logger.Error(err)
	}
}
