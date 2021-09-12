package main

import (
	"flag"
	"log"
)

func main() {
	id := flag.String("id", "", "the tunnel id to pull")
	destination := flag.String("destination", "", "the rtmp url of the destination to pull to");

	flag.Parse()

	log.Printf("id: %s, destination: %s", *id, *destination)
}