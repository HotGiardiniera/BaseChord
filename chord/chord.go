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

/* ************** Chord structs ************** */
//PingResponse -> response object to give a ping response channel
type PingResponse struct {
	ret         *pb.PingPredecessorRet
	err         error
	predecessor string
}

//FindSucessorResponse -> response object to give a find successor response channel
type FindSucessorResponse struct {
	ret *pb.FindSuccessorRet
	err error
}

// RingNode -> represents a ring node that our node connects to
type RingNode struct {
	Ip   string
	conn pb.ChordClient // Conenction to a ring node
}

// Chord -> object that represets our Node on the ring abstractly
type Chord struct {
	Id          uint64
	Ip          string
	successor   uint64
	predecessor uint64
	finger      [M]unint64
	ringMap     map[uint64]*RingNode
	// Request channels
	JoinChan          chan bool
	FindSuccessorChan chan FindSuccessorRequest
	NotifyChan        chan NotifyRequest
	PingChan          chan PingRequest
	// Response channels
	pingResponseChan          chan PingResponse
	findSuccessorResponseChan chan FindSucessorResponse
}

////////////////////////////////////////////////////////////////////////////////

/* ************** Chord RPCs ************** */
// Launch a GRPC service for us as a Chord peer
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

// Connect to another node in chord ring
func connectToNode(node string) (pb.ChordClient, error) {
	backoffConfig := grpc.DefaultBackoffConfig
	// Choose an aggressive backoff strategy here.
	backoffConfig.MaxDelay = 500 * time.Millisecond
	conn, err := grpc.Dial(node, grpc.WithInsecure(), grpc.WithBackoffConfig(backoffConfig))
	// Ensure connection dId not fail, which should not happen since this happens
	//in the background
	if err != nil {
		return pb.NewChordClient(nil), err
	}
	return pb.NewChordClient(conn), nil
}

// FindSuccessorRequest: Argument for find successor method
type FindSuccessorRequest struct {
	arg      *pb.FindSuccessorArgs
	response chan pb.FindSuccessorRet
}

// FindSuccessor in chord ring
func (kord *Chord) FindSuccessor(ctx context.Context, arg *pb.FindSuccessorArgs) (*pb.FindSuccessorRet, error) {
	c := make(chan pb.FindSuccessorRet)
	kord.FindSuccessorChan <- FindSuccessorRequest{arg: arg, response: c}
	result := <-c
	return &result, nil
}

// NotifyRequesst: Argument for notify method
type NotifyRequest struct {
	arg      *pb.NotifyArgs
	response chan pb.NotifyRet
}

// Notify: Tell node that we believe we are that node's predecessor
func (kord *Chord) Notify(ctx context.Context, arg *pb.NotifyArgs) (*pb.NotifyRet, error) {
	c := make(chan pb.NotifyRet)
	kord.NotifyChan <- NotifyRequest{arg: arg, response: c}
	result := <-c
	return &result, nil
}

// PingRequest: Arguement for ping method
type PingRequest struct {
	arg      *pb.PingPredecessorArgs
	response chan pb.PingPredecessorRet
}

// PingPredecessor: Ensure that predecessor is still working
func (kord *Chord) PingPredecessor(ctx context.Context, arg *pb.PingPredecessorArgs) (*pb.PingPredecessorRet, error) {
	c := make(chan pb.PingPredecessorRet)
	kord.PingChan <- PingRequest{arg: arg, response: c}
	result := <-c
	return &result, nil
}

////////////////////////////////////////////////////////////////////////////////
/* *********** Other chord methods *********** */

// Join: Ask different node to find our successor so we can join the network
func (kord *Chord) Join(peerIp string) {
	// Connect to other ring node
	peer, err := connectToNode(peerIP)
	if err != nil {
		kord.findSuccessorResponseChan <- FindSucessorResponse{err: err}
	}
	joinReq := &pb.FindSuccessorArgs{Id: kord.Id}
	ret, fsErr := peer.FindSuccessor(context.Background(), joinReq)
	kord.findSuccessorResponseChan <- FindSucessorResponse{ret: ret, err: fsErr}
}

// ClosestPreceding: Find node closest (and before) provided id
func (kord *Chord) ClosestPreceding(id unint64) unint64 {
	for i := M - 1; i > 0; i-- {
		if between(kord.finger[i], kord.Id, id) {
			return kord.finger[i]
		}
	}
	return kord.id
}

// Stabilize: Called periodically; Verifies our successor and tells successor
//about us
func (kord *Chord) Stabilize() {

}

////////////////////////////////////////////////////////////////////////////////

/* *********** Primary method for called by chord/main.go *********** */
func runChord(fs *FileSystem, myIp string, myId uint64, port int, debug bool) {
	log.Printf("Chord ARGS: %v %v %v", myIp, myId, port)

	var chord Chord
	var fTable [M]unint64
	if debug {
		id_3001 := generateIdFromIP("127.0.0.1:3001")
		id_3003 := generateIdFromIP("127.0.0.1:3003")
		peerId := id_3001
		peerIp := "127.0.0.1:3001"
		if myId == id_3001 {
			peerId = id_3003
			peerIp = "127.0.0.1:3003"
		}
		// Connect to our other ring node
		conn, err := connectToNode(peerIp)
		if err != nil {
			log.Fatalf("Failed to connect to GRPC server %v", err)
		}
		otherNode := &RingNode{Ip: peerIp, conn: conn}
		addToRing(peerId, peerIp, &chord.ringMap)
		for i := 0; i < M; i++ {
			fTable[i] = peerId
		}
		chord = Chord{
			Id:                    myId,
			Ip:                    myIp,
			successor:             peerId,
			predecessor:           peerId,
			finger:                fTable,
			ringMap:               ringMap,
			PingChan:              make(chan PingRequest),
			pingResponseChan:      make(chan PingResponse),
			successorResponseChan: make(chan FindSucessorResponse)}
	} else {
		for i := 0; i < M; i++ {
			fTable[i] = myId
		}
		chord = Chord{
			Id:                    myId,
			Ip:                    myIp,
			successor:             myId,
			predecessor:					 myId,
			finger:                fTable,
			ringMap:               map[uint64]*RingNode{},
			PingChan:              make(chan PingRequest),
			JoinChan:              make(chan bool, 1),
			pingResponseChan:      make(chan PingResponse),
			successorResponseChan: make(chan FindSucessorResponse)}
		// Leave ourselves a message to try to join a network
		chord.JoinChan <- true
	}

	go RunChordServer(&chord, port)
	pingTimer := time.NewTimer(PingTimeout * time.Millisecond)

	/* *********** Forever loop to run our chord functionality *********** */
	for {
		select {
		// We left ourselves a message to join the network
		case jn := <-chord.JoinChan:
			log.Printf("Attempting to join network...")
			peerIp := fmt.Sprintf("127.0.0.1:%d", port+2)
			chord.Join(peerIp)

		// Find successor has returned result
		case fr := <-chord.findSuccessorResponseChan:
			if fs.err != nil {
				peerIp := fmt.Sprintf("127.0.0.1:%d", port-2)
				chord.Join(peerIp)
			} else {
				chord.successor = fs.ret.Successor
				chord.finger[0] = chord.succesor
			}

		// We received find successor request
		case fr := <-chord.FindSuccessorChan:
			if between(fs.arg.Id, chord.Id, chord.successor) {
				fs.response <- pb.FindSuccessorRet{SuccessorId: chord.successor, SuccessorIp: ringMap[successor].Ip}
			} else {
				closest = chord.ClosestPreceding(fs.arg.Id)
				fs.response <- pb.FindSuccessorId{SuccessorId: closest.Id, SuccessorIp: closest.Ip}
			}

		// We received notify request from node that believes it's our predecessor
		case nr := <-chord.NotifyChan:
			if chord.predecessor == nil
			|| between(nr.arg.PredecessorId, chord.predecessor, chord.Id) {
				chord.predecessor = nr.arg.PredecessorId
				addToRing(nr.arg.PredecessorId, nr.arg.PredecessorIp, &chord.ringMap)
				nr.response <- pb.NotifyRet{Updated: true}
			} else {
				nr.response <- pb.NotifyRet{Updated: false}
			}

		case pr := <-chord.pingResponseChan:
			if pr.err != nil {
				log.Printf("Ping failed")
			} else {
				// We've recived a ping response from just respond
				log.Printf("Got ping response!")
			}

		case ping := <-chord.PingChan:
			log.Print("ping from successor")
			ping.response <- pb.PingPredecessorRet{}

		case <-pingTimer.C:
			// We need to ping our predecessor
			log.Printf("My ip and Id: %v %v", chord.IPAddress, chord.Id)
			if chord.successor != nil {
				log.Printf("Attempting to ping %v", chord.predecessor.IPAddress)
				go func(pred pb.ChordClient, predIP string) {
					ret, err := pred.PingPredecessor(context.Background(), &pb.PingPredecessorArgs{})
					pingResponseChan <- PingResponse{ret: ret, err: err, predecessor: predIP}
				}(chord.predecessor.conn, chord.predecessor.IPAddress)
			} else {
				log.Print("No successor")
			}
			restartTimer(pingTimer, PingTimeout)
		}
	}
}
