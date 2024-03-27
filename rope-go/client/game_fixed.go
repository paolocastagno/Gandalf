package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	"util"
)

type fixedGameRound struct {
	round time.Duration
	begin time.Time
	end   time.Time
}

type fixedGame struct {
	pl_id int64
	alpha float64
}

// Configuration

var fGmeCfg fixedGame = fixedGame{
	pl_id: 0,
	alpha: 0.5}

var fGmeRnd fixedGameRound = fixedGameRound{
	round: 0,
	begin: time.Now(),
	end:   time.Now()}

func InitFixedGame(plid interface{}, rnd_dtn interface{}, destinations interface{}, msize interface{}, appto interface{}, alpha interface{}, binsze interface{}) {

	if destinations == nil {
		die("No destination address specified")
	}

	destaddr := destinations.([]interface{})

	// Setup game configuration
	fGmeCfg.alpha = alpha.(float64)
	fGmeCfg.pl_id = plid.(int64)

	bsze, _ := time.ParseDuration(binsze.(string))
	binsize = float64(bsze) / float64(time.Millisecond)

	app_timeout, _ = time.ParseDuration(appto.(string))

	var s stats = stats{
		alpha: fGmeCfg.alpha,
		snt:   0,
		rcv:   0,
		succ:  0}
	statistics = append(statistics, s)

	// Setup round configuration
	var rnd_str string = rnd_dtn.(string)
	fGmeRnd.round, _ = time.ParseDuration((rnd_str))

	for i, d := range destaddr {
		fmt.Printf("%d:\t %s", i, d)
		// Initialize probabilities and destinations
		daddr = append(daddr, d.(string))
	}

	obwindow = 10 * time.Second
	btime = time.Now().Add(obwindow)

	msgsize = int32(msize.(int64))
	fmt.Println("Loading logic game")
	fmt.Printf("\n\t- destinations %s\n\t- request size %d bytes\n\t- response size %d bytes\n", destinations, msgsize, msgsize)
}

func FixedGameDecision(msg *util.RoPEMessage, destinations []string) string {
	if fGmeRnd.begin == fGmeRnd.end {
		fGmeRnd.begin = time.Now()
		fGmeRnd.end = fGmeRnd.begin.Add(fGmeRnd.round)
		fmt.Println("Setting up first round:\n\tbegin: ", fGmeRnd.begin, "\t end: ", fGmeRnd.end, "\n\talpha: ", fGmeCfg.alpha)
	} else {
		if time.Now().After(fGmeRnd.end) {
			fGmeRnd.begin = time.Now()
			fGmeRnd.end = fGmeRnd.begin.Add(fGmeRnd.round)
			var s stats = stats{
				alpha: fGmeCfg.alpha,
				snt:   0,
				rcv:   0,
				succ:  0}
			write_stats(fmt.Sprintf("/res/player_%d", fGmeCfg.pl_id), os.O_CREATE|os.O_WRONLY)
			write_hist(fmt.Sprintf("/res/player_%d_%d", fGmeCfg.pl_id, len(statistics)), os.O_CREATE|os.O_WRONLY)
			write_histograms(rtts, fmt.Sprintf("/res/player_%d__rtt_%d", fGmeCfg.pl_id, len(statistics)), os.O_CREATE|os.O_WRONLY)
			write_histograms(services, fmt.Sprintf("/res/player_%d__srv_%d", fGmeCfg.pl_id, len(statistics)), os.O_CREATE|os.O_WRONLY)
			statistics = append(statistics, s)
			hist = []bin{}
			rtts = [][]bin{{}, {}}
			services = [][]bin{{}, {}}
			fmt.Println("Setting up next round:\n\tbegin: ", fGmeRnd.begin, "\t end: ", fGmeRnd.end, "\n\talpha: ", fGmeCfg.alpha)
		}
	}
	p := rand.Float64()
	var i int = 0
	if p > fGmeCfg.alpha {
		i = 1
	}

	msg.Log = false
	msg.Source = fmt.Sprintf("player_%d", fGmeCfg.pl_id)
	msg.Destination = destinations[i]
	msg.ResSize = msgsize
	t := time.Now()

	statistics[len(statistics)-1].snt++

	encoding_exp := make([]byte, 4)
	binary.LittleEndian.PutUint32(encoding_exp, uint32(len(statistics)))
	// Calling MarshalBinary() method
	encoding_tmestp, _ := t.MarshalBinary()
	// Write to bytearray
	body := make([]byte, int(msgsize)-len(encoding_tmestp)-len(encoding_exp))
	msg.Body = append(encoding_tmestp, append(encoding_exp, body...)...)

	return destinations[i]
}

func FixedGameSetLastResponse(lastResp util.RoPEMessage) {
	if lastResp.Type == util.Response {
		var i int = 0
		for i < len(daddr) && daddr[i] != lastResp.Source {
			i++
		}
		var tstmp = time.Now()
		// Get the experiment whose the request belongs to
		exp := int(binary.LittleEndian.Uint32(lastResp.Body[15:20]))
		// Count the message as received
		statistics[exp-1].rcv += 1
		var u time.Time
		u.UnmarshalBinary(lastResp.Body[:15])
		duration := tstmp.Sub(u)
		statistics[exp-1].sum += float64(duration.Milliseconds())
		statistics[exp-1].ssum += math.Pow(float64(duration.Milliseconds()), 2)
		if duration <= app_timeout {
			statistics[exp-1].succ += 1
		}
		duration = duration.Truncate(time.Millisecond)
		idx := int(math.Floor(float64(duration.Milliseconds()) / binsize))
		if len(hist) <= int(idx) {
			hist = append(hist, make([]bin, 1+idx-len(hist))...)
		}
		hist[idx].count++
		hist[idx].value = float64(duration.Milliseconds())

		var arr, dep time.Time
		arr.UnmarshalBinary(lastResp.Body[(len(lastResp.Body) - 30):(len(lastResp.Body) - 15)])
		dep.UnmarshalBinary(lastResp.Body[(len(lastResp.Body) - 15):(len(lastResp.Body))])
		var service time.Duration = (dep.Sub(arr)).Truncate(time.Millisecond)
		duration = duration - service
		idx = int(math.Floor(float64(duration.Milliseconds()) / binsize))
		var src int
		var d string
		for src, d = range destinations {
			if d == lastResp.Source {
				break
			}
		}
		if len(rtts[src]) <= int(idx) {
			rtts[src] = append(rtts[src], make([]bin, 1+idx-len(rtts[src]))...)
		}
		rtts[src][idx].count++
		rtts[src][idx].value = float64(duration.Milliseconds())

		idx = int(math.Floor(float64(service.Milliseconds()) / binsize))
		if len(services[src]) <= int(idx) {
			services[src] = append(services[src], make([]bin, 1+idx-len(services[src]))...)
		}
		services[src][idx].count++
		services[src][idx].value = float64(service.Milliseconds())

		if btime.Before(time.Now()) {
			btime = time.Now().Add(obwindow)
			duration = tstmp.Sub(fGmeRnd.begin)
			fmt.Printf("Alpha:%.6f\tUplink (req/s):%.6f\tDownlink (req/s):%.6f\n",
				statistics[len(statistics)-1].alpha,
				float64(statistics[len(statistics)-1].snt)/float64(duration.Seconds()),
				float64(statistics[len(statistics)-1].rcv)/float64(duration.Seconds()))
		}
	}
}
