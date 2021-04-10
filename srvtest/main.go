package main

import (
	"encoding/json"
	"fmt"
	"net"

	utl "github.com/eensymachines-in/utilities"
)

func main() {
	end := make(chan interface{}, 1)
	defer close(end)
	start, stop, err := utl.ListenOnUnixSocket("/var/local/eensymachines/sockets/halt.sock", func(c net.Conn) {
		buf := make([]byte, 512)
		nr, _ := c.Read(buf)
		data := buf[0:nr]
		Message := struct {
			Auth bool `json:"auth"`
			Reg  bool `json:"reg"`
		}{}
		json.Unmarshal(data, &Message)
		fmt.Println(Message)
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
