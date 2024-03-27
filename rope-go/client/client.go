package main

import (
	//"context"
	"context"
	"crypto/tls"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"io"

	quic "github.com/quic-go/quic-go"

	"gonum.org/v1/gonum/stat/distuv"

	"github.com/lthibault/jitterbug"

	"util"
)

var GitCommit string = "master"

var interarrivalsDistribution []string = []string{"uni", "exp", "const"}

// ClientConfig is the configuration for the current client
// A configuration can be defined in a config file following
// the toml structure
// type ClientConfig struct {
// IdDevice						string			`json:"idDevice"`
// Proxy						string			`json:"proxy"`
// RequestsPerSec				uint			`json:"requestsPerSec"`
// MaxConcurrentConnections		uint			`json:"maxConcurrentConnections"`
// RequestSize					uint			`json:"requestSize"`
// ResponseSize					uint			`json:"responseSize"`
// TestDuration					string			`json:"testDuration"`
// Timeout						string			`json:"timeout"`
// Logger						util.LoggerConf	`json:"logger"`
// }
type ClientConfig struct {
	IdDevice                 string
	destinations             []string
	interarrival             string
	RequestsPerSec           uint
	RequestSize              uint
	ResponseSize             uint
	MaxConcurrentConnections uint
	TestDuration             string
	Timeout                  string
	Logger                   util.LoggerConf
}

// Default configuration for a remote client
var defConf = ClientConfig{
	"Client_1",                 //IdDevice
	[]string{"localhost:4242"}, //Proxy
	"exp",
	1,
	1000,
	2,
	100,
	"10s",
	"30s",
	util.DefConfLog,
}

/*var defConf ClientConfig = ClientConfig{
	IdDevice: "Client_1",
	Proxy:    "localhost:4242",
	RequestsPerSec:           1,
	MaxConcurrentConnections: 1000,
	RequestSize:              2,
	ResponseSize:             3,
	TestDuration:             "10s",
	Timeout:                  "30s",
	Logger:                   util.DefConfLog,
	MonroeNodeid:             "none",
}*/

// GLOBAL VARIABLES
var idDevice string
var destinations []string
var interarrival string
var requestsPerSec float64
var maxConcurrentConnections uint
var testDuration time.Duration
var timeout time.Duration
var appcfg string
var clicfg string

var loggerEnabled = false

func main() {
	flag.StringVar(&clicfg, "config", clicfg, "client's configuration file")
	flag.Parse()
	fmt.Printf("Running client version %s\n", GitCommit)

	cfg := loadParam(clicfg)
	fmt.Printf("Starting idDevice: %s\n", idDevice)

	err := util.InitLogger()
	if err != nil {
		panic(err)
	}
	defer util.CloseLogger()

	loggerEnabled = util.IsLoggerEnabled()

	var wgPing sync.WaitGroup

	if loggerEnabled {
		if !cfg {
			tcp_logger := "tcp://" + util.LoggerCfg.LoggerAddress
			util.ZmqInfo(tcp_logger, idDevice)
		}
		for _, d := range destinations {
			c := make(chan float64)
			var str string = idDevice + " ---> " + d
			go util.PingTest(d, str, 30*time.Second, &wgPing, c, idDevice)
			go util.PingTest(d, str, testDuration, &wgPing, c, idDevice)
		}

	}

	// Configure application
	if appcfg != "" {
		fmt.Println("Loading forwaring policy:", appcfg)
		// for i = range RTT {
		// 	fmt.Printf("\t - RTT %s: %f", dest[i], RTT[i])
		// }
		loadForwardingConf(appcfg, destinations)
	} else {
		die("No forwaring policy specified")
	}

	err = clientMain(destinations)
	if err != nil {
		//panic(err)
		fmt.Println("Error!", err)
	}

	fmt.Println("Finished! Waiting 20 seconds...")

	time.Sleep(20 * time.Second)

	wgPing.Wait()
}

func loadParam(config string) bool {
	jsonFile, err := os.Open(config)
	if err == nil {
		fmt.Println("Using config file: ", config)
		defer jsonFile.Close()

		byteValue, _ := io.ReadAll(jsonFile) //ioutil.ReadAll(jsonfile)

		var cfg interface{}
		errj := json.Unmarshal(byteValue, &cfg)

		cfg_map := cfg.(map[string]interface{})

		fmt.Println(cfg_map)
		if errj != nil {
			fmt.Println("error:", errj)
		}

		if defConf.IdDevice == "none" {
			idDevice = "device_default"
		} else {
			idDevice = cfg_map["IdDevice"].(string)
		}

		// destinations = defConf.Proxy
		// requestInterval, _ = time.ParseDuration(defConf.RequestInterval)
		for _, v := range cfg_map["destinations"].([]interface{}) {
			destinations = append(destinations, fmt.Sprint(v))
		}
		requestsPerSec = cfg_map["RequestsPerSec"].(float64)
		interarrival = cfg_map["interarrival"].(string)
		maxConcurrentConnections = uint(cfg_map["MaxConcurrentConnections"].(float64))
		testDuration, _ = time.ParseDuration(cfg_map["TestDuration"].(string))
		timeout, _ = time.ParseDuration(cfg_map["Timeout"].(string))
		appcfg = cfg_map["AppCfg"].(string)

		byteValue, err = json.Marshal(cfg_map["Logger"])
		if err == nil {
			var logger util.LoggerConf
			json.Unmarshal(byteValue, &logger)
			util.SetLoggerParamFromConf(logger)
		}

		fmt.Printf("Client %s configuration:\n", idDevice)
		printParams()
		return true
	} else {
		fmt.Println("Unable to find config file: ", config)
		return false
	}
}

func printParams() {
	fmt.Println("Test duration: ", testDuration)
	fmt.Println(timeout)
	fmt.Println(destinations)
	fmt.Println(idDevice)
	fmt.Println(requestsPerSec)
	fmt.Println(maxConcurrentConnections)
	fmt.Println(testDuration)
	fmt.Println(timeout)
	fmt.Println(appcfg)
}

func setupTestDuration(done chan<- bool) {
	if testDuration > 0 {
		fmt.Printf("Test duration set to %v\n", testDuration)
		testTimer := time.NewTimer(testDuration)
		go func() {
			<-testTimer.C
			fmt.Println("Test ended")
			done <- true
		}()
	} else {
		fmt.Println("No test duration set, running until stopped")
	}
}

func exponentialTicker(rps float64) *jitterbug.Ticker {

	// auto tune
	fmt.Println("Tuning value to get", rps, "request per second.")

	rand.Seed(time.Now().UTC().UnixNano())

	beta := 0.000000001 * float64(rps)
	// var beta float64 = 1.0 / rps

	t := jitterbug.New(
		time.Millisecond*0,
		&jitterbug.Univariate{
			Sampler: &distuv.Gamma{
				Alpha: 1,
				Beta:  beta,
			},
		},
	)

	done := make(chan bool)

	var testSec uint = 20

	testTimer := time.NewTimer(time.Second * time.Duration(testSec))
	go func() {
		<-testTimer.C
		fmt.Println("Tuning ended")
		done <- true
	}()

	var counter float64 = 0

	start := time.Now()
external:
	for {
		//fmt.Println(counter)
		counter++

		select {
		case <-done:
			t.Stop()
			break external
		case <-t.C:
			continue external
		}
	}

	end := time.Now()

	fmt.Println("Duration:", end.Sub(start))

	fmt.Println("Expected ", rps*float64(testSec), "requests in", testSec, "seconds")
	fmt.Println("Got", counter, "requests in", testSec, "seconds")

	newBeta := beta + (beta - counter/float64(testSec)*0.000000001)
	// newBeta := beta

	fmt.Println("New beta:", newBeta)

	newTicker := jitterbug.New(
		time.Millisecond*0,
		&jitterbug.Univariate{
			Sampler: &distuv.Gamma{
				Alpha: 1,
				Beta:  newBeta,
			},
		},
	)

	return newTicker
	// return t
}

func clientMain(destinations []string) error {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"RoPEProtocol"},
	}

	quicConf := &quic.Config{
		MaxIdleTimeout:     timeout,
		MaxIncomingStreams: 10000000,
		KeepAlivePeriod:    10 * time.Second,
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "source", idDevice)

	var wg sync.WaitGroup

	fmt.Printf("Requests per second set to %v\n", requestsPerSec)
	fmt.Printf("Requests timeout set to %v\n", timeout)
	var sessions []quic.EarlyConnection
	for i, d := range destinations {
		// connect to all destinations
		s, err := quic.DialAddrEarly(ctx, d, tlsConf, quicConf)
		sessions = append(sessions, s)
		// func DialAddrEarly(addr string, tlsConf *tls.Config, config *Config) (EarlySession, error)
		if err != nil {
			fmt.Printf("Cannot connect to %s \n", d)
			log.Println(err)

			return err
		}
		fmt.Printf("Connected to %s\n", destinations[i])
		defer sessions[i].CloseWithError(0x1337, "Test finished!")

	}

	// ticker setup
	ticker := exponentialTicker(requestsPerSec)
	done := make(chan bool)
	counter := make(chan int64, maxConcurrentConnections)

	// https://yizhang82.dev/go-pattern-for-worker-queue
	// https://gobyexample.com/worker-pools

	setupTestDuration(done)
	util.SetupGracefulShutdown(func() {
		done <- true
	})

	// requests loop
external:
	for {
		if len(counter) == cap(counter) {
			fmt.Printf("maxConcurrentConnections=%d reached\n", maxConcurrentConnections)
			continue
		}
		id := time.Now().UnixNano()
		counter <- id
		wg.Add(1)
		go func() {
			defer wg.Done()
			newReq(sessions, id)
			<-counter
		}()

		select {
		case <-done:
			ticker.Stop()
			break external
		case <-ticker.C:
			continue external
		}
	}
	fmt.Printf("Waiting %d connections...\n", len(counter))
	wg.Wait()
	fmt.Println("Stopped")

	return nil
}

// https://pkg.go.dev/github.com/lucas-clemente/quic-go
// func newReq(session quic.EarlySession, id int64) error {
func newReq(session []quic.EarlyConnection, id int64) error {

	// CREA LA RICHIESTA
	idReq := idDevice + "_" + strconv.FormatInt(id, 10)
	req := util.RoPEMessage{ReqID: idReq,
		Log:  loggerEnabled,
		Type: util.Request}

	dest := ForwardDecision(&req, destinations)

	var idxdest int = 0
	for i, d := range destinations {
		if d == dest {
			idxdest = i
		}
	}

	stream, err := session[idxdest].OpenStream()
	if err != nil {
		return err
	}

	util.LogEvent(idReq, util.Sent, "New Request", loggerEnabled, idDevice)

	// INVIA LA RICHIESTA
	encoder := gob.NewEncoder(stream)
	err = encoder.Encode(req)

	if err != nil {
		fmt.Printf("Error sending: %s\n", idReq)
		return err
	}

	// RICEVE LA RISPOSTA
	// var resp util.RoPEMessage
	// decoder := gob.NewDecoder(stream)
	// err = decoder.Decode(&resp)

	var packet util.RoPEMessage
	var rcv bool = false
	decoder := gob.NewDecoder(stream)
	var wg sync.WaitGroup
	var cnt int64 = 0
	for {
		cnt += 1
		err = decoder.Decode(&packet)
		if err == io.EOF || err != nil {
			break
		} else {
			rcv = true
		}
		wg.Add(1)
		go forwardResponse(packet, &wg)
		if packet.Type == util.Response {
			util.LogEvent(idReq, util.Received, "Response", loggerEnabled, idDevice)
		} else {
			util.LogEvent(idReq, util.ReceivedError, "Error", loggerEnabled, idDevice)
		}

	}

	wg.Wait()

	if !rcv {
		fmt.Printf("Timeout or broken: %s\n", idReq)
		util.LogEvent(req.ReqID, util.Timeout, "Timeout response", loggerEnabled, idDevice)
		return err
	} else {
		stream.Close()
		return nil
	}

	// if resp.Type == util.Response {
	// 	util.LogEvent(idReq, util.Received, "Response", loggerEnabled, idDevice)
	// } else {
	// 	util.LogEvent(idReq, util.ReceivedError, "Error", loggerEnabled, idDevice)
	// }

	// fmt.Printf("Client: Received %s  %s\n", resp.ReqID, resp.Type)
}

func forwardResponse(packet util.RoPEMessage, wg *sync.WaitGroup) {
	defer wg.Done()

	// Log traffic
	if ForwardSetLastResponse != nil {
		ForwardSetLastResponse(packet)
	}
}
