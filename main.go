package main

import (
	"flag"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/goraft/raft"
	"github.com/igm/raftdzmq/command"
	"github.com/igm/raftdzmq/server"
)

var verbose bool
var trace bool
var debug bool

// zmq  host:port to bind socket to (tcp://host:port) -> raft messaging
// http host:httpport to serve HTTP resquests (http://host:httpport) -> for clients
var host string
var port int
var httpport int
var join string

func init() {
	flag.BoolVar(&verbose, "v", false, "verbose logging")
	flag.BoolVar(&trace, "trace", false, "Raft trace debugging")
	flag.BoolVar(&debug, "debug", false, "Raft debugging")
	flag.StringVar(&host, "h", "localhost", "hostname")
	flag.IntVar(&port, "p", 5555, "port")
	flag.IntVar(&httpport, "hp", 8080, "http port")
	flag.StringVar(&join, "join", "", "host:port of leader to join (http)")
}

func main() {
	log.SetFlags(0)
	flag.Parse()
	if verbose {
		log.Print("Verbose logging enabled.")
	}
	if trace {
		raft.SetLogLevel(raft.Trace)
		log.Print("Raft trace debugging enabled.")
	} else if debug {
		raft.SetLogLevel(raft.Debug)
		log.Print("Raft debugging enabled.")
	}

	rand.Seed(time.Now().UnixNano())

	// Setup commands.
	raft.RegisterCommand(&command.WriteCommand{})

	// Set the data directory.
	if flag.NArg() == 0 {
		log.Fatal("Data path argument required")
	}
	path := flag.Arg(0)
	if err := os.MkdirAll(path, 0744); err != nil {
		log.Fatalf("Unable to create path: %v", err)
	}

	log.SetFlags(log.LstdFlags)
	s := server.New(path, host, port, httpport)
	log.Fatal(s.ListenAndServe(join))
}
