package app

import (
	"github.com/baka-lavr/gaea/src/db"
	"net/http"
	"log"
	"time"
	"context"
	"strings"
	"github.com/julienschmidt/httprouter"
	"github.com/patrickmn/go-cache"
	_"github.com/google/uuid"
	"crypto/sha512"
	"crypto/hmac"
	"encoding/base64"
	"bytes"
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	secret string
	db *db.Database
	whitelist map[string]string
	handler httprouter.Router
	refresh *cache.Cache
}

type JWTheader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}
type JWTpayload struct {
	Sub string `json:"sub"`
	Iat string `json:"iat"`
}

func NewAuth(handler httprouter.Router, db *db.Database) *Auth {
	auth := Auth {
		secret: "thegoddessoftheearth",
		db: db,
		whitelist: make(map[string]string),
		handler: handler,
		refresh: cache.New(time.Hour, 10*time.Minute),
	}
	return &auth
}

func (auth *Auth) GenerateAccessToken(user string) (string,error) {
	jwt_header := JWTheader{"HS512","JWT",}
	jwt_payload := JWTpayload{user,time.Now().Add(time.Minute*5).Format("20060102150405"),}

	var buffer bytes.Buffer
	b64_encoder := base64.NewEncoder(base64.URLEncoding,&buffer)
	json_encoder := json.NewEncoder(b64_encoder)
	err := json_encoder.Encode(jwt_header)
	if err != nil {
		return "", err
	}
	j_head := buffer.String()
	buffer.Reset()
	err = json_encoder.Encode(jwt_payload)
	if err != nil {
		return "", err
	}
	j_pay := buffer.String()
	b64_encoder.Close()

	mac := hmac.New(sha512.New,[]byte(auth.secret))
	mac.Write([]byte(j_head+"."+j_pay))
	signature := base64.URLEncoding.WithPadding(-1).EncodeToString(mac.Sum(nil))
	jwt := j_head+"."+j_pay+"."+signature
	return jwt, nil
}
func (auth *Auth) GenerateRefreshToken(user, fingerprint string) (string,error) {
	str := user+fingerprint
	hash,err := bcrypt.GenerateFromPassword([]byte(str),10)
	return base64.URLEncoding.WithPadding(-1).EncodeToString(hash),err
}

func (auth *Auth) VerifySessionToken(token string) (time.Time,string,bool) {
	log.Print("DEBUG")
	sub := strings.Split(token,".")
	json_str,err := base64.URLEncoding.DecodeString(sub[1])
	
	if err != nil {
		return time.Now(),"",false
	}
	
	var data JWTpayload
	err = json.Unmarshal(json_str,&data)
	if err != nil {
		return time.Now(),"",false
	}
	expire,err := time.Parse("20060102150405", data.Iat)
	if err != nil {
		return time.Now(),"",false
	}

	sign := sub[2]
	mac := hmac.New(sha512.New,[]byte(auth.secret))
	mac.Write([]byte(sub[0]+"."+sub[1]))
	signature := base64.URLEncoding.WithPadding(-1).EncodeToString(mac.Sum(nil))
	correct := hmac.Equal([]byte(signature),[]byte(sign))
	return expire,data.Sub,correct
}
func (auth *Auth) VerifyRefreshToken(token,fingerprint string) (string,bool) {
	var value interface{}
	value, found := auth.refresh.Get(token)
	if !found {
		return "",false
	}
	user := value.(string)
	str,err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return "",false
	}
	err = bcrypt.CompareHashAndPassword([]byte(str),[]byte(user+fingerprint))
	if err != nil {
		return "",false
	}
	return user,true
}

func (auth *Auth) NewTokens(user string, w http.ResponseWriter, r *http.Request) error {
	s_token,err := auth.GenerateAccessToken(user)
	if err != nil {
		return err
	}
	r_token,err := auth.GenerateRefreshToken(user, r.Header.Get("User-Agent"))
	if err != nil {
		return err
	}
	auth.refresh.Set(r_token, user, cache.DefaultExpiration)
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
	return nil
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
	var expire time.Time
	var user, found = "", false
	tokenCache, err := r.Cookie("session_token")
	if err == nil {
		token := tokenCache.Value
		expire, user, found = auth.VerifySessionToken(token)
	}
	log.Print(found)
	log.Print(expire)
	if err != nil || !found || time.Now().After(expire) {
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
		user, found = auth.VerifyRefreshToken(token,r.Header.Get("User-Agent"))
		auth.refresh.Delete(token)
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
	err = auth.NewTokens(user,w,r)
	if err != nil {
		w.Header().Set("Location", "/login")
		w.WriteHeader(303)
	}
	
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
