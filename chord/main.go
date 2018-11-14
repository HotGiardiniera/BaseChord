package main

import (
	"flag"
	"fmt"
	"log"
	rand "math/rand"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"

	"github.com/nyu-distributed-systems-fa18/BaseChord/pb"
)

func main() {
	// Argument parsing
	var r *rand.Rand
	var seed int64
	var clientPort int
	var chordPort int
	flag.Int64Var(&seed, "seed", -1,
		"Seed for random number generator, values less than 0 result in use of time")
	flag.IntVar(&clientPort, "port", 3000,
		"Port on which server should listen to client requests")
	flag.IntVar(&raftPort, "chord", 3001,
		"Port on which server should listen to Raft requests")
	flag.Parse()

	// Initialize the random number generator
	if seed < 0 {
		r = rand.New(rand.NewSource(time.Now().UnixNano()))
	} else {
		r = rand.New(rand.NewSource(seed))
	}

	// Get hostname
	name, err := os.Hostname()
	if err != nil {
		// Without a host name we can't really get an ID, so die.
		log.Fatalf("\u001b[31mCould not get hostname\u001b[0m")
	}

	id := fmt.Sprintf("%s:%d", name, chordPort)
	log.Printf("Starting peer with ID %s", id)

	// Convert port to a string form
	portString := fmt.Sprintf(":%d", clientPort)
	// Create socket that listens on the supplied port
	c, err := net.Listen("tcp", portString)
	if err != nil {
		// Note the use of Fatalf which will exit the program after reporting the error.
		log.Fatalf("\u001b[31mCould not create listening socket %v\u001b[0m", err)
	}
	// Create a new GRPC server
	s := grpc.NewServer()

	// Initialize KVStore
	store := FileSystem{C: make(chan InputChannelType), store: make(map[string]string)}
	go serve(&store, r, &peers, id, raftPort)

	// Tell GRPC that s will be serving requests for the KvStore service and should use store (defined on line 23)
	// as the struct whose methods should be called in response.
	pb.RegisterKvStoreServer(s, &store)
	log.Printf("\u001b[32mGoing to listen for client requests on port %v\u001b[0m", clientPort)
	// Start serving, this will block this function and only return when done.
	if err := s.Serve(c); err != nil {
		log.Fatalf("\u001b[31mFailed to serve %v\u001b[0m", err)
	}
	log.Printf("\u001b[34mDone listening\u001b[0m")
}
