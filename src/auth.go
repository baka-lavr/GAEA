package main

import (
	"github.com/baka-lavr/gaea/src/db"
	"net/http"
	"log"
	"time"
	"context"
	"strings"
	"github.com/julienschmidt/httprouter"
	"github.com/patrickmn/go-cache"
	"github.com/google/uuid"
)

type Auth struct {
	db *db.Database
	whitelist map[string]string
	handler httprouter.Router
	session *cache.Cache
	refresh *cache.Cache
}

func NewAuth(handler httprouter.Router, db *db.Database) *Auth {
	auth := Auth {
		db: db,
		whitelist: make(map[string]string),
		handler: handler,
		session: cache.New(5*time.Minute, 10*time.Minute),
		refresh: cache.New(time.Hour, 10*time.Minute),
	}
	return &auth
}

func (auth *Auth) NewTokens(user string) (string,string) {
	s_token := uuid.New().String()
	r_token := uuid.New().String()
	auth.session.Set(s_token, user, cache.DefaultExpiration)
	auth.refresh.Set(r_token, user, cache.DefaultExpiration)
	return s_token, r_token
}

func (auth *Auth) GET(path string, role string, f httprouter.Handle) {
	path2,_,found := strings.Cut(path,":")
	if found {
		path2 = path2+":"
	}
	auth.handler.GET(path,f)
	auth.whitelist[path2] = role
}
func (auth *Auth) POST(path string, role string, f httprouter.Handle) {
	auth.handler.POST(path,f)
	auth.whitelist[path] = role
}

func (auth Auth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	role := auth.whitelist[r.URL.Path]
	//log.Println(role)
	if role=="" {
		st := strings.Split(r.URL.Path,"/")
		st2 := strings.TrimRight(r.URL.Path,st[len(st)-1])
		role = auth.whitelist[st2+":"]
		//log.Println(role)
		if role == "" {
			auth.handler.ServeHTTP(w,r)
			return
		}
	}
	var user, found = "", false
	tokenCache, err := r.Cookie("session_token")
	if err == nil {
		token := tokenCache.Value
		var value interface{}
		value, found = auth.session.Get(token)
		if found {
			user = value.(string)
		}
	}
	if err != nil || !found {
		w.Header().Set("Location", "/refresh?return="+string(r.URL.Path))
		w.WriteHeader(303)
		//http.Redirect(w,r,"/refresh?return="+string(r.URL.Path),http.StatusSeeOther)
		return
	}
	userObj, err := auth.db.GetUser(user)
	if err != nil {
		http.Error(w, "User not found", http.StatusRequestTimeout)
		return
	}
	access := false
	
	for _,j := range userObj.Roles {
		if j == role {access = true}
	}
	if !access {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	ctx := context.WithValue(r.Context(),"user", userObj)
	auth.handler.ServeHTTP(w,r.WithContext(ctx))
}

func (auth *Auth) Refresh(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	user, found := "", false
	tokenCache, err := r.Cookie("refresh_token")
	if err == nil {
		token := tokenCache.Value
		var value interface{}
		value, found = auth.refresh.Get(token)
		if found {
			user = value.(string)
		}
	}
	if err != nil || !found {
		if r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
			http.Error(w,"Login please",http.StatusUnauthorized)
			return
		}
		w.Header().Set("Location", "/login")
		w.WriteHeader(303)
		return
		//http.Redirect(w,r,"/login",http.StatusSeeOther)
	}
	s_token, r_token := auth.NewTokens(user)
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
	//http.Redirect(w,r,r.URL.Query().Get("return"),http.StatusSeeOther)
	w.Header().Set("Location", r.URL.Query().Get("return"))
	w.WriteHeader(303)
}

func NotAuthorize(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		log.Println("Authorized")
		h(w,r,nil)
	}
}
