package main

import (
	"fmt"
	"math/rand"
	"time"
)

const UINT64_MAX = ^uint64(0)

// Generates a random 32 bit IP seeded on system time
// Typically usage will be for local testing
func Generate_random_ip() string {
	rand.Seed(time.Now().UTC().UnixNano())
	return fmt.Sprintf("%d.%d.%d.%d", rand.Intn(256), rand.Intn(256),
		rand.Intn(256), rand.Intn(256))
}

// Converts a byte slize into a 64 bit number.
// This implementation will shift overflow if given a byte slice
// over 8 bytes.
func bytes_to_int64(bites []byte) (ret uint64) {

	ret = 0
	for _, bite := range bites {
		ret <<= 8
		ret += uint64(bite)
	}

	return ret
}

// Given an int64 this will truncate our hash id down to m bits for numerical IDs
// Since go's sha1 returns a 20 byte slice we can feed the sha1 hash converted
// to a 64 bit (8 bytes) number and truncate it to `m` using this function.
func truncate_bits(m int, base_64_rep uint64) uint64 {
	return base_64_rep & (UINT64_MAX >> (64 - uint64(m))) // Get the lower `m` bytes of an int64
}
