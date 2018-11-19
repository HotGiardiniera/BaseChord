package main

import (
	"fmt"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"net"
	"time"

	"github.com/nyu-distributed-systems-fa18/BaseChord/pb"
)

/* ************** Chord structs ************** */

// PingPredecessorResponse is response object to give a ping response channel
type PingPredecessorResponse struct {
	ret *pb.PingPredecessorRet
	err error
}

// PingSuccessorResponse is response object to give a ping response channel
type PingSuccessorResponse struct {
	ret *pb.PingSuccessorRet
	err error
}

// FindSuccessorResponse -> response object to give a find successor response channel
type FindSuccessorResponse struct {
	ret     *pb.FindSuccessorRet
	err     error
	forJoin bool
}

// RingNode -> represents a ring node that our node connects to
type RingNode struct {
	IP   string
	conn pb.ChordClient // Conenction to a ring node
}

// Chord -> object that represets our Node on the ring abstractly
type Chord struct {
	ID          uint64
	IP          string
	successor   uint64
	predecessor uint64
	ringMap     map[uint64]*RingNode
	finger      [M]uint64
	next        uint64
	// Request channels
	JoinChan                chan bool
	FindSuccessorChan       chan FindSuccessorRequest
	NotifyChan              chan NotifyRequest
	PingFromSuccessorChan   chan PingPredecessorRequest
	PingFromPredecessorChan chan PingSuccessorRequest
	// Response channels
	pingPredecessorResponseChan chan PingPredecessorResponse
	pingSuccessorResponseChan   chan PingSuccessorResponse
	findSuccessorResponseChan   chan FindSuccessorResponse
	fixFingersResponseChan      chan FindSuccessorResponse
	// Timers
	pingTimer      *time.Timer
	stabilizeTimer *time.Timer
}

////////////////////////////////////////////////////////////////////////////////

/* ************** Chord RPCs ************** */

// RunChordServer Launch a GRPC service for us as a Chord peer
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

// FindSuccessorRequest is argument for find successor method
type FindSuccessorRequest struct {
	arg      *pb.FindSuccessorArgs
	response chan pb.FindSuccessorRet
}

// FindSuccessorRPC in chord ring
func (kord *Chord) FindSuccessorRPC(ctx context.Context, arg *pb.FindSuccessorArgs) (*pb.FindSuccessorRet, error) {
	c := make(chan pb.FindSuccessorRet)
	kord.FindSuccessorChan <- FindSuccessorRequest{arg: arg, response: c}
	result := <-c
	return &result, nil
}

// NotifyRequest is argument for notify method
type NotifyRequest struct {
	arg      *pb.NotifyArgs
	response chan pb.NotifyRet
}

// NotifyRPC node that we believe we are that node's predecessor
func (kord *Chord) NotifyRPC(ctx context.Context, arg *pb.NotifyArgs) (*pb.NotifyRet, error) {
	c := make(chan pb.NotifyRet)
	kord.NotifyChan <- NotifyRequest{arg: arg, response: c}
	result := <-c
	return &result, nil
}

// PingPredecessorRequest is arguement for ping predecessor method
type PingPredecessorRequest struct {
	arg      *pb.PingPredecessorArgs
	response chan pb.PingPredecessorRet
}

// PingPredecessorRPC ensures that predecessor is still working
func (kord *Chord) PingPredecessorRPC(ctx context.Context, arg *pb.PingPredecessorArgs) (*pb.PingPredecessorRet, error) {
	c := make(chan pb.PingPredecessorRet)
	kord.PingFromSuccessorChan <- PingPredecessorRequest{arg: arg, response: c}
	result := <-c
	return &result, nil
}

// PingSuccessorRequest is arguement for ping succesor method
type PingSuccessorRequest struct {
	arg      *pb.PingSuccessorArgs
	response chan pb.PingSuccessorRet
}

// PingSuccessorRPC asks successor who its predecessor is
func (kord *Chord) PingSuccessorRPC(ctx context.Context, arg *pb.PingSuccessorArgs) (*pb.PingSuccessorRet, error) {
	c := make(chan pb.PingSuccessorRet)
	kord.PingFromPredecessorChan <- PingSuccessorRequest{arg: arg, response: c}
	result := <-c
	return &result, nil
}

////////////////////////////////////////////////////////////////////////////////
/* *********** Internal chord methods *********** */

// JoinInternal asks different node to find our successor so we can join the network
func (kord *Chord) JoinInternal(peerIP string) {
	// Connect to other ring node
	conn, err := connectToNode(peerIP)
	if err != nil {
		log.Printf("Could not connect to peer: %v", err)
		kord.findSuccessorResponseChan <- FindSuccessorResponse{err: err, forJoin: true}
	} else {
		joinReq := &pb.FindSuccessorArgs{Id: kord.ID}
		log.Printf("Making RPC to %v", peerIP)
		ret, fsErr := conn.FindSuccessorRPC(context.Background(), joinReq)
		kord.findSuccessorResponseChan <- FindSuccessorResponse{ret: ret, err: fsErr, forJoin: true}
	}
}

// ClosestPrecedingInternal finds node closest (and before) provided id
func (kord *Chord) ClosestPrecedingInternal(id uint64) uint64 {
	for i := M - 1; i > 0; i-- {
		if between(kord.finger[i], kord.ID, id, false) {
			return kord.finger[i]
		}
	}
	return kord.ID
}

// StabilizeInternal is called periodically; Verifies our successor and tells successor
//about us
func (kord *Chord) StabilizeInternal(successor pb.ChordClient) {
	pingReq := &pb.PingSuccessorArgs{}
	log.Printf("Asking our successor for its predecessor.")
	ret, err := successor.PingSuccessorRPC(context.Background(), pingReq)
	kord.pingSuccessorResponseChan <- PingSuccessorResponse{ret: ret, err: err}
}

// FixFingersInternal is called periodically (with stabilize); Refreshes finger table
func (kord *Chord) FixFingersInternal() {
	kord.next = (kord.next + 1) % M
	nextEntry := (kord.ID + PowTwo(kord.next)) % PowTwo(M)
	log.Printf("Looking for successor of finger table index %v. Finding successor for key %v", kord.next+1, nextEntry)
	ret := kord.FindSuccessorInternal(nextEntry)
	go func() { kord.fixFingersResponseChan <- FindSuccessorResponse{ret: ret, err: nil, forJoin: false} }()
}

// FindSuccessorInternal implements find successor at our node
func (kord *Chord) FindSuccessorInternal(id uint64) *pb.FindSuccessorRet {
	if between(id, kord.ID, kord.successor, true) {
		log.Printf("Successor found: %v:%v", kord.successor, kord.ringMap[kord.successor].IP)
		return &pb.FindSuccessorRet{SuccessorId: kord.successor, SuccessorIp: kord.ringMap[kord.successor].IP}
	}
	closest := kord.ClosestPrecedingInternal(id)
	log.Printf("Successor found: %v:%v", closest, kord.ringMap[closest].IP)
	return &pb.FindSuccessorRet{SuccessorId: closest, SuccessorIp: kord.ringMap[closest].IP}
}

// NotifyInternal implements the notify behavior at our Node
func (kord *Chord) NotifyInternal(id uint64, ip string) *pb.NotifyRet {
	// Nothing to update: node that notified us is already our predecessor
	if kord.predecessor == id {
		return &pb.NotifyRet{Updated: false}
	}
	// Node that notified us is indeed our predecessor
	if kord.predecessor == kord.ID || between(id, kord.predecessor, kord.ID, false) {
		log.Printf("Updating our predecessor: %v:%v", id, ip)
		kord.predecessor = id
		addToRing(id, ip, kord.ringMap)
		restartTimer(kord.pingTimer, PingTimeout)
		if kord.ID == kord.successor {
			kord.successor = kord.predecessor
			restartTimer(kord.stabilizeTimer, StabilizeTimeout)
		}
		return &pb.NotifyRet{Updated: true}
	}
	// Node that notified us, is not our predecessor
	return &pb.NotifyRet{Updated: false}
}

// PingPredecessorInternal makes RPC call at our node
func (kord *Chord) PingPredecessorInternal(predecessor pb.ChordClient) {
	ret, err := predecessor.PingPredecessorRPC(context.Background(), &pb.PingPredecessorArgs{})
	kord.pingPredecessorResponseChan <- PingPredecessorResponse{ret: ret, err: err}
}

////////////////////////////////////////////////////////////////////////////////

/* *********** Primary method for called by chord/main.go *********** */

func runChord(fs *FileSystem, myIP string, myID uint64, port int, debug bool) {
	log.Printf("Chord ARGS: %v %v %v. Ring size 2^%v", myIP, myID, port, M)

	var chord Chord
	var fTable [M]uint64
	rM := map[uint64]*RingNode{myID: &RingNode{IP: myIP}}
	if debug {
		peerID := generateIDFromIP("127.0.0.1:3001")
		peerIP := "127.0.0.1:3001"
		if myID == peerID {
			peerID = generateIDFromIP("127.0.0.1:3003")
			peerIP = "127.0.0.1:3003"
		}
		// Connect to our other ring node
		addToRing(peerID, peerIP, rM)
		for i := 0; i < M; i++ {
			fTable[i] = peerID
		}
		chord = Chord{
			ID:                          myID,
			IP:                          myIP,
			successor:                   peerID,
			predecessor:                 peerID,
			ringMap:                     rM,
			finger:                      fTable,
			next:                        0,
			JoinChan:                    make(chan bool),
			FindSuccessorChan:           make(chan FindSuccessorRequest),
			NotifyChan:                  make(chan NotifyRequest),
			PingFromSuccessorChan:       make(chan PingPredecessorRequest),
			PingFromPredecessorChan:     make(chan PingSuccessorRequest),
			pingPredecessorResponseChan: make(chan PingPredecessorResponse),
			pingSuccessorResponseChan:   make(chan PingSuccessorResponse),
			findSuccessorResponseChan:   make(chan FindSuccessorResponse),
			fixFingersResponseChan:      make(chan FindSuccessorResponse),
			pingTimer:                   time.NewTimer(PingTimeout * time.Millisecond),
			stabilizeTimer:              time.NewTimer(StabilizeTimeout * time.Millisecond)}
	} else {
		for i := 0; i < M; i++ {
			fTable[i] = myID
		}
		chord = Chord{
			ID:                          myID,
			IP:                          myIP,
			successor:                   myID,
			predecessor:                 myID,
			ringMap:                     rM,
			finger:                      fTable,
			next:                        0,
			JoinChan:                    make(chan bool, 1),
			FindSuccessorChan:           make(chan FindSuccessorRequest),
			NotifyChan:                  make(chan NotifyRequest),
			PingFromSuccessorChan:       make(chan PingPredecessorRequest),
			PingFromPredecessorChan:     make(chan PingSuccessorRequest),
			pingPredecessorResponseChan: make(chan PingPredecessorResponse),
			pingSuccessorResponseChan:   make(chan PingSuccessorResponse),
			findSuccessorResponseChan:   make(chan FindSuccessorResponse),
			fixFingersResponseChan:      make(chan FindSuccessorResponse),
			pingTimer:                   time.NewTimer(5000 * time.Millisecond),
			stabilizeTimer:              time.NewTimer(5000 * time.Millisecond)}

		if port == 3001 {
			chord.JoinChan <- true // Leave ourselves a message to join network
		}
	}

	go RunChordServer(&chord, port)

	/* *********** Forever loop to run our chord functionality *********** */
	for {
		select {
		// We left ourselves a message to join the network
		case <-chord.JoinChan:
			log.Printf("Attempting to join network...")
			peerIP := fmt.Sprintf("127.0.0.1:%d", port+2)
			go chord.JoinInternal(peerIP)

		// We received client request
		case op <- fs.C:
			log.Printf("Received command from client to %v data key %v", op.command.Operation, op.command.Arg)
			// If we are not connected to the ring yet, defer this command until later
			if chord.successor == myID || chord.predecessor == myID {
				log.Printf("Not connected to ring yet. Deferring command to later")
				fs.C <- op
			} else {
				fs.HandleCommand(op)
			}

		// Find successor has returned result
		case fsRes := <-chord.findSuccessorResponseChan:
			if fsRes.err != nil && fsRes.forJoin {
				log.Fatalf("Could not join network. Err: %v", fsRes.err)
			} else {
				log.Printf("Our successor: %v:%v", fsRes.ret.SuccessorId, fsRes.ret.SuccessorIp)
				chord.successor = fsRes.ret.SuccessorId
				chord.finger[0] = chord.successor
				addToRing(fsRes.ret.SuccessorId, fsRes.ret.SuccessorIp, chord.ringMap)
				if fsRes.forJoin {
					chord.predecessor = fsRes.ret.SuccessorId
					// Notify our successor we've joined the network
					notifyReq := &pb.NotifyArgs{PredecessorId: myID, PredecessorIp: myIP}
					go chord.ringMap[chord.successor].conn.NotifyRPC(context.Background(), notifyReq)
					// Restart timers with actual timeouts
					restartTimer(chord.pingTimer, PingTimeout)
					restartTimer(chord.stabilizeTimer, StabilizeTimeout)
				}
			}

		// We received fix fingers response back from ourselves
		case ff := <-chord.fixFingersResponseChan:
			chord.finger[chord.next] = ff.ret.SuccessorId
			addToRing(ff.ret.SuccessorId, ff.ret.SuccessorIp, chord.ringMap)

		// We received find successor request
		case fsReq := <-chord.FindSuccessorChan:
			log.Printf("We received request to find successor for key %v", fsReq.arg.Id)
			fsReq.response <- *(chord.FindSuccessorInternal(fsReq.arg.Id))

		// We received notification from node that believes it's our predecessor
		case nr := <-chord.NotifyChan:
			log.Printf("Received notify from potential predecessor. %v:%v", nr.arg.PredecessorId, nr.arg.PredecessorIp)
			nr.response <- *(chord.NotifyInternal(nr.arg.PredecessorId, nr.arg.PredecessorIp))

		// We received ping from our successor to make sure we're still online
		case ping := <-chord.PingFromSuccessorChan:
			log.Printf("Received ping from successor")
			ping.response <- pb.PingPredecessorRet{}

		// We received ping from a node behind us asking us for our predecessor's id
		case ping := <-chord.PingFromPredecessorChan:
			log.Printf("Ping from node behind us in the ring. Responding: %v:%v", chord.predecessor, chord.ringMap[chord.predecessor].IP)
			ping.response <- pb.PingSuccessorRet{PredecessorId: chord.predecessor,
				PredecessorIp: chord.ringMap[chord.predecessor].IP}

		// We received a response from pinging our precedessor
		case pr := <-chord.pingPredecessorResponseChan:
			if pr.err != nil {
				log.Printf("Ping failed!")
			} else {
				// We've recived a ping response from just respond
				log.Printf("Got ping response!")
			}

		// We received a response from asking our successor who its predecessor is
		//(This is part of the stabilize protocol)
		case pr := <-chord.pingSuccessorResponseChan:
			if pr.err != nil {
				log.Printf("Failed to ping our successor!")
			} else {
				if between(pr.ret.PredecessorId, chord.ID, chord.successor, false) {
					log.Printf("Updating successor: %v:%v", pr.ret.PredecessorId, pr.ret.PredecessorIp)
					chord.successor = pr.ret.PredecessorId
					addToRing(pr.ret.PredecessorId, pr.ret.PredecessorIp, chord.ringMap)
				}
				// Do not need to notify here, b/c we've already updated our successor
				// notifyReq := &pb.NotifyArgs{PredecessorId: myID, PredecessorIp: myIP}
				// go chord.ringMap[chord.successor].conn.NotifyRPC(context.Background(), notifyReq)
			}

		// Stabilize timer went off
		case <-chord.stabilizeTimer.C:
			log.Printf("Running stabilize & fix fingers protocols...")
			succ := chord.ringMap[chord.successor].conn
			go chord.StabilizeInternal(succ)
			chord.FixFingersInternal()
			restartTimer(chord.stabilizeTimer, StabilizeTimeout)

		// Ping predecessor timer went off
		case <-chord.pingTimer.C:
			if chord.successor != uint64MAX {
				log.Printf("Pinging our predecessor %v:%v", chord.predecessor, chord.ringMap[chord.predecessor].IP)
				pred := chord.ringMap[chord.predecessor].conn
				go chord.PingPredecessorInternal(pred)
			} else {
				log.Printf("No successor")
			}
			restartTimer(chord.pingTimer, PingTimeout)
		}
	}
}
