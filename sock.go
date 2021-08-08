package main

import (
	"encoding/json"
	"net"
	"os"
	"strings"

	core "github.com/eensymachines-in/lumincore"
	log "github.com/sirupsen/logrus"
)

// sendOverSock : sends the message over unix socket
// Socket location is found loaded on the environment
// this can send a message to multiple sockets
// message is marshalled to json before dispatch
func sendOverSock(m core.ISockMessage) {
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
	socks := strings.Split(haltSock, ",")
	// https://unix.stackexchange.com/questions/615330/what-happens-when-two-processes-listen-on-the-same-berkeley-unix-file-socket
	// since we learn that for tcp unix sockets its not possible to have 2 simulteneous clients listening on the same socket
	// we resort to sending to multiple clients via idependent sockets
	for _, s := range socks {
		c, err := net.Dial("unix", s)
		if err != nil {
			// You need someone listening on the socket else this Dial action will fail
			log.WithFields(log.Fields{
				"err": err,
			}).Error("Failed to connect to unix socket, Check if the socket has been started")
			continue
		}
		// halt command is pushed to the socket, all the other microservices listening on the same socket will have to quit as well
		data, err := json.Marshal(m)
		if err != nil {
			// Json marshalling has failed - this is unlikely
			log.WithFields(log.Fields{
				"msg": m,
			}).Infof("sendOverSock/json.Marshal(m): Error marshalling message to json %s", err)
			return
		}
		_, err = c.Write(data)
		if err != nil {
			log.WithFields(log.Fields{
				"connection": c,
				"data":       data,
			}).Infof("sendOverSock/c.Write(data): Failed to write data to socket %s", err)
			return
		}
	}
	return
}
