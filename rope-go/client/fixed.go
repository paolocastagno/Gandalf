package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"
	"util"
)

var fnm string

func InitFixed(destination interface{}, reqsize interface{}, ressize interface{}, appto interface{}, binsze interface{}) {

	if destination == nil {
		die("No destination address specified")
	} else {
		destaddr = destination.(string)
	}
	msgsize = int32(reqsize.(int64))
	respsize = int32(ressize.(int64))
	fmt.Println("Loading logic fixed")
	fmt.Printf("\n\t- destination %s\n\t- request size %d bytes\n\t- response size %d bytes\n", destinations, msgsize, respsize)

	bsze, _ := time.ParseDuration(binsze.(string))
	binsize = float64(bsze) / float64(time.Millisecond)

	var s stats = stats{
		alpha: 1.0,
		snt:   0,
		rcv:   0,
		succ:  0}
	statistics = append(statistics, s)

	app_timeout, _ = time.ParseDuration(appto.(string))

}

func FixedDecision(msg *util.RoPEMessage, destinations []string) string {

	msg.ResSize = respsize
	t := time.Now()

	statistics[len(statistics)-1].snt++

	encoding_exp := make([]byte, 4)
	binary.LittleEndian.PutUint32(encoding_exp, uint32(len(statistics)))
	// Calling MarshalBinary() method
	encoding_tmestp, _ := t.MarshalBinary()
	// Write to bytearray
	body := make([]byte, int(msgsize)-len(encoding_tmestp)-len(encoding_exp))
	msg.Body = append(encoding_tmestp, append(encoding_exp, body...)...)

	return destaddr
}

func FixedSetLastResponse(lastResp util.RoPEMessage) {
	if lastResp.Type == util.Response {
		// received += int64(lastResp.ResSize)
		var tstmp = time.Now()
		// Get the experiment whose the request belongs to
		exp := int(binary.LittleEndian.Uint32(lastResp.Body[15:20]))
		// Count the message as received
		statistics[exp-1].rcv += 1
		var u time.Time
		u.UnmarshalBinary(lastResp.Body[:15])
		duration := tstmp.Sub(u)
		// if u.Add(timeout).After(tstmp) {
		statistics[exp-1].sum += float64(duration.Milliseconds())
		statistics[exp-1].ssum += math.Pow(float64(duration.Milliseconds()), 2)
		if duration <= app_timeout {
			statistics[exp-1].succ += 1
		}
		duration = duration.Truncate(time.Millisecond)
		idx := int(math.Floor(float64(duration.Milliseconds()) / binsize))
		if len(hist) < int(idx) {
			hist = append(hist, make([]bin, idx-len(hist))...)
		}
		hist[idx-1].count++
		hist[idx-1].value = float64(duration.Milliseconds())
	}
}
