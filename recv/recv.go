package recv

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"

	"github.com/evallen/ntpescape/common"
)

var key = []byte{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa}

// Seconds
const rootDelayLow = 0.001
const rootDelayHigh = 0.002
const rootDispLow = 0.0001
const rootDispHigh = 0.0005
const rootInfoUpdateDelayLow = 10
const rootInfoUpdateDelayHigh = 20

var rootInfo = common.RootInfo{}
var rootInfoMutex sync.Mutex

var outwriter io.Writer = nil

// Randomly generate new root info.
//
// Locks / unlocks rootInfoMutex.
func updateRootInfo() {
	rootInfoMutex.Lock()

	rootInfo.RefTimeSec, rootInfo.RefTimeFrac = common.GetNTPTime(time.Now())

	rootServerIpStr := common.NtpServerIps[rand.Intn(len(common.NtpServerIps))]
	rootInfo.ReferenceID = binary.BigEndian.Uint32(net.ParseIP(rootServerIpStr).To4())

	rootDelayFloat := common.RandF64InRange(rootDelayLow, rootDelayHigh)
	rootInfo.RootDelay = common.ToNTPShortFormat(rootDelayFloat)

	rootDispFloat := common.RandF64InRange(rootDispLow, rootDispHigh)
	rootInfo.RootDispersion = common.ToNTPShortFormat(rootDispFloat)

	rootInfoMutex.Unlock()
}

// Goroutine to occasionally change the stored root info.
//
// The root info is the information that the receiver _would_ update
// if it were a real NTP server. This makes the responses more believeable.
func rootInfoDaemon() {
	for {
		updateRootInfo()

		nextDelay := common.RandF64InRange(rootInfoUpdateDelayLow, rootInfoUpdateDelayHigh)
		time.Sleep(time.Duration(nextDelay * 1e9))
	}
}

func Main() {
	listendest := flag.String("d", ":123", "dest: What host and port to listen to, like :123")
	outfile := flag.String("f", "", "file: File to also output results to")
	help := flag.Bool("h", false, "help: Show this help")
	flag.Parse()

	if *help {
		flag.Usage()
		return
	}

	if *outfile != "" {
		writer, err := os.Create(*outfile)
		if err != nil {
			log.Fatalf("Error opening out file %v: %v", *outfile, err)
		}
		outwriter = writer
	}

	udpaddr, err := net.ResolveUDPAddr("udp", *listendest)
	if err != nil {
		log.Fatalf("Error resolving UDP address: %v", err.Error())
	}

	conn, err := net.ListenUDP("udp", udpaddr)
	if err != nil {
		log.Fatalf("Error listening: " + err.Error())
	}
	defer conn.Close()

	go rootInfoDaemon()
	listenToPackets(conn)
}

// Continuously listen and process incoming packets.
func listenToPackets(conn *net.UDPConn) {
	var packet common.NTPPacket
	for {
		buf := make([]byte, 512)
		_, addr, err := conn.ReadFromUDP(buf)

		if err != nil {
			log.Println("Error reading UDP packet: " + err.Error())
			continue
		}

		err = binary.Read(bytes.NewBuffer(buf), binary.BigEndian, &packet)
		if err != nil {
			log.Println("Error reading UDP data into struct: " + err.Error())
			continue
		}

		err = processPacket(&packet, addr, conn)
		if err != nil {
			log.Printf("Error processing packet: %v\n", err)
		}
	}
}

// Decrypt and repsond to a given `packet`.
//
// Pass in the address `addr` it came from and the connection `conn` to use
// when responding.
func processPacket(packet *common.NTPPacket, addr *net.UDPAddr, conn *net.UDPConn) error {
	plaintext, err := packet.ReadPacketEncrypted(key)
	if err != nil {
		return fmt.Errorf("could not read encrypted packet: %v", err)
	}

	recordMessage(plaintext)

	err = sendResponsePkt(packet, addr, conn)
	if err != nil {
		return fmt.Errorf("could not send response packet: %v", err)
	}

	return nil
}

// Record a `message` received.
func recordMessage(message []byte) {
	if outwriter != nil {
		outwriter.Write(message)
	}

	fmt.Print(string(message))
}

// Send a response packet after receiving a client packet.
//
// Pass in the client `packet` to respond to, the address `raddrUdp` to send
// the response back to, and the connection `conn` to use.
func sendResponsePkt(packet *common.NTPPacket, raddrUdp *net.UDPAddr, conn *net.UDPConn) error {
	rootInfoMutex.Lock()
	responsePkt := packet.GenerateResponsePkt(&rootInfo)
	rootInfoMutex.Unlock()

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, responsePkt); err != nil {
		return fmt.Errorf("failed sending packet %v to %v: %v", packet, raddrUdp, err)
	}
	conn.WriteToUDP(buf.Bytes(), raddrUdp)

	return nil
}
