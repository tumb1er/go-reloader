package main

import (
	"log"
	"os"
	"time"
)

func main() {
	for {
		time.Sleep(time.Second)
		log.Printf("Sleep1 %s", os.Args[1])
	}
}
