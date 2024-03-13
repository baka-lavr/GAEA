package main

import (
	"github.com/baka-lavr/gaea/src/app"
	"os/signal"
	"syscall"
	"os"
)

//var logger service.Logger


func main() {
	c := &app.Controller{}
	_ = c.Start()
	quit := make(chan bool,1)
	sig := make(chan os.Signal,1)
	signal.Notify(sig,syscall.SIGINT,syscall.SIGTERM,syscall.SIGQUIT)
	go func() {
		<-sig
		quit<-true
	}()
	<-quit
	c.Stop()
}