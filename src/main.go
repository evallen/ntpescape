package main

import (
	"encoding/binary"
	"flag"
	"log"
	"net"
)

// Goal: Send an NTP packet to myself with the TX timestamp
// looking normal, but the final two bytes being 0a 0b
func main() {
	host := flag.String("h", "localhost:123", "host to send the NTP packets to")
	flag.Parse()

	conn, err := net.Dial("udp", *host)
	if err != nil {
		log.Fatalf("Failed to connect to %v", *host)
	}
	defer conn.Close()

	packet := GenerateClientPkt()
	if err := binary.Write(conn, binary.BigEndian, packet); err != nil {
		log.Fatalf("Failed sending packet %v to %v", packet, *host)
	}
}