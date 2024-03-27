package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"encoding/gob"

	//modulo locale
	"util"

	quic "github.com/quic-go/quic-go"
)

// Sinks
var dst []string

// rtt
var RTT []float64

// Average service time
var srv time.Duration

// Number of packets sent back
var pkt int64

// Counter
var cc, rr, ff int64

// For computing moving average
var stme = time.Time{}
var obswnw time.Duration = 10 * time.Second

const twnw = 60

var b_u util.Mavg = util.NewMavg(twnw)
var b_d util.Mavg = util.NewMavg(twnw)

var b_f util.Mavg = util.NewMavg(twnw)

// Access to shared counters
var mutex sync.Mutex
var ctl_id string = ""
var ctl_snt bool = false

// func InitRReply(workTime interface{}, dests interface{}, rtts interface{}, packets interface{}) {
func InitRReply(workTime interface{}, dests interface{}, packets interface{}) {
	// Initialize app configuration
	service, _ = time.ParseDuration(workTime.(string))
	pkts = packets.(int64)
	if dests != nil {
		fmt.Println(dests)
		ds := dests.([]interface{})
		fmt.Println("Using destinations:")
		for i, di := range ds {
			fmt.Printf("%d - %s", i, di)
			d = append(d, ds[i].(string))
		}
	}

	// Initialize counters
	cc = 0
	rr = 0
}

func RReplyBlock(req *util.RoPEMessage, stream quic.Stream) {
	req.Body = make([]byte, req.ResSize)
	req.Type = util.Response
	tmp := req.Source
	req.Source = req.Destination
	req.Destination = tmp
	ids := strings.Split(req.ReqID, ",")

	for i := 0; i < int(pkt); i++ {
		if i == 0 {
			var fwdctl bool
			mutex.Lock()
			fwdctl = ctl_snt
			ctl_snt = true
			mutex.Unlock()
			if !fwdctl {
				// add the id
				req.ReqID += fmt.Sprintf("%s,,%s,%d", ids[0], ctl_id, i)
			} else {
				// add an empty field
				req.ReqID += fmt.Sprintf("%s,,,%d", ids[0], i)
			}
		} else {
			req.ReqID = fmt.Sprintf("%s,,,%d", ids[0], i)
		}
		rr += int64(len(req.Body))
	}
	req.Type = util.QueueFull
	req.Body = make([]byte, 0)
	req.ReqID = fmt.Sprintf("%s,%s", ids[0], ids[1])
	req.Hop = req.Destination
	req.Destination = req.Source
	req.Source = req.Hop
	encoder := gob.NewEncoder(stream)
	err := encoder.Encode(req)
	util.LogEvent(req.ReqID, util.Sent, "Sending 'Queue full'", req.Log, idDevice)
	if err != nil {
		fmt.Printf("Error while sending 'Queue full' message: %s \n", err)
	}
}

func RReplyDecision(req *util.RoPEMessage, session *map[string]quic.EarlyConnection, i int64) bool {
	// Emulate processing time
	time.Sleep(srv)
	req.Body = make([]byte, req.ResSize)
	req.Type = util.Response
	tmp := req.Source
	req.Source = req.Destination
	req.Destination = tmp

	ids := strings.Split(req.ReqID, ",")
	if req.Destination == dst[0] {
		// pkt from the other server
		mutex.Lock()
		// store the pkt id for future use
		ctl_id = ids[1]
		ctl_snt = false
		mutex.Unlock()
		ff += int64(len(req.Body))
		// set pkt.Destination to an empty string so that no packet is sent back to the source
		req.Destination = ""
	} else {
		// pkt from the client
		var fwdctl bool
		// Has the id of the forwarded pkt already been sent?
		mutex.Lock()
		fwdctl = ctl_snt
		ctl_snt = true
		mutex.Unlock()
		if !fwdctl {
			// add the id
			req.ReqID += fmt.Sprintf("%s,%s,%d", req.ReqID, ctl_id, i)
		} else {
			// add an empty field
			req.ReqID += fmt.Sprintf("%s,,%d", req.ReqID, i)
		}
		if ids[0] == ids[1] {
			go func(s *map[string]quic.EarlyConnection, r *util.RoPEMessage) {
				// it is a control packet
				stream, err := (*s)[destinations[0]].OpenStream()
				if err != nil {
					panic(err)
				}
				// INVIA LA RICHIESTA
				encoder := gob.NewEncoder(stream)
				encoder.Encode(req)
				stream.Close()
			}(session, req)
		}

		cc += int64(len(req.Body))
		go func() {
			if (stme == time.Time{}) {
				stme = time.Now()
			} else {
				if time.Now().After(stme.Add(twnw)) {
					stime = time.Now()
					util.Mavg_push(&b_u, cc)
					util.Mavg_push(&b_d, rr)
					util.Mavg_push(&b_f, ff)

					cc = 0
					rr = 0
					ff = 0

					fmt.Printf("Uplink:  %f \n", util.Mavg_eval(b_u, int64(obswnw/time.Second)))
					fmt.Printf("Downlink:  %f \n", util.Mavg_eval(b_d, int64(obswnw/time.Second)))
					fmt.Printf("Relay:  %f \n", util.Mavg_eval(b_f, int64(obswnw/time.Second)))
				}
			}
		}()
	}
	var ret bool = false
	if i < pkt {
		ret = true
	}
	return ret
}

func RReplySetLastResponse(lastResp *util.RoPEMessage) {
	if lastResp.Type == util.Response {
		rr += int64(len(lastResp.Body))
	}
}
