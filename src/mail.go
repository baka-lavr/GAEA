package main

import (
	"fmt"
	"github.com/xhit/go-simple-mail/v2"
	"time"
	"log"
)

type MailConfig struct {
	host string
	port int
	name string
	pass string
	from string
}

type MailConnection struct {
	Client *mail.SMTPClient
	Config MailConfig
	Test int
}

func MailConnect(conf Configuration) (MailConnection, error) {
	server := mail.NewSMTPClient()
	conf2 := MailConfig{conf.MailHost, conf.MailPort, conf.MailUser, conf.MailPass, fmt.Sprintf("ООО СПАС Природа <%s>",conf.MailUser)}
	server.Host = conf2.host
	server.Port = conf2.port
	server.Username = conf2.name
	server.Password = conf2.pass
	server.Encryption = mail.EncryptionTLS
	server.KeepAlive = true
	server.ConnectTimeout = time.Second * 10
	server.SendTimeout = time.Second * 10
	client,err := server.Connect()
	conn := MailConnection{Client: client, Config: conf2, Test: 0}
	return conn,err
}

func (m *MailConnection) Reconnect() error {
	m.Client.Close()
	server := mail.NewSMTPClient()
	server.Host = m.Config.host
	server.Port = m.Config.port
	server.Username = m.Config.name
	server.Password = m.Config.pass
	server.Encryption = mail.EncryptionTLS
	server.KeepAlive = true
	server.ConnectTimeout = time.Second * 10
	server.SendTimeout = time.Second * 10
	client,_ := server.Connect()
	m.Client = client
	err := m.Client.Noop()
	if err != nil {
		log.Print(err)
		//logger.Error(err)
	}
	return err
}

func (m *MailConnection) Send(title,body string,paths []string, to []string) error {
	if m.Client.Noop() != nil {
		log.Print("Mail: trying to reconnect")
		//logger.Info("Mail: trying to reconnect")
		err := m.Reconnect()
		if err != nil {
			log.Print("Mail: cannot reconnect")
			//logger.Info("Mail: cannot reconnect")
			return err
		}
	}
	m.Test = m.Test + 1
	email := mail.NewMSG()
	email.SetFrom(m.Config.from).AddTo(to...).SetSubject(title)
	email.SetBody(mail.TextHTML, body)
	for _,j := range paths {
		file := &mail.File{FilePath: j}
		email.Attach(file)
	}
	if email.Error != nil {
		return email.Error
	}
	err := email.Send(m.Client)
	
	return err
}