package main

import (
	"log"
)

/* ************** Chord RPCs ************** */

/* **************************************** */

type Chord struct {
	ID         uint64
	IP_address string
	Hash       []byte
}

func chord(fs *FileSystem, ip string, id uint64, port int) {
	//TODO Add chord implementation
	log.Printf("Chord ARGS: %v %v %v", ip, id, port)
}
