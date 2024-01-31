package main

import (
	"net/http"
	"github.com/julienschmidt/httprouter"
)

func Routing(router *httprouter.Router, c *Controller) Auth {
	router.GET("/login", c.Login)
	router.ServeFiles("/static/*filepath", http.Dir("./front/static"))
	auth := NewAuth(*router,&c.db)
	router.POST("/login_attempt", auth.LoginAttempt)
	router.GET("/logout", auth.Logout)
	router.GET("/refresh", auth.Refresh)
	router.POST("/refresh", auth.Refresh)
	auth.GET("/","user", c.Main)
	auth.GET("/document/:doc","user",c.DocPage)
	auth.GET("/pdocument/:doc","head",c.DocPageProtected)
	auth.GET("/send_form","user",c.FormSend)
	auth.GET("/redirect_form","user",c.FormRedirect)
	auth.POST("/redirect","user",c.Redirect)
	auth.POST("/send","user",c.Send)
	auth.GET("/actual_docs","user",c.ActualDocs)
	auth.GET("/head_docs","head",c.HeadDocs)
	auth.GET("/head_archive","head",c.HeadArchive)
	auth.GET("/open","user",c.GetFile)
	auth.POST("/test","user", c.Test)
	auth.POST("/upload","user", c.UploadDoc)
	return *auth
}