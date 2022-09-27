package recv

import (
	"log"
	"net"
	"encoding/binary"

	"github.com/evallen/ntpescape/common"
)

func Main() {
	udpaddr, _ := net.ResolveUDPAddr("udp", "localhost:123")

	conn, err := net.ListenUDP("udp", udpaddr)
	if err != nil {
		log.Fatalf("Error listening: " + err.Error())
	}
	defer conn.Close()

	var packet common.NTPPacket
	for {
		err := binary.Read(conn, binary.BigEndian, &packet)
		if err != nil {
			log.Fatalf("Error reading: " + err.Error())
		}
		log.Printf("%v", packet)

		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, packet.TxTimeFrac)
		log.Printf("%s", string(b[2:4]))
	}
}