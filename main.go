package main

/*Microservices from the main line function will need a auxilliary microservice to verify registration and indicate accordingly
If the device is registered, this microservice will just drop off letting others in the main line cotinue, while if there is a problem with the registration it drops a signal on a socket to stop all main line operations. This service also can self-register the device but that incase only when the device serial is not blacklisted.
Incase the device serial is blacklisted, no registration and hence no further operations
This package here can authenticate the device with srvauth
See the Wiki documentation for more details*/
import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	auth "github.com/eensymachines-in/auth/v2"
	core "github.com/eensymachines-in/lumincore"
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
func RegisterDevice(makepl MakePayload, fail func(core.ISockMessage), success func(core.ISockMessage)) {
	// Getting the device registration
	// for the device registration we need the user details
	log.Info("Now trying to verify registration of the device with luminapi...")
	regUrl := os.Getenv("REGBASEURL")
	// Here if the registration url is not set, it would mean the client does not want any regisrations to be checked
	if regUrl == "" {
		success(nil)
		return
	}
	payload, err := makepl()
	if err != nil {
		fail(nil)
		return
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s", regUrl), bytes.NewBuffer(body))
	resp, err := (&http.Client{}).Do(req)

	if err != nil {
		log.WithFields(log.Fields{
			"reg_base_url": regUrl,
		}).Error("Failed to contact server for registration, device may have lost internet connection or the service on the cloud may not be running.")
		fail(nil)
		return
	}
	defer resp.Body.Close()
	var sockMsg core.ISockMessage
	body, err = ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		log.WithFields(log.Fields{
			"Reg_resp_status": resp.StatusCode,
			"body":            string(body),
		}).Error("Failed to register device with lumin server")
		sockMsg = &core.SockMessage{Reg: false}
		fail(sockMsg)
		return
	}
	// Incase the request succeeds the body of the response contains the default schedules
	// httpResult := map[string]interface{}{} // expected output data shape
	log.Info("Done registering the device withg luminapi")
	httpResult := &core.SchedSockMessage{SockMessage: &core.SockMessage{Reg: true}}
	if json.Unmarshal(body, httpResult) != nil {
		log.WithFields(log.Fields{
			"response_body": string(body),
		}).Error("RegisterDevice:failed to unmarshal response body")
	}
	success(httpResult)
	return
}

func AuthenticateDevice(uponFail func(core.ISockMessage), uponOk func(core.ISockMessage)) {
	log.Info("Now authenticating the device...")
	baseURL := os.Getenv("AUTHBASEURL")
	if baseURL == "" {
		// If the base url
		log.WithFields(log.Fields{
			"auth_base_url": baseURL,
		}).Error("AuthenticateDevice: Url for device authentication is invalid")
		uponFail(&core.SockMessage{SID: "", Auth: false}) //serial of the device is yet not read
	}
	_, err := http.Get(fmt.Sprintf("%s/ping", baseURL))
	if err != nil {
		log.Error("AuthenticateDevice/ping:Failed to ping uplink server, check internet connectivity for device")
		uponFail(&core.SockMessage{SID: "", Auth: false})
		return
	}
	reg, err := getDeviceReg()
	if err != nil {
		log.Errorf("AuthenticateDevice/getDeviceReg: failed to get device registration %s", err)
		uponFail(&core.SockMessage{SID: "", Auth: false})
		return
	}
	// From here on we have the serial of the device
	status := &auth.DeviceStatus{}
	if auth.DeviceStatusOnCloud(fmt.Sprintf("%s/devices/%s", baseURL, reg.Serial), status) != nil {
		log.Error("AuthenticateDevice/DeviceStatusOnCloud:Failed to query device status on cloud, servers are unreachable or busy")
		uponFail(&core.SockMessage{SID: reg.Serial, Auth: false})
		return
	}
	if (auth.DeviceStatus{}) == *status {
		log.Warn("Device is not registered on the cloud, Now attempting to register this device on the cloud")
		if err := reg.Register(fmt.Sprintf("%s/devices", baseURL)); err != nil {
			log.Errorf("Failed to register device %s", err)
			uponFail(&core.SockMessage{SID: reg.Serial, Auth: false})
			return
		}
		// If the registration was success, then no need to continue further steps
		uponOk(&core.SockMessage{SID: reg.Serial, Auth: true})
		return
	}
	if status.Lock {
		log.Error("Device is locked by the admin, cannot continue. Please contact an admin to unlock the device")
		uponFail(&core.SockMessage{SID: reg.Serial, Auth: false})
		return
	}
	if status.User != reg.User {
		log.Error("Device ownership is invalid, cannot continue. Please contact an admin to reassign the device to a valid account")
		uponFail(&core.SockMessage{SID: reg.Serial, Auth: false})
		return
	}
	log.WithFields(log.Fields{
		"serial": status.Serial,
		"user":   status.User,
		"lock":   status.Lock,
	}).Info("Device authenticated")
	uponOk(&core.SockMessage{SID: reg.Serial, Auth: true})
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
	AuthenticateDevice(func(m core.ISockMessage) {
		// When authentication fails - will not proceed for any registration check
		sendOverSock(m)
		return
	}, func(authM core.ISockMessage) {
		// When authentication succeeds
		// we can proceed for registration check
		RegisterDevice(MakeRegPayloadWithRlyDefn, func(m core.ISockMessage) {
			// TODO: authM auth status should be merged with m here so that we have the complete status
			// registration has failed
			m.(core.IAuthSockMsg).SetAuth(authM.(core.IAuthSockMsg).IsAuthPass())
			sendOverSock(m)
			return
		}, func(m core.ISockMessage) {
			// TODO: authM auth status should be merged with m here so that we have the complete status
			// On success of the registration this will send along the default schedules received from the api in the http reponse
			// result : map[string]interface{} is the type we receive as http response body
			// here no need for type conversion since we just want to dispatch it via socket
			// on the receiving side though this message shall be interpretted as and the scheds need to be read back as JRS
			m.(core.IAuthSockMsg).SetAuth(authM.(core.IAuthSockMsg).IsAuthPass())
			sendOverSock(m)
			return
		})
	})
}
