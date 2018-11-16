package main

import (
	"github.com/nyu-distributed-systems-fa18/BaseChord/pb"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"time"

	"fmt"
	"net"
)

//PingTimeout -> milliseconds to ping a predecessor
const PingTimeout = 1000

//PingResponse -> response opject to give a ping response channel
type PingResponse struct {
	ret         *pb.PingPredecessorRet
	err         error
	predecessor string
}

/* ************** Chord RPCs ************** */
// Launch a GRPC service for this Raft peer.
func RunChordServer(kord *Chord, port int) {
	// Convert port to a string form
	portString := fmt.Sprintf(":%d", port)
	// Create socket that listens on the supplied port
	c, err := net.Listen("tcp", portString)
	if err != nil {
		// Note the use of Fatalf which will exit the program after reporting the error.
		log.Fatalf("Could not create listening socket %v", err)
	}
	// Create a new GRPC server
	s := grpc.NewServer()

	pb.RegisterChordServer(s, kord)
	log.Printf("Going to listen on port %v", port)

	// Start serving, this will block this function and only return when done.
	if err := s.Serve(c); err != nil {
		log.Fatalf("Failed to serve %v", err)
	}
}

func connectToNode(node string) (pb.ChordClient, error) {
	backoffConfig := grpc.DefaultBackoffConfig
	// Choose an aggressive backoff strategy here.
	backoffConfig.MaxDelay = 500 * time.Millisecond
	conn, err := grpc.Dial(node, grpc.WithInsecure(), grpc.WithBackoffConfig(backoffConfig))
	// Ensure connection did not fail, which should not happen since this happens in the background
	if err != nil {
		return pb.NewChordClient(nil), err
	}
	return pb.NewChordClient(conn), nil
}

type FindSuccessorRequest struct {
	arg      *pb.FindSuccessorArgs
	response chan pb.FindSuccessorRet
}

func (kord *Chord) FindSuccessor(ctx context.Context, arg *pb.FindSuccessorArgs) (*pb.FindSuccessorRet, error) {
	c := make(chan pb.FindSuccessorRet)
	kord.FindSuccessorChan <- FindSuccessorRequest{arg: arg, response: c} // TODO FIX!!
	result := <-c
	return &result, nil
}

type NotifyRequest struct {
	arg      *pb.NotifyArgs
	response chan pb.NotifyRet
}

func (kord *Chord) Notify(ctx context.Context, arg *pb.NotifyArgs) (*pb.NotifyRet, error) {
	c := make(chan pb.NotifyRet)
	kord.NotifyChan <- NotifyRequest{arg: arg, response: c}
	result := <-c
	return &result, nil
}

// PingRequest represents when a Node tries to ping us
type PingRequest struct {
	arg      *pb.PingPredecessorArgs
	response chan pb.PingPredecessorRet
}

func (kord *Chord) PingPredecessor(ctx context.Context, arg *pb.PingPredecessorArgs) (*pb.PingPredecessorRet, error) {
	c := make(chan pb.PingPredecessorRet)
	kord.PingChan <- PingRequest{arg: arg, response: c}
	result := <-c
	return &result, nil
}

/* **************************************** */

// Helper to stop a timer
func stopTimer(timer *time.Timer) {
	if !timer.Stop() {
		for len(timer.C) > 0 {
			<-timer.C
		}
	}
}

// Restart the supplied timer using a random timeout based on function above
func restartTimer(timer *time.Timer) {
	stopTimer(timer)
	timer.Reset(PingTimeout * time.Millisecond)
}

//RingNode -> represents a ring node that our node connects to i.e
type RingNode struct {
	ID        uint64
	IPAddress string
	conn      pb.ChordClient // Conenction to a ring node
}

// Chord -> object that represets our Node on the ring abstractly
type Chord struct {
	ID          uint64
	IPAddress   string
	successor   *RingNode
	predecessor *RingNode

	// Channels
	FindSuccessorChan chan FindSuccessorRequest
	NotifyChan        chan NotifyRequest
	PingChan          chan PingRequest
}

func runChord(fs *FileSystem, ip string, id uint64, port int, debug bool) {
	//TODO Finish chord implementation
	log.Printf("Chord ARGS: %v %v %v", ip, id, port)

	var chord Chord
	if debug {
		ringMap := map[int]string{
			3001: "127.0.0.1:3003",
			3003: "127.0.0.1:3001",
		}
		sIP := ringMap[port]
		// Connect to our other ring node
		succ, err := connectToNode(sIP)
		if err != nil {
			log.Fatalf("Failed to connect to GRPC server %v", err)
		}
		otherNode := &RingNode{ID: generateIDFromIP(sIP), IPAddress: sIP, conn: succ}
		chord = Chord{ID: id, IPAddress: ip,
			successor: otherNode, predecessor: otherNode, PingChan: make(chan PingRequest)}
	} else {
		chord = Chord{ID: id, IPAddress: ip, PingChan: make(chan PingRequest)}
	}

	go RunChordServer(&chord, port)

	pingTimer := time.NewTimer(PingTimeout * time.Millisecond)

	pingResponseChan := make(chan PingResponse)

	for {
		select {
		case pr := <-pingResponseChan:
			if pr.err != nil {
				log.Printf("Ping failed")
			} else {
				// We've recived a ping response from just respond
				log.Printf("Got ping response!")
			}

		case <-pingTimer.C:
			// We need to ping our predecessor
			log.Printf("My ip and id: %v %v", chord.IPAddress, chord.ID)
			if chord.successor != nil {
				log.Printf("Attempting to ping %v", chord.predecessor.IPAddress)
				go func(pred pb.ChordClient, predIP string) {
					ret, err := pred.PingPredecessor(context.Background(), &pb.PingPredecessorArgs{})
					pingResponseChan <- PingResponse{ret: ret, err: err, predecessor: predIP}
				}(chord.predecessor.conn, chord.predecessor.IPAddress)
			} else {
				log.Print("No successor")
			}
			restartTimer(pingTimer)

		case ping := <-chord.PingChan:
			log.Print("ping from successor")
			ping.response <- pb.PingPredecessorRet{}
		}
	}
}
