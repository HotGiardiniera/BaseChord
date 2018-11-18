package main

import (
	"crypto/sha1"
	"fmt"
	"math"
	"math/rand"
	"time"
)

/* ************** Constants ************** */
const uint64MAX = ^uint64(0)
const PingTimeout = 1000      // PingTimeout -> milliseconds to ping a predecessor
const StabilizeTimeout = 3000 // StabilizeTimeout -> milliseconds to run stabilize

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
func between(key, Id, successorId uint64) bool {
	// If a high node's successor wraps the ring past 0
	if Id >= successorId { // We are wrapping
		return (key > Id || key < successorId)
	}
	return key > Id && key < successorId
}

func generateIdFromIP(ip string) uint64 {
	shaHash := sha1.New()
	shaHash.Write([]byte(ip))
	hashed := shaHash.Sum(nil)
	hashed64 := bytesToInt64(hashed[:8])
	Id := truncateBits(M, hashed64) % uint64(math.Pow(2, float64(M)))
	return Id
}

func addToRing(id uint64, ip string, ringMap *map[uint64]*RingNode) {
	_, ok := *ringMap[id]
	if !ok {
		conn, err := connectToNode(ip)
		newRingNode = &RingNode{Ip: ip, conn: conn}
		*ringMap[uint64] = newRingNode
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
