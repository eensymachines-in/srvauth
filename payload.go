package main

import (
	"fmt"
	"os"
	"strings"
)

/*Registration for each of the specific apps could have their own way of making payloads
for luminapi here we need serial of the device against the relays it would operate*/
type MakePayload func() (interface{}, error)

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
