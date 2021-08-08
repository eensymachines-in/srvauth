package main

import (
	"encoding/json"
	"net"

	core "github.com/eensymachines-in/lumincore"
	utl "github.com/eensymachines-in/utilities"
	log "github.com/sirupsen/logrus"
)

func main() {
	end := make(chan interface{}, 1)
	defer close(end)
	start, stop, err := utl.ListenOnUnixSocket("/tmp/eensy.srvauth.sock", func(c net.Conn) {
		buf := make([]byte, 512)
		nr, _ := c.Read(buf)
		data := buf[0:nr]
		Message := &core.SchedSockMessage{}
		json.Unmarshal(data, Message)
		log.WithFields(log.Fields{
			"serial": core.ISockMessage(Message).Serial(),
			"pass":   core.ISockMessage(Message).Pass(),
			"scheds": core.ISchedSockMsg(Message).JRStates(),
			"auth":   core.IAuthSockMsg(Message).IsAuthPass(),
			"reg":    core.IAuthSockMsg(Message).IsRegPass(),
		}).Info("We have received message on the socket")
		end <- struct{}{}
	})
	if err != nil {
		panic(err)
	}
	go start()
	defer stop()
	<-end // this will end this main function

	// program can end from within the socket listening handler
}
