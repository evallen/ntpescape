package recv

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/evallen/ntpescape/common"
)

var key = []byte{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa}

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

	listenToPackets(conn)
}

func listenToPackets(conn *net.UDPConn) {
	var packet common.NTPPacket
	for {
		err := binary.Read(conn, binary.BigEndian, &packet)
		if err != nil {
			log.Println("Error reading: " + err.Error())
			continue
		}

		processPacket(&packet)
	}
}

func processPacket(packet *common.NTPPacket) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, packet.TxTimeFrac)

	ciphertext := b[2:4]
	plaintext, err := common.Decrypt(ciphertext, packet.GetNonce(), key)
	if err != nil {
		return fmt.Errorf("could not decrypt ciphertext %v: %v", ciphertext, err)
	}

	fmt.Printf("%v", string(plaintext))

	return nil
}