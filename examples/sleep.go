package main

// sleep is demo executable that periodically outputs some text to stderr.

import (
	"log"
	"os"
	"time"
)

const version = "1.2.4"

func main() {
	for {
		time.Sleep(time.Second)
		log.Printf("Sleep %s %s", version, os.Args[1])
	}
}
