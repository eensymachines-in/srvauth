package main

/*Microservices from the main line function will need a auxilliary microservice to verify registration and indicate accordingly
If the device is registered, this microservice will just drop off letting others in the main line cotinue, while if there is a problem with the registration it drops a signal on a socket to stop all main line operations. This service also can self-register the device but that incase only when the device serial is not blacklisted.
Incase the device serial is blacklisted, no registration and hence no further operation s*/
import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	auth "github.com/eensymachines-in/auth/v2"
	"github.com/eensymachines-in/luminapi/core"
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
)

func init() {
	utl.SetUpLog()
	flag.BoolVar(&Flog, "flog", true, "direction of log messages, set false for terminal logging. Default is true")
	flag.BoolVar(&FVerbose, "verbose", false, "Determines what level of log messages are to be output, Default is info level")
}

// Its this message that this microservice will shuttle thru the socket
type Message struct {
	Auth bool `json:"auth"`
	Reg  bool `json:"reg"`
}

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
	c.Write(data)
	// time to close this service
	return
}

// Gets the device to register if not already, Send in the url and the relay ids
func RegisterDevice(fail func(), success func()) {
	// Getting the device registration
	// for the device registration we need the user details
	user := os.Getenv("USER")
	if user == "" {
		log.WithFields(log.Fields{
			"user": user,
		}).Error("Owner ID for the device invalid")
		fail()
	}
	reg, err := auth.ThisDeviceReg(user)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error reading device registration")
		fail()
		return
	}
	/*Forming url for regitrations, we need the url and list of relay ids*/
	regUrl := os.Getenv("REGBASEURL")
	rlys := os.Getenv("RLYS")
	payload := &core.DevRegHttpPayload{
		Serial:   reg.Serial,
		RelayIDs: strings.Split(rlys, ","),
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/", regUrl), bytes.NewBuffer(body))
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"reg_base_url": regUrl,
		}).Error("Failed to contact server for registration, device may have lost internet connection")
	}
	if resp.StatusCode != 200 {
		fail()
		return
	}
	success()
	return
}

func AuthenticateDevice(uponFail func(), uponOk func()) {
	baseURL := os.Getenv("AUTHBASEURL")
	if baseURL == "" {
		// If the base url
		log.WithFields(log.Fields{
			"auth_base_url": baseURL,
		}).Error("Url for device authentication is invalid")
		uponFail()
	}
	_, err := http.Get(fmt.Sprintf("%s/ping", baseURL))
	if err != nil {
		log.Error("The device needs to be online on bootup, We tried pinging the uplink servers, could not reach. Check your WiFi and internet connectivity")
		uponFail()
		return
	}
	user := os.Getenv("USER")
	if user == "" {
		log.WithFields(log.Fields{
			"user": user,
		}).Error("Owner ID for the device invalid")
		uponFail()
	}
	reg, err := auth.ThisDeviceReg(user)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error reading device registration")
		uponFail()
		return
	}
	status := &auth.DeviceStatus{}
	if auth.DeviceStatusOnCloud(fmt.Sprintf("%s/devices/%s", baseURL, reg.Serial), status) != nil {
		log.Error("Failed to query device status on cloud, servers are unreachable or busy")
		uponFail()
		return
	}
	if (auth.DeviceStatus{}) == *status {
		log.Warn("Device is not registered on the cloud")
		log.Info("Now attempting to register this device on the cloud")
		if err := reg.Register(fmt.Sprintf("%s/devices", baseURL)); err != nil {
			log.Errorf("Failed to register device %s", err)
			uponFail()
			return
		}
		return
	}
	if status.Lock {
		log.Error("Device is locked by the admin, cannot continue. Please contact an admin to unlock the device")
		uponFail()
		return
	}
	if status.User != user {
		log.Error("Device ownership is invalid, cannot continue. Please contact an admin to reassign the device to a valid account")
		uponFail()
		return
	}
	uponOk()
	return
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
	AuthenticateDevice(func() {
		// When authentication fails
		sendOverSock(Message{Auth: false, Reg: false})
		return
	}, func() {
		// When authentication succeeds
		// we can proceed for registration check
		RegisterDevice(func() {
			// registration has failed
			sendOverSock(Message{Auth: true, Reg: false})
			return
		}, func() {
			sendOverSock(Message{Auth: true, Reg: true})
			return
		})
	})
}
