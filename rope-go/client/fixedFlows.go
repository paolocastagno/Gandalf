package main

import (
	"fmt"
	"sync"
	"time"

	"util"
)

const twin = 60

var byte_up util.Mavg = util.NewMavg(twin)
var byte_down util.Mavg = util.NewMavg(twin)

// Access to shared counters
var mutex sync.Mutex
var cnt int64 = 0
var ctl int64 = 0
var ctl_id string = ""

func InitFFixed(destination interface{}, reqsize interface{}, ressize interface{}, ctlpktapart interface{}) {

	if destination == nil {
		die("No destination address specified")
	} else {
		destaddr = destination.(string)
	}
	msgsize = int32(reqsize.(int64))
	respsize = int32(ressize.(int64))
	ctl = ctlpktapart.(int64)

	fmt.Println("Loading logic fixed")
	fmt.Printf("\n\t- destination %s\n\t- request size %d bytes\n\t- response size %d bytes\n", destinations, msgsize, respsize)

}

func FFixedDecision(msg *util.RoPEMessage, destinations []string) string {
	msg.ResSize = respsize
	msg.Body = make([]byte, msgsize)
	if (st == time.Time{}) {
		st = time.Now()
	} else {
		if time.Now().After(st.Add(obswin)) {
			st = time.Now()

			util.Mavg_push(&b_up, snt)
			util.Mavg_push(&b_down, rcv)

			snt = 0
			rcv = 0

			fmt.Printf("Uplink:  %f \n", util.Mavg_eval(byte_up, int64(obswin/time.Second)))
			fmt.Printf("Downlink:  %f \n", util.Mavg_eval(byte_down, int64(obswin/time.Second)))
		}
	}

	mutex.Lock()
	if cnt%ctl == 0 {
		ctl_id = msg.ReqID
	} else {
		cnt++
	}
	mutex.Unlock()
	msg.ReqID += "," + ctl_id
	return destaddr
}

func FFixedSetLastResponse(lastResp util.RoPEMessage) {
	if lastResp.Type == util.Response {
		rcv += int64(lastResp.ResSize)
	}
}
