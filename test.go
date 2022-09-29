package main

import (
	"encoding/binary"
	"fmt"

	"github.com/evallen/ntpescape/common"
)

func main() {
	packet := common.GenerateClientPkt()
	fmt.Printf("%v", packet.GetNonce())

	b := make([]byte, 4)

	binary.BigEndian.PutUint32(b, packet.TxTimeSec)
	fmt.Printf("%v", b)

	binary.BigEndian.PutUint32(b, packet.TxTimeFrac)

	fmt.Printf("%v", b)
}
