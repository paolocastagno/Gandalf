package main

import (
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml"

	//modulo locale
	"util"
)

/////////// Logic parsing //////////////

var ForwardDecision func(msg *util.RoPEMessage, destinations []string) string
var ForwardSetLastResponse func(util.RoPEMessage)

var logicsMap = map[string]func(*toml.Tree){
	"fixed":     parseFixed,
	"game":      parseGame,
	"gameFixed": parseFixedGame,
}

func parseFixed(config *toml.Tree) {
	// Read the destinatin address from configuration
	dest := config.Get("variables.destination")
	reqsize := config.Get("variables.requestSize")
	ressize := config.Get("variables.responseSize")
	appto := config.Get("game.app_timeout")
	bsze := config.Get("game.bin_size")

	InitFixed(dest, reqsize, ressize, appto, bsze)
	ForwardDecision = FixedDecision
	ForwardSetLastResponse = FixedSetLastResponse
}

func parseGame(config *toml.Tree) {
	// Read the destinatin address from configuration
	appto := config.Get("game.app_timeout")
	plid := config.Get("game.player_id")
	rnd_dtn := config.Get("game.round")
	step := config.Get("game.step")
	minstep := config.Get("game.min_step")
	maxstep := config.Get("game.max_step")
	dfact := config.Get("game.d_fact")
	mfact := config.Get("game.m_fact")
	alpha := config.Get("game.alpha")
	bsze := config.Get("game.bin_size")

	var dest = config.Get("variables.destination")
	var msgsize = config.Get("variables.requestSize")

	InitGame(plid, rnd_dtn, dest, msgsize, appto, alpha, step, minstep, maxstep, dfact, mfact, bsze)
	ForwardDecision = GameDecision
	ForwardSetLastResponse = GameSetLastResponse
}

func parseFixedGame(config *toml.Tree) {
	// Read the destinatin address from configuration
	appto := config.Get("game.app_timeout")
	plid := config.Get("game.player_id")
	rnd_dtn := config.Get("game.round")
	alpha := config.Get("game.alpha")
	bsze := config.Get("game.bin_size")

	var dest = config.Get("variables.destination")
	var msgsize = config.Get("variables.requestSize")

	InitFixedGame(plid, rnd_dtn, dest, msgsize, appto, alpha, bsze)
	ForwardDecision = FixedGameDecision
	ForwardSetLastResponse = FixedGameSetLastResponse
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

func loadForwardingConf(confFile string, destinations []string) {
	config, err := toml.LoadFile(confFile)
	if err != nil {
		die("Error loading configuration", err.Error())
	} else {
		// retrieve data directly
		logicName := config.Get("logic.name").(string)

		if isLogicSupported(logicName) {
			logicsMap[logicName](config)
		} else {
			die("No supported logic name specified")
		}
	}
}
