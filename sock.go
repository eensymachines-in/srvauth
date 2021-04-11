package main

import (
	"encoding/json"
	"net"
	"os"
	"strings"

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
		data, _ := json.Marshal(m)
		log.WithFields(log.Fields{
			"msg": string(data),
		}).Info("Now sending to socket")
		c.Write(data)
	}
	return
}
