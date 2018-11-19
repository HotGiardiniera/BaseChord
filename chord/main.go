package main

import (
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"

	"net"
	"os"

	"github.com/nyu-distributed-systems-fa18/BaseChord/pb"
)

// M is our max number of nodes on our ring
const M = 7

func main() {
	// Argument parsing
	var clientPort int
	var chordPort int
	var joinIp string // Start a chord node and join to it immediately

	// Debug mode: this will force a node to assume it is joined in a ring of size 2
	// This is mainly just to test RPCs. It will assume Node 1 is port 3001 and node 2 is 3003.
	// in debug mode these two will only be joined together and (TODO) ignore any other joins
	var debug bool

	flag.IntVar(&clientPort, "port", 3000,
		"Port on which server should listen to client requests")
	flag.IntVar(&chordPort, "chord", 3001,
		"Port on which server should listen to Raft requests")
	flag.BoolVar(&debug, "debug", false,
		"Allows for debug printing mechanisms")
	flag.StringVar(&joinIp, "peer", "",
		"Join on the ring through a peer's IP address")
	flag.Parse()

	// Get hostname
	name, err := os.Hostname()
	if err != nil {
		// Without a host name we can't really get an ID, so die.
		log.Fatalf("\u001b[31mCould not get hostname\u001b[0m")
	}

	ip := fmt.Sprintf("%s:%d", name, chordPort)
	log.Printf("Starting peer with ID %s", ip)

	// Generate id as a ring number
	id := generateIDFromIP(ip)

	// Convert port to a string form
	portString := fmt.Sprintf(":%d", clientPort)
	// Create socket that listens on the supplied port
	c, err := net.Listen("tcp", portString)
	if err != nil {
		// Note the use of Fatalf which will exit program after reporting the error.
		log.Fatalf("\u001b[31mCould not create listening socket %v\u001b[0m", err)
	}

	// Create a new GRPC server
	fs := grpc.NewServer()

	// Initialize FileSystem
	fileSystem := FileSystem{C: make(chan InputChannelType), fileSystem: make(map[string]string)}
	go runChord(&fileSystem, ip, id, chordPort, debug)

	// Tell GRPC that fs will be serving requests for the fileSystem service and
	//should use file system as struct whose methods should be called in response.
	pb.RegisterFileSystemServer(fs, &fileSystem)
	{
		log.Printf("\u001b[32mGoing to listen for client requests on port %v\u001b[0m", clientPort)
	} // Start serving, this will block this function and only return when done.
	if err := fs.Serve(c); err != nil {
		log.Fatalf("\u001b[31mFailed to serve %v\u001b[0m", err)
	}
	log.Printf("\u001b[34mDone listening\u001b[0m")
}
