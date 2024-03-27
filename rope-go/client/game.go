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

var corr float64 = 1

type gameRound struct {
	round time.Duration
	begin time.Time
	end   time.Time
}

type game struct {
	pl_id          int64
	alpha_step     float64
	d_fact         float64
	m_fact         float64
	min_step_alpha float64
	max_step_alpha float64
	alpha          float64
}

// Configuration

var gmeCfg game = game{
	pl_id:          0,
	alpha_step:     0.01,
	d_fact:         2,
	m_fact:         1.1,
	min_step_alpha: 1e-6,
	max_step_alpha: 0.1,
	alpha:          0.5}

var gmeRnd gameRound = gameRound{
	round: 0,
	begin: time.Now(),
	end:   time.Now()}

func InitGame(plid interface{}, rnd_dtn interface{}, destinations interface{}, msize interface{}, appto interface{}, alpha interface{}, step interface{}, minstep interface{}, maxstep interface{}, dfact interface{}, mfact interface{}, binsze interface{}) {

	if destinations == nil {
		die("No destination address specified")
	}

	destaddr := destinations.([]interface{})

	// Setup game configuration
	gmeCfg.pl_id = plid.(int64)
	gmeCfg.alpha_step = step.(float64)
	gmeCfg.d_fact = dfact.(float64)
	gmeCfg.m_fact = mfact.(float64)
	gmeCfg.min_step_alpha = minstep.(float64)
	gmeCfg.max_step_alpha = maxstep.(float64)
	gmeCfg.alpha = alpha.(float64)

	bsze, _ := time.ParseDuration(binsze.(string))
	binsize = float64(bsze) / float64(time.Millisecond)

	app_timeout, _ = time.ParseDuration(appto.(string))

	var s stats = stats{
		alpha: gmeCfg.alpha,
		snt:   0,
		rcv:   0,
		succ:  0}
	statistics = append(statistics, s)

	//Setup round configuration
	var rnd_str string = rnd_dtn.(string)
	gmeRnd.round, _ = time.ParseDuration(rnd_str)

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

func next_alpha() float64 {
	var round int = len(statistics)
	var current stats = statistics[round-1]
	var nextalpha float64
	nextalpha = current.alpha
	if round%2+1 == int(gmeCfg.pl_id) {
		if round <= 2 {
			nextalpha = nextalpha + gmeCfg.alpha_step
		} else {
			var previous stats = statistics[round-3]
			var delta_pf float64 = float64(previous.succ)/float64(previous.snt) - float64(current.succ)/float64(current.snt)
			if (float64(current.succ)/float64(current.snt))/(float64(previous.succ)/float64(previous.snt)) > 1e-2 {
				var sign_pf, sign_a float64 = 1.0, 1.0
				if delta_pf < 0 {
					sign_pf = -1.0
					corr = corr * gmeCfg.m_fact
				} else {
					corr = corr / gmeCfg.d_fact
				}
				if current.alpha < previous.alpha {
					sign_a = -1.0
				}
				var next_step float64 = sign_pf * sign_a * corr * gmeCfg.alpha_step
				if math.Abs(next_step) < gmeCfg.min_step_alpha {
					if next_step > 0 {
						next_step = gmeCfg.min_step_alpha
					} else {
						next_step = gmeCfg.min_step_alpha * (-1.0)
					}
				} else {
					if math.Abs(next_step) > gmeCfg.max_step_alpha {
						if next_step > 0 {
							next_step = gmeCfg.max_step_alpha
						} else {
							next_step = gmeCfg.max_step_alpha * (-1.0)
						}
					}
				}
				nextalpha = nextalpha - next_step
				if nextalpha >= 1.0 {
					nextalpha = 1.0 - gmeCfg.min_step_alpha
				} else {
					if nextalpha <= 0.0 {
						nextalpha = gmeCfg.min_step_alpha
					}
				}
			}
		}
	}
	return nextalpha
}

func GameDecision(msg *util.RoPEMessage, destinations []string) string {
	if gmeRnd.begin == gmeRnd.end {
		gmeRnd.begin = time.Now()
		gmeRnd.end = gmeRnd.begin.Add(gmeRnd.round)
		fmt.Println("Setting up first round:\n\tbegin: ", fGmeRnd.begin, "\t end: ", fGmeRnd.end, "\n\talpha: ", fGmeCfg.alpha)
	} else {
		if time.Now().After(gmeRnd.end) {
			gmeRnd.begin = time.Now()
			gmeRnd.end = gmeRnd.begin.Add(gmeRnd.round)
			gmeCfg.alpha = next_alpha()
			var s stats = stats{
				alpha: gmeCfg.alpha,
				snt:   0,
				rcv:   0,
				succ:  0}
			write_stats(fmt.Sprintf("/res/player_%d", gmeCfg.pl_id), os.O_CREATE|os.O_WRONLY)
			write_hist(fmt.Sprintf("/res/player_%d_%d", gmeCfg.pl_id, len(statistics)), os.O_CREATE|os.O_WRONLY)
			write_histograms(rtts, fmt.Sprintf("/res/player_%d__rtt_%d", gmeCfg.pl_id, len(statistics)), os.O_CREATE|os.O_WRONLY)
			write_histograms(services, fmt.Sprintf("/res/player_%d__srv_%d", gmeCfg.pl_id, len(statistics)), os.O_CREATE|os.O_WRONLY)
			statistics = append(statistics, s)
			hist = []bin{}
			rtts = [][]bin{{}, {}}
			services = [][]bin{{}, {}}
			fmt.Println("Setting up next round:\n\tbegin: ", fGmeRnd.begin, "\t end: ", fGmeRnd.end, "\n\talpha: ", fGmeCfg.alpha)
		}
	}
	p := rand.Float64()
	var i int = 0
	if p > gmeCfg.alpha {
		i = 1
	}

	msg.Log = false
	msg.Source = fmt.Sprintf("player_%d", gmeCfg.pl_id)
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

func GameSetLastResponse(lastResp util.RoPEMessage) {
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
			duration = tstmp.Sub(gmeRnd.begin)
			fmt.Printf("Alpha:%.6f\tUplink (req/s):%.6f\tDownlink (req/s):%.6f\n",
				statistics[len(statistics)-1].alpha,
				float64(statistics[len(statistics)-1].snt)/float64(duration.Seconds()),
				float64(statistics[len(statistics)-1].rcv)/float64(duration.Seconds()))
		}
	}
}
