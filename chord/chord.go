package main

import (
	"log"
)

var _ = log.Printf // TODO remove, just to get the linter to shut up

func chord(fs *FileSystem, ip string, id uint64, port int) {
	//TODO Add chord implementation
    log.Printf("Chord ARGS: %v %v %v", ip, id, port)
}
