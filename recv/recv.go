package recv

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
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

func rootInfoDaemon() {
	for {
		updateRootInfo()

		nextDelay := common.RandF64InRange(rootInfoUpdateDelayLow, rootInfoUpdateDelayHigh)
		time.Sleep(time.Duration(nextDelay * 1e9))
	}
}

func Main() {
	listenhost := flag.String("h", ":123", "host: What host and port to listen to, like :123")

	udpaddr, err := net.ResolveUDPAddr("udp", *listenhost)
	if err != nil {
		log.Fatalf("Error resolving UDP address %v: %v", udpaddr, err.Error())
	}

	conn, err := net.ListenUDP("udp", udpaddr)
	if err != nil {
		log.Fatalf("Error listening: " + err.Error())
	}
	defer conn.Close()

	go rootInfoDaemon()
	listenToPackets(conn)
}

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

func processPacket(packet *common.NTPPacket, addr *net.UDPAddr, conn *net.UDPConn) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, packet.TxTimeFrac)

	ciphertext := b[2:4]
	plaintext, err := common.Decrypt(ciphertext, packet.GetNonce(), key)
	if err != nil {
		return fmt.Errorf("could not decrypt ciphertext %v: %v", ciphertext, err)
	}

	fmt.Printf("%v", string(plaintext))

	err = sendResponsePkt(packet, addr, conn)
	if err != nil {
		return fmt.Errorf("could not send response packet: %v", err)
	}

	return nil
}

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
