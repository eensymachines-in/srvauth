package main

import (
	"encoding/json"
	"net"
	"os"

	log "github.com/sirupsen/logrus"
)

// haltService : drops the
func sendOverSock(m Message) {
	// NOTE: there are multiple sockets listed in haltsock
	// this uService is capable of communicating to multiple such uServices for halting action
	valEnv := os.Getenv("HALTSOCK")
	if valEnv == "" {
		log.WithFields(log.Fields{
			"haltsocket": valEnv,
		}).Error("Failed to connect to halt socket. Need location of atleast one socket")
		return
	}
	haltSock := valEnv
	/*We are trying out this phenomenon where multiple listeners are listening on the same halt socket
	This microservice shall push the message on the same socket*/
	c, err := net.Dial("unix", haltSock)
	if err != nil {
		// You need someone listening on the socket else this Dial action will fail
		log.WithFields(log.Fields{
			"err": err,
		}).Panic("Failed to connect to unix socket")
		return
	}
	// halt command is pushed to the socket, all the other microservices listening on the same socket will have to quit as well
	data, _ := json.Marshal(m)
	log.WithFields(log.Fields{
		"msg": string(data),
	}).Info("Now sending to socket")
	c.Write(data)
	// time to close this service
	return
}
