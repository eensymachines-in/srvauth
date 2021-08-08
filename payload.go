package main

/*Device will try to register to the lumin database.
This involves sending a payload of data to the api-database.
Payload data shape is determined by the api and such is populated by the device with the contextual data
Here various implementations of MakePayload can help you achieve open-for-extension-closed-for-modification */
import (
	"fmt"
	"os"
	"strings"
)

/*Registration for each of the specific apps could have their own way of making payloads
for luminapi here we need serial of the device against the relays it would operate*/
type MakePayload func() (interface{}, error)

// MakeRegPayload : makes a simple payload {serial:"", rlys:["","","",""]}
// this has no sophisticated definition for the relays as ids
func MakeRegPayload() (interface{}, error) {
	reg, err := getDeviceReg()
	if err != nil {
		return nil, fmt.Errorf("MakeRegPayload: failed %s", err)
	}
	/*Forming url for regitrations, we need the url and list of relay ids*/
	rlys := os.Getenv("RLYS")
	return struct {
		Serial   string   `json:"serial"`
		RelayIDs []string `json:"rlys"`
	}{
		Serial:   reg.Serial,
		RelayIDs: strings.Split(rlys, ","),
	}, nil
}

// MakeRegPayloadWithRlyDefn : before this we were sending relay ids only with the registration
// but now the definition of the relay goes alongside
// relay definition are user -readables alongside the relay id
// your machines still understand ids- humans dont
func MakeRegPayloadWithRlyDefn() (interface{}, error) {
	reg, err := getDeviceReg()
	if err != nil {
		return nil, fmt.Errorf("MakeRegPayload: failed %s", err)
	}
	// Visit the environment file for the variables
	rlys := strings.Split(os.Getenv("RLYS"), ",")
	defns := strings.Split(os.Getenv("RLYDFN"), ",")
	if len(rlys) == 0 {
		return nil, fmt.Errorf("MakeRegPayloadWithRlyDefn")
	}
	if len(defns) < len(rlys) {
		// Definitions cannot be less than relay ids
		return nil, fmt.Errorf("MakeRegPayloadWithRlyDefn: Number of relay definitions less than relays. Expecting %d values under RLYDFN in the environment", len(rlys))
	}
	pl := struct {
		Serial string              `json:"serial"`
		RMaps  []map[string]string `json:"rmaps"`
	}{Serial: reg.Serial}
	pl.RMaps = make([]map[string]string, len(rlys))
	for i, r := range rlys {
		pl.RMaps[i] = map[string]string{"rid": r, "defn": defns[i]}
	}
	return pl, nil
}
