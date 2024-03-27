package main

import (
	"fmt"
	"math"
	"os"
	"time"
)

type stats struct {
	alpha float64
	snt   int64
	rcv   int64
	succ  int64
	sum   float64
	ssum  float64
}

type bin struct {
	value float64
	count int
}

var statistics []stats
var hist []bin
var rtts [][]bin
var services [][]bin

// Configuration
var daddr []string
var destaddr string
var msgsize int32 = 0
var respsize int32 = 0

// App configuration
var app_timeout time.Duration

// Results
var binsize float64 = float64(time.Millisecond)

// For printing
var btime = time.Time{}
var obwindow time.Duration

func write_stats(fnm string, flag int) {
	fmt.Printf("Writing to file %s...\n", fnm)
	f, err := os.OpenFile(fnm, flag, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
	for k, current := range statistics {
		mean := current.sum / float64(current.rcv)
		stddev := math.Sqrt(float64(current.ssum)/float64(current.rcv) - math.Pow(float64(current.sum)/float64(current.rcv), 2))
		newLine := fmt.Sprintf("step: %d\t| alpha: %.4f\t| pkts sent: %d\t| pkt received: %d\t| successes: %d\t| mean[ms]: %.9f\t| stddev[ms]: %.9f",
			k, current.alpha, current.snt, current.rcv, current.succ, mean, stddev)
		_, err = fmt.Fprintln(f, newLine)
		if err != nil {
			fmt.Println(err)
			f.Close()
			return
		}
	}
	err = f.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
}

func write_hist(fnm string, flag int) {
	f, err := os.OpenFile(fnm, flag, 0644)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, b := range hist {
		newLine := fmt.Sprintf("%.6f\t %d", b.value, b.count)
		_, err = fmt.Fprintln(f, newLine)
		if err != nil {
			fmt.Println(err)
			f.Close()
			return
		}
	}
	err = f.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
}

func write_histograms(hst_tgt [][]bin, fnm string, flag int) {
	for idx, hrtt := range hst_tgt {
		f, err := os.OpenFile(fmt.Sprintf("%s_%d", fnm, idx), flag, 0644)
		if err != nil {
			fmt.Println(err)
			return
		}
		for _, b := range hrtt {
			newLine := fmt.Sprintf("%.6f\t %d", b.value, b.count)
			_, err = fmt.Fprintln(f, newLine)
			if err != nil {
				fmt.Println(err)
				f.Close()
				return
			}
		}
		err = f.Close()
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}
