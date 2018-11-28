package main

import (
	"crypto/sha1"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"
)

/* ************** Constants ************** */
// unint64MAX -> largest unsigned 64-bit int
const uint64MAX = ^uint64(0)

// PingTimeout -> milliseconds to ping a predecessor
const PingTimeout = 2000

// StabilizeTimeout -> milliseconds to run stabilize
const StabilizeTimeout = 5000

// PowTwo raises 2 to the power of x
func PowTwo(x uint64) uint64 {
	return uint64(math.Pow(2, float64(x)))
}

// GenerateRandomIP generates a random 32 bit IP seeded on system time
// Typically usage will be for local testing
func GenerateRandomIP() string {
	rand.Seed(time.Now().UTC().UnixNano())
	return fmt.Sprintf("%d.%d.%d.%d", rand.Intn(256), rand.Intn(256),
		rand.Intn(256), rand.Intn(256))
}

// Converts a byte slize into a 64 bit number.
// This implementation will shift overflow if given a byte slice
// over 8 bytes.
func bytesToInt64(bites []byte) (ret uint64) {
	ret = 0
	for _, bite := range bites {
		ret <<= 8
		ret += uint64(bite)
	}
	return ret
}

// Given an int64 this will truncate our hash Id down to m bits for numerical Ids
// Since go's sha1 returns a 20 byte slice we can feed the sha1 hash converted
// to a 64 bit (8 bytes) number and truncate it to `m` using this function.
func truncateBits(m int, base64Rep uint64) uint64 {
	return base64Rep & (uint64MAX >> (64 - uint64(m))) // Get the lower `m` bytes of an int64
}

// Between function that takes care of wrapping past 0.
// This will not succeed in cases where the key we are looking for is the successor
// To handle that case (i.e in find successors) we can just first check if the key == successor
func between(key, ID, successorID uint64, inclusive bool) bool {
	// If a high node's successor wraps the ring past 0
	if ID >= successorID { // We are wrapping
		return key > ID || key < successorID || (inclusive && key == successorID)
	}
	return (key > ID && key < successorID) || (inclusive && key == successorID)
}

func generateIDFromIP(IP string) uint64 {
	shaHash := sha1.New()
	shaHash.Write([]byte(IP))
	hashed := shaHash.Sum(nil)
	hashed64 := bytesToInt64(hashed[:8])
	ID := truncateBits(M, hashed64) % uint64(math.Pow(2, float64(M)))
	return ID
}

func addToRing(id uint64, ip string, ringMap map[uint64]*RingNode) {
	_, ok := ringMap[id]
	if !ok {
		conn, err := connectToNode(ip)
		if err != nil {
			log.Fatalf("Failed to connect to GRPC server %v", err)
		}
		newRingNode := &RingNode{IP: ip, conn: conn}
		ringMap[id] = newRingNode
	}
}

// Restart the supplied timer using a random timeout based on function above
func restartTimer(timer *time.Timer, timeout int64) {
	// Clear timer in case it was stopped prior to calling restart
	if !timer.Stop() {
		for len(timer.C) > 0 {
			<-timer.C
		}
	}
	timer.Reset(time.Duration(timeout) * time.Millisecond)
}

/* ********************  Color Printing ******************** */
// All color printing functions act as wrappers around strings
// and just insert ANSII escape sequences for colors.
// Example: green("My String") -> "\u001b[32mMystring\u001b[0m"

func green(s string) string {
	return fmt.Sprintf("\u001b[32m%s\u001b[0m", s)
}

func red(s string) string {
	return fmt.Sprintf("\u001b[31m%s\u001b[0m", s)
}

func yellow(s string) string {
	return fmt.Sprintf("\u001b[33m%s\u001b[0m", s)
}

func blue(s string) string {
	return fmt.Sprintf("\u001b[34m%s\u001b[0m", s)
}

func magenta(s string) string {
	return fmt.Sprintf("\u001b[35m%s\u001b[0m", s)
}

func cyan(s string) string {
	return fmt.Sprintf("\u001b[36m%s\u001b[0m", s)
}

/* ********************  Finger Table formatting ******************** */
func fingertoString(id uint64, ft *[M]uint64) string {
	retString := "|-----|-----|-----|\n|    i|  mod|  key|\n|-----|-----|-----|\n"
	twoToTheM := PowTwo(M)
	for i := 0; i < M; i++ {
		retString += fmt.Sprintf("|%5v|%5v|%5v|\n|-----|-----|-----|\n",
			i, (id+PowTwo(uint64(i)))%twoToTheM, ft[i])
	}
	return retString
}

/* ********************  Debug Printing ******************** */
func (kord Chord) String() string {
	retString := fmt.Sprintf("\n---------------------- Node %v ----------------------\nID: %v\nIP: %s\n",
		kord.ID, kord.ID, kord.IP)
	retString += fmt.Sprintf("Successor: %v\n", kord.successor)
	retString += fmt.Sprintf("Predecessor: %v\n", kord.predecessor)
	// Finger table TODO
	retString += fmt.Sprintf("Finger_table:\n%v", fingertoString(kord.ID, &kord.finger))
	return retString
}
