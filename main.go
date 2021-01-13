package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/eensymachines-in/auth"
	utl "github.com/eensymachines-in/utilities"
	log "github.com/sirupsen/logrus"
)

var (
	// this file location is created if not found
	logFile = "/var/local/srvauth/device.log"
	// Flog : determines if the direction of log output
	Flog bool
	// FVerbose :  determines the level of log
	FVerbose bool
	// owner of the device, this should come from the environment loaded on the container
	user     = "kneeru@gmail.com"
	baseURL  = "http://localhost:8080/"
	haltSock = "/var/local/srvauth/halt.sock"
)

func init() {
	utl.SetUpLog()
	flag.BoolVar(&Flog, "flog", true, "direction of log messages, set false for terminal logging. Default is true")
	flag.BoolVar(&FVerbose, "verbose", false, "Determines what level of log messages are to be output, Default is info level")
}

// haltService : drops the
func haltService() {
	// this can help send a message to the socket
	c, err := net.Dial("unix", haltSock)
	if err != nil {
		log.Errorf("Failed to halt authentication service, %s", err)
		return
	}
	// halt command is pushed to the socket, all the other microservices listening on the same socket will have to quit as well
	data, _ := json.Marshal(map[string]bool{"interrupt": true})
	c.Write(data)
	<-time.After(1 * time.Second) // let the command assimilate in the sock
	// time to close this service
	return
}

func main() {
	log.Info("SrvAuth: initializing...")
	defer log.Warn("SrvAuth: now closing service..")

	// Here we read all the environment variables
	valEnv := os.Getenv("LOGF")
	if valEnv != "" {
		logFile = valEnv
	}
	valEnv = os.Getenv("baseURL")
	if valEnv != "" {
		baseURL = valEnv
	}
	valEnv = os.Getenv("HALTSOCK")
	if valEnv != "" {
		haltSock = valEnv
	}
	// Log direction and the level of logging
	flag.Parse()
	closeLogFile := utl.CustomLog(Flog, FVerbose, logFile)
	defer closeLogFile()

	// Lsitening on system signals
	// start, interrupt := utl.SysSignalListener()
	// go start()

	reg, err := auth.ThisDeviceReg(user)
	if err != nil {
		log.Errorf("Failed to read local device registration details")
		haltService()
		return
	}
	regurl := fmt.Sprintf("%sdevices/%s", baseURL, reg.Serial)
	ok, err := auth.IsRegistered(regurl)
	if err != nil {
		log.Errorf("Failed to verify if device is registered %s", err)
		haltService()
		return
	}
	if !ok {
		// device is not registered, so device will register itself
		if reg.Register(fmt.Sprintf("%s/devices/", baseURL)) != nil {
			log.Errorf("Failed to register device %s", err)
			haltService()
			return
		}
		log.Info("Device successfully registered itself")
		return
	} else {
		// device is already registered
		// check for ownership and the lock status
		owned, err := auth.IsOwnedBy(regurl, user)
		locked, err := auth.IsLocked(regurl)
		if err != nil {
			log.Errorf("Failed to verify device details %s", err)
			haltService()
			return
		}
		if !owned || locked {
			log.Errorf("Device ownership invalid, or the device is locked. %s", err)
			haltService()
			return
		} //else the service just exists, and let other microservices continue
	}

}
