package main

import (
	"fmt"
	"net"

	utl "github.com/eensymachines-in/utilities"
)

func main() {
	end := make(chan interface{}, 1)
	defer close(end)
	start, stop, err := utl.ListenOnUnixSocket("/var/local/eensymachines/sockets/halt.sock", func(c net.Conn) {
		result := []byte{}
		nr, _ := c.Read(result)
		data := result[0:nr]
		fmt.Println(string(data))
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
