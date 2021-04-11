package main

/*Microservices from the main line function will need a auxilliary microservice to verify registration and indicate accordingly
If the device is registered, this microservice will just drop off letting others in the main line cotinue, while if there is a problem with the registration it drops a signal on a socket to stop all main line operations. This service also can self-register the device but that incase only when the device serial is not blacklisted.
Incase the device serial is blacklisted, no registration and hence no further operations
This package here can authenticate the device with srvauth */
import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

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

// Gets the host device registration details using the user id loaded from environment
func getDeviceReg() (*auth.DeviceReg, error) {
	user := os.Getenv("USER")
	if user == "" {
		log.WithFields(log.Fields{
			"user": user,
		}).Error("Owner ID for the device invalid")
		return nil, fmt.Errorf("getDeviceReg: failed")
	}
	reg, err := auth.ThisDeviceReg(user)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Error reading device registration")
		return nil, fmt.Errorf("getDeviceReg: failed")
	}
	return reg, nil
}

// Gets the device to register if not already, Send in the url and the relay ids
func RegisterDevice(makepl MakePayload, fail func(), success func()) {
	// Getting the device registration
	// for the device registration we need the user details
	regUrl := os.Getenv("REGBASEURL")
	payload, err := makepl()
	if err != nil {
		fail()
		return
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/", regUrl), bytes.NewBuffer(body))
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"reg_base_url": regUrl,
		}).Error("Failed to contact server for registration, device may have lost internet connection")
		fail()
		return
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
	reg, err := getDeviceReg()
	if err != nil {
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
	if status.User != reg.User {
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
		RegisterDevice(MakeRegPayload, func() {
			// registration has failed
			sendOverSock(Message{Auth: true, Reg: false})
			return
		}, func() {
			sendOverSock(Message{Auth: true, Reg: true})
			return
		})
	})
}
