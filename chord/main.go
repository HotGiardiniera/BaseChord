package main

import (
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"
	rand "math/rand"
	"math"
	"net"
	"os"
	"time"
	"crypto/sha1"

	"github.com/nyu-distributed-systems-fa18/BaseChord/pb"
)

const M = 7 // Max number of nodes on our ring

func main() {
	// Argument parsing
	//var r *rand.Rand
	var seed int64
	var clientPort int
	var chordPort int

	flag.Int64Var(&seed, "seed", -1,
		"Seed for random number generator, values less than 0 result in use of time")
	flag.IntVar(&clientPort, "port", 3000,
		"Port on which server should listen to client requests")
	flag.IntVar(&chordPort, "chord", 3001,
		"Port on which server should listen to Raft requests")
	flag.Parse()

	/* Don't know if we need the below, so commenting it out for safe keeping
	// Initialize the random number generator
	if seed < 0 {
		r = rand.New(rand.NewSource(time.Now().UnixNano()))
	} else {
		r = rand.New(rand.NewSource(seed))
	}
	*/
	// TODO remove the line below and the import if we don't need time anymore
	var _ = time.Now
	var _ = rand.New

	// Get hostname
	name, err := os.Hostname()
	if err != nil {
		// Without a host name we can't really get an ID, so die.
		log.Fatalf("\u001b[31mCould not get hostname\u001b[0m")
	}

	ip := fmt.Sprintf("%s:%d", name, chordPort)
	log.Printf("Starting peer with ID %s", ip)

	// Determine generate id as a ring number
	sha_hash := sha1.New()
	sha_hash.Write([]byte(ip))
	hashed := sha_hash.Sum(nil)
	hashed64 := bytes_to_int64(hashed[:8])
	id := truncate_bits(M, hashed64) % uint64(math.Pow(2, float64(M)))

	// Convert port to a string form
	portString := fmt.Sprintf(":%d", clientPort)
	// Create socket that listens on the supplied port
	c, err := net.Listen("tcp", portString)
	if err != nil {
		// Note the use of Fatalf which will exit the program after reporting the error.
		log.Fatalf("\u001b[31mCould not create listening socket %v\u001b[0m", err)
	}
	// Create a new GRPC server
	fs := grpc.NewServer()

	// Initialize KVStore
	fileSystem := FileSystem{C: make(chan InputChannelType), fileSystem: make(map[string]string)}
	go chord(&fileSystem, ip, id, chordPort)

	// Tell GRPC that fs will be serving requests for the fileSystem service and
	//should use store as the struct whose methods should be
	//called in response.
	pb.RegisterFileSystemServer(fs, &fileSystem)
	log.Printf("\u001b[32mGoing to listen for client requests on port %v\u001b[0m", clientPort)
	// Start serving, this will block this function and only return when done.
	if err := fs.Serve(c); err != nil {
		log.Fatalf("\u001b[31mFailed to serve %v\u001b[0m", err)
	}
	log.Printf("\u001b[34mDone listening\u001b[0m")
}
