package main

import (
	"context"
	"crypto/tls"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	quic "github.com/quic-go/quic-go"

	//librerie locali
	"util"
)

type ServerConfig struct {
	IdDevice    string
	Proxy       string
	ServiceTime time.Duration
	Port        string
	QueueLen    uint
	Workers     uint
	Logger      util.LoggerConf
}

type destAddresses []string

func (i *destAddresses) String() string {
	return "my string representation"
}

func (i *destAddresses) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// Global variables
var listenAddr string = ":4242"
var idDevice string = "Server"
var destinations destAddresses
var appcfg string = "/appcfg.toml"
var enablelog bool = false

var queueLen int = 10
var workers int = 5

var timeout = 30 * time.Second

var config ServerConfig

type JobRequest struct {
	Request    util.RoPEMessage
	QuicStream quic.Stream
}

var sessions map[string]quic.EarlyConnection

func main() {
	// Read configuration from commandline
	flag.StringVar(&listenAddr, "listen", listenAddr, "LISTEN host:port")
	flag.Var(&destinations, "connections", "opens connections with the provided address")
	flag.StringVar(&idDevice, "idDevice", idDevice, "id device")
	flag.IntVar(&queueLen, "queueLen", queueLen, "length queue")
	flag.IntVar(&workers, "workers", workers, "number of workers")
	flag.StringVar(&appcfg, "appConfig", appcfg, "configurationf file for the application running in the server")
	flag.BoolVar(&enablelog,"log", false, "Enable logging permanence time in the server")
	util.SetLoggerParam()
	flag.Parse()

	fmt.Printf("Starting idDevice: %s\n", idDevice)

	var wgPing sync.WaitGroup
	var RTT []float64
	if len(destinations) != 0 {
		// Ping test

		var channels []chan float64
		var i int
		for i = range destinations {
			channels = append(channels, make(chan float64, 1))
			var str string = idDevice + " ---> " + destinations[i]
			go util.PingTest(destinations[i], str, 20*time.Second, &wgPing, channels[i], idDevice)
		}
		for i = range channels {
			RTT = append(RTT, <-channels[i])
		}
	}

	// Configure application
	if appcfg != "" {
		fmt.Println("Loading forwaring policy:", appcfg)
		// loadForwardingConf(appcfg, destinations, RTT)
		loadForwardingConf(appcfg, destinations)
	} else {
		die("No forwaring policy specified")
	}

	err := util.InitLogger()
	if err != nil {
		panic(err)
	}
	util.SetupGracefulShutdown(func() {
		util.CloseLogger()
		os.Exit(0)
	})
	defer util.CloseLogger()

	// Wait until all the ping tests are done
	wgPing.Wait()

	log.Fatal(server(listenAddr))
}

// Server starts a listener and creates workers to handle incoming requests
func server(listenAddr string) error {
	requestQueue := make(chan JobRequest, queueLen)
	listener, err := quic.ListenAddrEarly(listenAddr, util.GenerateTLSConfig(), &quic.Config{MaxIdleTimeout: timeout, MaxIncomingStreams: 10000000, KeepAlivePeriod: 10 * time.Second})
	if err != nil {
		fmt.Printf("Error in creating listener: %s\n", err)
	}

	if len(destinations) != 0 {
		go func() {
			t, _ := time.ParseDuration("10s")
			time.Sleep(t)
			sessions = make(map[string]quic.EarlyConnection)
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
			for i, d := range destinations {
				// connect to all destinations
				fmt.Println("Connecting to ", d)
				s, err := quic.DialAddrEarly(ctx, d, tlsConf, quicConf)
				if err == nil {
					fmt.Printf("Cannot connect to %s \n", d)
					log.Println(err)
					panic(err)
				}
				sessions[d] = s
				fmt.Printf("Connected to %s\n", destinations[i])
			}
		}()
	}

	time := time.Now().UnixNano()
	// Create and launch go routines to handle incoming requests
	for i := 0; i < workers; i++ {
		go worker(i, requestQueue, time)
	}

	fmt.Printf("Server ready %s, workers %d, queue length=%d\n", listenAddr, workers, queueLen)

	for {
		sess, err := listener.Accept(context.Background())
		if err != nil {
			fmt.Printf("Session error: %s\n", err)
			return err
		}

		go newRequest(sess, requestQueue)

	}
	return nil
}

// newRequest handles an incoming stream
// func newRequest(sess quic.Session, queue chan<- JobRequest) {
func newRequest(sess quic.Connection, queue chan<- JobRequest) {
	fmt.Println("Preparazione dell'elaborazione delle richieste")
	for {
		// Session's stream
		stream, err := sess.AcceptStream(context.Background())
		if err != nil {
			fmt.Println("Accept Stream error:", err)
			fmt.Println(err.Error())
			if err.Error() == "Application error 0x1337: Finish" {
				os.Exit(0) // Terminates the experimemt
			}
			return
		}
		// Launch a new go routine to process a request and send a response
		go newRequestServer(stream, queue)
	}
}

// newRequestServer receives and enqueue incoming packets
func newRequestServer(stream quic.Stream, queue chan<- JobRequest) {
	var packet util.RoPEMessage
	// Read the incoming request
	decoder := gob.NewDecoder(stream)
	err := decoder.Decode(&packet)

	if err != nil {
		fmt.Printf("Request error: %s\n", err)
	}
	util.LogEvent(packet.ReqID, util.Received, "Recieved packet", packet.Log, idDevice)
	// Write to bytearray
	if enablelog{
		t := time.Now()
		encoding_tmestp, _ := t.MarshalBinary()
		var body_start []byte = packet.Body[:(len(packet.Body)-2*len(encoding_tmestp))]
		var body_end []byte = packet.Body[(len(packet.Body)-len(encoding_tmestp)):]
		packet.Body = append(body_start, append(encoding_tmestp,body_end...)...)
	}
	select {
	case queue <- JobRequest{packet, stream}:
		util.LogEvent(packet.ReqID, util.Enqueued, fmt.Sprintf("Queuing packet; Queue occupacy [%d]", len(queue)), packet.Log, idDevice)
	default:
		fmt.Printf("Queue full\n")
		util.LogEvent(packet.ReqID, util.FullQueue, "Queue full", packet.Log, idDevice)
		if ForwardBlock != nil {
			ForwardBlock(&packet, stream)
		} else {
			packet.Type = util.QueueFull
			packet.Body = make([]byte, 0)
			packet.Hop = packet.Destination
			packet.Destination = packet.Source
			packet.Source = packet.Hop
			encoder := gob.NewEncoder(stream)
			err := encoder.Encode(packet)
			util.LogEvent(packet.ReqID, util.Sent, "Sending 'Queue full'", packet.Log, idDevice)
			if err != nil {
				fmt.Printf("Error while sending 'Queue full' message: %s \n", err)
			}
		}
		stream.Close()
	}

}

// worker dequeue a packet, process it, and sends a response
func worker(i int, queue <-chan JobRequest, t int64) {
	//viene ritornato la JobRequest all'interno della queue
	for request := range queue {
		packet := request.Request    //pacchetto di tipo Message
		stream := request.QuicStream //Ricava lo stream

		util.LogEvent(packet.ReqID, util.Dequeued, fmt.Sprintf("Worker %d preleva richiesta dalla coda; Occupazione [%d]", i, len(queue)), packet.Log, idDevice)

		// Call the application logic
		send := true
		var i int64 = 0
		for send {
			send = ForwardDecision(&packet, &sessions, i)
			// Adjust the size of the response while keeping the initial bytes
			if packet.ResSize > int32(len(packet.Body)) {
				packet.Body = append(packet.Body, make([]byte, packet.ResSize-int32(len(packet.Body)))...)
			} else {
				if packet.ResSize < int32(len(packet.Body)) {
					packet.Body = packet.Body[:packet.ResSize]
				}
			}
			packet.Type = util.Response

			i += 1
			util.LogEvent(packet.ReqID, util.Processed, fmt.Sprintf("Worker %d termina elaborazione", i), packet.Log, idDevice)
			// Send a response
			if packet.Destination != "" {
				encoder := gob.NewEncoder(stream)
				err := encoder.Encode(packet)
				util.LogEvent(packet.ReqID, util.Sent, fmt.Sprintf("Worker %d invia risposta", i), packet.Log, idDevice)

				if ForwardSetLastResponse != nil {
					ForwardSetLastResponse(&packet)
				}

				if err != nil {
					fmt.Printf("Errore nell'invio della risposta %s \n", err)
				}
			}
		}
		stream.Close()

	}
}
