package main

import (
	"context"
	"crypto/tls"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	quic "github.com/quic-go/quic-go"
	//modulo locale
	"util"
)

type DestServer string

var GitCommit string = "master"

var routingTbl map[string]string
var listenAddr string = ":4242"
var idDevice string = "RoutingProxy"
var confFile string = "../config/GoConf.toml"
var timeout = 30 * time.Second
var parallelConn int = -1
var tokens_up, tokens_dn chan struct{}

func main() {
	// fmt.Println("+main")
	fmt.Printf("Running proxy version %s\n", GitCommit)

	var sinks, nh string = "", ""

	// Command-line parameters
	flag.StringVar(&sinks, "sinks", sinks, "list of destinations")
	flag.StringVar(&nh, "nexthop", nh, "list of routes toward the sestinations")
	flag.StringVar(&listenAddr, "listen", listenAddr, "listen address")
	flag.StringVar(&idDevice, "idDevice", idDevice, "id device")
	flag.StringVar(&confFile, "confFile", confFile, fmt.Sprintf("forwarding policy configuration file path (supported logics: %v)", supportedLogics()))
	flag.IntVar(&parallelConn, "parallelConn", parallelConn, "max number of concurent connections")
	flag.DurationVar(&timeout, "timeout", timeout, "request timeout duration")

	// Logger
	util.SetLoggerParam()

	// Parse command-line parameters
	flag.Parse()

	// Split destination addresses at whitespaces
	dest := strings.Split(sinks, " ")
	nhop := strings.Split(nh, " ")

	routingTbl = make(map[string]string)
	fmt.Println("Populating routing table")
	for j, d := range dest {
		fmt.Println(d, " ", nhop[j])
		routingTbl[d] = nhop[j]
	}

	printParams()

	fmt.Printf("Starting idDevice: %s\n", idDevice)

	err := util.InitLogger()
	if err != nil {
		panic(err)
	}

	util.SetupGracefulShutdown(func() {
		util.CloseLogger()
		os.Exit(0)
	})

	// Limit parallelism
	if parallelConn > 0 {
		tokens_up = make(chan struct{}, parallelConn)
		tokens_dn = make(chan struct{}, parallelConn)
	}

	// Ping test
	var wgPing sync.WaitGroup

	var RTT map[string]float64 = make(map[string]float64)
	var channels map[string]chan float64 = make(map[string]chan float64)
	for d, h := range routingTbl {
		channels[d] = make(chan float64, 1)
		var str string = "\n\t" + listenAddr + " ---> " + h
		go util.PingTest(h, str, 20*time.Second, &wgPing, channels[d], idDevice)
	}
	for d, c := range channels {
		h := routingTbl[d]
		RTT[h] = <-c
		fmt.Printf("\t - RTT %s: %f\n", h, RTT[h])
	}

	// Load forwarding logic
	if confFile != "" {
		fmt.Println("Loading forwaring policy:", confFile)
		loadForwardingConf(confFile, RTT)
	}
	//  else {
	// 	die("No forwaring policy specified")
	// }

	log.Fatal(proxyMain(listenAddr))
	// Wait until all the ping tests are done
	wgPing.Wait()
	// fmt.Println("-main")
}

func printParams() {
	for key, value := range routingTbl {
		fmt.Println("Destination: ", key, " Next hop: ", value)
	}
	fmt.Println(idDevice)
	fmt.Println(confFile)
	fmt.Println(listenAddr)
	fmt.Println(confFile)
	fmt.Println(timeout)
}

func proxyMain(in_addr string) error {
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
	// ctx = context.WithValue(ctx, "source", idDevice)

	// Open connections towards different destinations
	var sessions map[string]quic.Connection = make(map[string]quic.Connection)
	for _, addr := range routingTbl {
		s, err := quic.DialAddrEarly(ctx, addr, tlsConf, quicConf)
		sessions[addr] = s
		if err != nil {
			fmt.Printf("Cannot connect to %s \n", addr)
			log.Println(err)
			return err
		}
		defer s.CloseWithError(0x1337, "Proxy finished")
	}

	// Wait for incoming connections
	listener, err := quic.ListenAddrEarly(in_addr, util.GenerateTLSConfig(), quicConf)
	if err != nil {
		return err
	}

	fmt.Printf("GO Proxy Ready %s\n", in_addr)
	fmt.Printf("Requests timeout set to %v\n", timeout)

	for {
		// Accept incoming connection at in_addr
		sess, err := listener.Accept(context.Background())
		if err != nil {
			continue
		}

		// Handle a new connection in a go routine
		go newClient(sess, sessions)
	}
}

// Lisen for traffic incoming from a connection
func newClient(sess quic.Connection, sessions map[string]quic.Connection) {
	fmt.Printf("New connection from %s\n", sess.RemoteAddr().String())

	for {
		// Limit parallelism
		if parallelConn > 0 {
			tokens_up <- struct{}{}
		}
		stream, err := sess.AcceptStream(context.Background())
		if err != nil {
			fmt.Println("AcceptStream error:", err)
			return
		}

		// Incoming requests are processed in a working thread
		go newRequest(stream, sessions)

	}
}

func newRequest(stream quic.Stream, sessions map[string]quic.Connection) {
	defer stream.Close()

	// Read the request
	var packet util.RoPEMessage
	decoder := gob.NewDecoder(stream)
	err := decoder.Decode(&packet)

	if err != nil {
		return
	}

	util.LogEvent(packet.ReqID, util.Received, "Ricevuta richiesta proxy", packet.Log, idDevice)

	// Create a response
	var resp util.RoPEMessage

	// Forwarding logic (application logic)
	if confFile != "" {
		ForwardDecision(&packet)
	}

	dest, ok := routingTbl[packet.Destination]
	if ok {
		// Send the response to the intended destination (the server)
		resp = forwardRequest(sessions[dest], packet)
	} else {
		// Destination not found
		util.LogEvent(packet.ReqID, util.RoutingErr, "No route", packet.Log, idDevice)
		fmt.Println("No route for ", packet.Destination, ". Available routes: ", dest)
		packet.Type = util.NoRoute
		packet.Body = make([]byte, 0)
		encoder := gob.NewEncoder(stream)
		_ = encoder.Encode(packet)

		util.LogEvent(packet.ReqID, util.Sent, "Answering No route", packet.Log, idDevice)
		return
	}
	// Log traffic
	// if ForwardSetLastResponse != nil {
	// 	ForwardSetLastResponse(&packet)
	// }

	// util.LogEvent(resp.ReqID, util.Sent, "Sending response", resp.Log, idDevice)
	// util.LogEvent(resp.ReqID, util.Received, packet.Destination, resp.Log, idDevice)

	// Forward packet to the inended destination (the client who started the connection)
	encoder := gob.NewEncoder(stream)
	encoder.Encode(resp)

}

func forwardRequest(serverSession quic.Connection, packet util.RoPEMessage) util.RoPEMessage {

	error_resp := util.RoPEMessage{ReqID: packet.ReqID,
		Log:         packet.Log,
		ResSize:     packet.ResSize,
		Type:        util.ServerNotFound,
		Body:        make([]byte, 0),
		Source:      packet.Source,
		Hop:         idDevice,
		Destination: packet.Destination}

	// Forward the packet
	stream, err := serverSession.OpenStream() //OpenStreamSync(context.Background())
	if err != nil {
		return error_resp
	}
	defer stream.Close()

	encoder := gob.NewEncoder(stream)
	err = encoder.Encode(packet)

	util.LogEvent(packet.ReqID, util.Sent, packet.Destination, packet.Log, idDevice)

	if parallelConn > 0 {
		<-tokens_up
	}

	if err != nil {
		fmt.Printf("Error forwarding: %s\n", packet.ReqID)
		return error_resp
	}

	// RICEVE LA RISPOSTA DAL SERVER (MEC O CLOUD)
	if parallelConn > 0 {
		tokens_dn <- struct{}{}
	}
	var resp util.RoPEMessage
	decoder := gob.NewDecoder(stream)
	err = decoder.Decode(&resp)

	if err != nil {
		fmt.Printf("Timeout or broken: %s\n", packet.ReqID)
		error_resp.Type = util.ServerTimeout
		util.LogEvent(packet.ReqID, util.Timeout, packet.Destination, packet.Log, idDevice)
		return error_resp
	}

	if ForwardSetLastResponse != nil {
		ForwardSetLastResponse(&resp)
	}

	// util.LogEvent(resp.ReqID, util.Received, packet.Destination, resp.Log, idDevice)
	util.LogEvent(resp.ReqID, util.Sent, "Sending response", resp.Log, idDevice)

	if parallelConn > 0 {
		<-tokens_dn
	}

	return resp
}
