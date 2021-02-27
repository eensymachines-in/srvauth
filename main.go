package main

/*Microservices from the main line function will need a auxilliary microservice to verify registration and indicate accordingly
If the device is registered, this microservice will just drop off letting others in the main line cotinue, while if there is a problem with the registration it drops a signal on a socket to stop all main line operations. This service also can self-register the device but that incase only when the device serial is not blacklisted.
Incase the device serial is blacklisted, no registration and hence no further operation s*/
import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	auth "github.com/eensymachines-in/auth/v2"
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
	user     = ""
	baseURL  = ""
	haltSock = ""
)

func init() {
	utl.SetUpLog()
	flag.BoolVar(&Flog, "flog", true, "direction of log messages, set false for terminal logging. Default is true")
	flag.BoolVar(&FVerbose, "verbose", false, "Determines what level of log messages are to be output, Default is info level")
}

// haltService : drops the
func haltService() {
	// NOTE: there are multiple sockets listed in haltsock
	// this uService is capable of communicating to multiple such uServices for halting action
	socks := strings.Split(haltSock, ",")
	for _, sock := range socks {
		// this can help send a message to the socket
		c, err := net.Dial("unix", sock)
		if err != nil {
			// You need someone listening on the socket else this Dial action will fail
			log.Errorf("srvauth: Failed to connect socket, cannot halt service, %s", err)
			return
		}
		// halt command is pushed to the socket, all the other microservices listening on the same socket will have to quit as well
		data, _ := json.Marshal(map[string]int{"interrupt": 1})
		c.Write(data)
	}
	// time to close this service
	return
}

func isOnline() bool {
	_, err := http.Get(fmt.Sprintf("%s/ping", baseURL))
	if err != nil {
		return false
	}
	return true
}

func main() {
	log.Info("srvauth: initializing...")
	defer log.Warn("srvauth: now closing service..")

	// ++++++++++++ Reading environment variables
	valEnv := os.Getenv("LOGF")
	if valEnv != "" {
		logFile = valEnv
	}
	flag.Parse()
	closeLogFile := utl.CustomLog(Flog, FVerbose, logFile) // Log direction and the level of logging
	defer closeLogFile()

	valEnv = os.Getenv("BASEURL")
	if valEnv != "" {
		baseURL = valEnv
	}
	valEnv = os.Getenv("HALTSOCK")
	if valEnv != "" {
		haltSock = valEnv
	}
	valEnv = os.Getenv("USER")
	if valEnv != "" {
		user = valEnv
	}
	log.WithFields(log.Fields{
		"logfile":  logFile,
		"baseurl":  baseURL,
		"haltsock": haltSock,
		"user":     user,
	}).Debug("SrvAuth: now read all the environment")
	if haltSock == "" {
		panic("srvauth: invalid halt-socket to communicate to, cannot continue")
	}
	if user == "" || baseURL == "" {
		haltService()
		panic("srvauth: Invalid owner email or the uplink base url")
	}
	// Lsitening on system signals
	// start, interrupt := utl.SysSignalListener()
	// go start()
	reg, err := auth.ThisDeviceReg(user)
	if err != nil {
		log.Errorf("Failed to read local device registration details")
		haltService()
		return
	}
	log.WithFields(log.Fields{
		"reg": reg,
	}).Debug("srvauth: Local device registration details")
	// We check for internet connectivity before making calls to the authorization service
	if !isOnline() {
		log.Error("The device needs to be online on bootup, We tried pinging the uplink servers, could not reach. Check your WiFi and internet connectivity")
		haltService()
		return
	}
	regurl := fmt.Sprintf("%s/devices/%s", baseURL, reg.Serial)
	var status *auth.DeviceStatus
	if auth.DeviceStatusOnCloud(regurl, status) != nil {
		log.Error("Failed to query device status on cloud, servers are unreachable or busy")
		haltService()
		return
	}
	if status == nil {
		log.Warn("Device is not registered on the cloud")
		log.Info("Now attempting to register this device on the cloud")
		if err := reg.Register(fmt.Sprintf("%s/devices", baseURL)); err != nil {
			log.Errorf("Failed to register device %s", err)
			haltService()
			return
		}
		return
	}
	if status.Lock {
		log.Error("Device is locked by the admin, cannot continue. Please contact an admin to unlock the device")
		haltService()
		return
	}
	if status.User != user {
		log.Error("Device ownership is invalid, cannot continue. Please contact an admin to reassign the device to a valid account")
		haltService()
		return
	} // device is registered and everything is a green signal
	// the container from here on just exits & lets the services continue
}
