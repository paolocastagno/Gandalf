package main

import (
	"fmt"
	"log"
	"os"

	toml "github.com/pelletier/go-toml"
	quic "github.com/quic-go/quic-go"

	//modulo locale
	"util"
)

/////////// Logic parsing //////////////

var ForwardDecision func(*util.RoPEMessage, *map[string]quic.EarlyConnection, int64) bool
var ForwardBlock func(*util.RoPEMessage, quic.Stream)
var ForwardSetLastResponse func(*util.RoPEMessage)

var logicsMap = map[string]func(*toml.Tree){
	"reply":      parseReply,
	"replyrelay": parseRReply,
}

func parseReply(config *toml.Tree) {

	dest := config.Get("variables.destination")
	// rtt := config.Get("variables.rtt")
	wtt := config.Get("variables.workTimeType")
	wt := config.Get("variables.workTime")
	pkts := config.Get("variables.packets")

	// InitReply(wt, dest, rtt, pkts)
	InitReply(wt, wtt, dest, pkts)
	ForwardDecision = ReplyDecision
	ForwardBlock = nil
	ForwardSetLastResponse = ReplySetLastResponse
}

func parseRReply(config *toml.Tree) {

	dest := config.Get("variables.destination")
	// rtt := config.Get("variables.rtt")
	wt := config.Get("variables.workTime")
	pkts := config.Get("variables.packets")

	// InitRReply(wt, dest, rtt, pkts)
	InitRReply(wt, dest, pkts)
	ForwardDecision = RReplyDecision
	ForwardBlock = RReplyBlock
	ForwardSetLastResponse = RReplySetLastResponse
}

/////////////// Helper functions ////////////

func die(msg ...interface{}) {
	fmt.Println(msg...)
	os.Exit(1)
}

func supportedLogics() []string {
	logicsNames := make([]string, 0, len(logicsMap))
	for name := range logicsMap {
		logicsNames = append(logicsNames, name)
	}
	return logicsNames
}

func isLogicSupported(logicName string) bool {
	for _, name := range supportedLogics() {
		if name == logicName {
			return true
		}
	}
	return false
}

func loadForwardingConf(confFile string, addr []string) { //, rtt []float64) {
	config, err := toml.LoadFile(confFile)
	if err != nil {
		log.Fatal(err)
	} else {
		// Add to the configuration the available connections, if any
		// if len(addr) != 0 {
		// 	fmt.Println("Adding destinations to appconfig: ", addr)
		// 	config.Set("variables.destination", addr)
		// 	fmt.Println("Adding rtt to appconfig: ", rtt)
		// 	config.Set("variables.destination", rtt)
		// }
		logicName := config.Get("logic.name").(string)
		if isLogicSupported(logicName) {
			logicsMap[logicName](config)
		} else {
			die("Application is not supported (" + logicName + ")")
		}
	}
}
