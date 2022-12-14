package common

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// NTP time starts at 00:00:00 1 Jan 1900 UTC, while
// Unix time starts at 00:00:00 1 Jan 1970 UTC.
//
// This calculates the number of seconds between those two times.
// There were 70*365 (+17 leap) days in that period, each with
// 24*60*60 seconds.
const secsSinceNTPEpoch = (70*365 + 17) * (24 * 60 * 60)

// NTP Packet format found from RFC 5905
// https://datatracker.ietf.org/doc/html/rfc5905#section-7
//
// Adapted from struct used in the following tutorial:
// https://medium.com/learning-the-go-programming-language/lets-make-an-ntp-client-in-go-287c4b9a969f
type NTPPacket struct {
	Flags          uint8
	Stratum        uint8
	Poll           uint8
	Precision      uint8
	RootDelay      uint32
	RootDispersion uint32
	ReferenceID    uint32
	RefTimeSec     uint32
	RefTimeFrac    uint32
	OrigTimeSec    uint32
	OrigTimeFrac   uint32
	RecvTimeSec    uint32
	RecvTimeFrac   uint32
	TxTimeSec      uint32
	TxTimeFrac     uint32
}

// Info related to a root NTP server.
// This stores anything that might change when our fake
// "NTP server" reaches out to a stratum 1 server to update
// itself.
type RootInfo struct {
	RootDelay      uint32
	RootDispersion uint32
	ReferenceID    uint32
	RefTimeSec     uint32
	RefTimeFrac    uint32
}

// Generate a basic, legitimate NTP client packet with the current
// NTP time filled in the transmit timestamp.
func GenerateClientPkt() *NTPPacket {
	// Flags:
	// 00 --------- Leap year (0: no warning)
	//    100 ----- Version (4)
	//        011 - Mode (3: client)
	flags := uint8(0x23)
	ntpSecs, ntpFrac := GetNTPTime(time.Now())

	packet := &NTPPacket{
		Flags:      flags,
		TxTimeSec:  ntpSecs,
		TxTimeFrac: ntpFrac,
	}

	return packet
}

// Gets the NTP time in the form (NTP seconds, NTP fraction)
// from a Go time.Time.
func GetNTPTime(now time.Time) (ntpSecs uint32, ntpFrac uint32) {
	unixSecs := uint32(now.Unix())
	nanoseconds := uint64(now.Nanosecond())

	ntpSecs = unixSecs + secsSinceNTPEpoch

	// The NTP fraction is a uint32 representing the
	// current fraction of a second where each increment
	// in its value represents 1/2^32 seconds. So:
	//
	// 		ntpFrac * (1/2^32) = nanoseconds / 1e9
	// => 	ntpFrac = nanoseconds * 2^32 / 1e9
	ntpFrac = uint32((nanoseconds << 32) / 1e9)

	return ntpSecs, ntpFrac
}

// Convert a float64 `time` representing seconds to
// a uint32 in NTP short format.
//
// NTP short format is defined in the NTP RFC as
// two 16-bit numbers concatenated together -
// a seconds segment, and a fractional segment.
func ToNTPShortFormat(time float64) (result uint32) {
	timeSecs, timeFrac := math.Modf(time)

	var shortSecs, shortFrac uint16
	shortSecs = uint16(timeSecs)

	// Convert floating fraction (0.0 <= timeFrac < 1.0)
	// to NTP short fraction, where each value is 1/2^16.
	//
	// 		shortFrac * (1/2^16) = timeFrac
	// =>	shortFrac = timeFrac * 2^16
	shortFrac = uint16(timeFrac * (1 << 16))

	result = 0
	result |= uint32(shortSecs) << 16
	result |= uint32(shortFrac)

	return result
}

// Generate a response packet from scratch given a set of root info to
// populate it with. Fills in the transmit timestamp of the receiver `packet`
// in the response packet's origin timestamp.
func (packet *NTPPacket) GenerateResponsePkt(rootInfo *RootInfo) *NTPPacket {
	newPacket := &NTPPacket{}

	// Do first upon "receive"
	newPacket.RecvTimeSec, newPacket.RecvTimeFrac = GetNTPTime(time.Now())

	// Flags:
	// 00 --------- Leap year (0: no warning)
	//    100 ----- Version (4)
	//        100 - Mode (4: server)
	newPacket.Flags = uint8(0x24)
	newPacket.Stratum = 2 // (secondary reference)
	newPacket.Poll = 3    // (invalid)
	newPacket.Precision = 0

	// Information about the last-queried Stratum 1 server
	// -- this might change as time goes on
	newPacket.RootDelay = rootInfo.RootDelay
	newPacket.RootDispersion = rootInfo.RootDispersion
	newPacket.ReferenceID = rootInfo.ReferenceID
	newPacket.RefTimeSec = rootInfo.RefTimeSec
	newPacket.RefTimeFrac = rootInfo.RefTimeFrac

	newPacket.OrigTimeSec = packet.TxTimeSec
	newPacket.OrigTimeFrac = packet.TxTimeFrac

	// Do last upon "transmit"
	newPacket.TxTimeSec, newPacket.TxTimeFrac = GetNTPTime(time.Now())

	return newPacket
}

// Patch an existing packet so that it includes a given 2-byte (16-bit)
// message in the last two bytes of the transmit timestamp fraction,
// unencrypted.
func (packet *NTPPacket) PatchPacketUnencrypted(message []byte) error {
	if len(message) > 2 || len(message) < 1 {
		return fmt.Errorf("invalid message length %v", len(message))
	}

	// Initialize to zeroes so that if the message is only one byte, the final byte
	// is just 0x00.
	plaintext := []byte{0, 0}
	copy(plaintext, message)

	packet.TxTimeFrac &^= 0xFFFF // Clear bottom two bytes
	packet.TxTimeFrac |= uint32(binary.BigEndian.Uint16(plaintext))
	packet.setLengthBitUnencrypted(len(message))

	return nil
}

// Patch an existing packet so that it includes a given 1 or 2-byte
// message in the last two bytes of the transmit timestamp fraction,
// encrypted.
//
// See (*NTPPacket).GetNonce() for the details on the nonce used.
func (packet *NTPPacket) PatchPacketEncrypted(message []byte, key []byte) error {
	if len(message) > 2 || len(message) < 1 {
		return fmt.Errorf("invalid message length %v", len(message))
	}

	// Initialize to zeroes so that if the message is only one byte, the final byte
	// is just 0x00.
	plaintext := []byte{0, 0}
	copy(plaintext, message)

	ciphertext, err := Encrypt(plaintext, packet.GetNonce(), key)
	if err != nil {
		return fmt.Errorf("couldn't encrypt message %v: %v", plaintext, err.Error())
	}

	packet.TxTimeFrac &^= 0xFFFF // Clear bottom two bytes
	packet.TxTimeFrac |= uint32(binary.BigEndian.Uint16(ciphertext))

	packet.setLengthBitEncrypted(len(message), key)

	return nil
}

// Sets the length bit in encrypted mode.
//
// The length bit signals whether or not the message
// should be interpreted as one or two bytes.
//
// In encrypted mode, the bit is set to the last bit
// of Encrypt(0x00, packet.GetNonce(), key) if the
// message is two bytes or the opposite if the message
// is one byte. We do this so that the length bit
// appears statistically random and cannot be detected
// by an IDS as being abnormally biased towards 1 or 0.
// It is only interpretable by someone with the key.
func (packet *NTPPacket) setLengthBitEncrypted(length int, key []byte) error {
	zeroEncrypted, _ := Encrypt([]byte{0x00}, packet.GetNonce(), key)

	lastBit := zeroEncrypted[0] & 1
	lastBitBool := lastBit == 1 // Convert 1 -> true, 0 -> false

	if length == 2 {
		packet.setLengthBitRaw(lastBitBool)
	} else if length == 1 {
		packet.setLengthBitRaw(!lastBitBool)
	} else {
		return fmt.Errorf("invalid length %v", length)
	}
	
	return nil
}

// Sets the length bit in unencrypted mode.
//
// The length bit signals whether or not the message
// should be interpreted as one or two bytes.
//
// In unencrypted mode, the length bit being '1' signals
// a message of two bytes and a length bit being '0' signals
// a message of one byte.
func (packet *NTPPacket) setLengthBitUnencrypted(length int) error {
	if length == 2 {
		packet.setLengthBitRaw(true)
	} else if length == 1 {
		packet.setLengthBitRaw(false)
	} else {
		return fmt.Errorf("invalid length %v", length)
	}

	return nil
}

// Actually modify the length bit of the packet.
// `bit` == true represents 1,
// `bit` == false represents 0.
func (packet *NTPPacket) setLengthBitRaw(bit bool) {
	if bit {
		packet.TxTimeFrac |= 0x00010000
	} else {
		packet.TxTimeFrac &^= 0x00010000
	}
}

// Interpret the length bit of an encrypted packet.
//
// Returns the length of the message in bytes:
// 2 if the length bit matches the bottom bit of 
// Encrypt([]byte{0x00}, packet.GetNonce(), key), or 1
// if not.
func (packet *NTPPacket) readLengthBitEncrypted(key []byte) int {
	zeroEncrypted, _ := Encrypt([]byte{0x00}, packet.GetNonce(), key)

	lastBit := zeroEncrypted[0] & 1
	lastBitBool := lastBit == 1 // Convert 1 -> true, 0 -> false

	lengthBit := packet.readLengthBitRaw()
	
	if lengthBit == lastBitBool {
		return 2
	} else {
		return 1
	}
}

// Interpret the length bit of an unencrypted packet.
//
// Returns the length of the message in bytes:
// 2 if the length bit is set, and 1 if not.
func (packet *NTPPacket) readLengthBitUnencrypted() int {
	bit := packet.readLengthBitRaw()
	if bit {
		return 2
	} else {
		return 1
	}
}

// Get the literal value of the length bit of the 
// packet.
// true represents 1, false represents 0.
func (packet *NTPPacket) readLengthBitRaw() bool {
	return (packet.TxTimeFrac & 0x00010000) > 0
}

// Read an unencrypted client packet and get the message
// out as a []byte slice.
func (packet *NTPPacket) ReadPacketUnencrypted() []byte {
	message := make([]byte, 4)
	binary.BigEndian.PutUint32(message, packet.TxTimeFrac)

	// We only care about the bottom two bytes for the actual message
	message = message[2:4]

	if packet.readLengthBitUnencrypted() == 2 {
		return message
	} else {
		return message[:1]
	}
}

// Read an encrypted client packet and get the message
// out as a []byte slice.
func (packet *NTPPacket) ReadPacketEncrypted(key []byte) ([]byte, error) {
	message := make([]byte, 4)
	binary.BigEndian.PutUint32(message, packet.TxTimeFrac)

	// We only care about the bottom two bytes for the actual message
	ciphertext := message[2:4]

	plaintext, err := Decrypt(ciphertext, packet.GetNonce(), key)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt ciphertext %v: %v", ciphertext, err)
	}

	if packet.readLengthBitEncrypted(key) == 2 {
		return plaintext, nil
	} else {
		return plaintext[:1], nil
	}
}

// Get the nonce from a packet's Transmitted Timestamp.
//
// The nonce does not include information from the bottom
// two bytes of the TxTimeFrac since that is overwritten
// with our encrypted data.
//
// The nonce is 16 bytes:
//
//	00 01 02 03 04 05 06 07 08 09 10 11 12 13 14 15
//	\_________/ \___/ \___________________________/
//	     |        |                        |
//	 TxTimeSec  TxTimeFrac[0:2] &^ 1       Zeroes
//
// The first four bytes are the transmit timestamp
// seconds bytes; the next two bytes are the most significant
// two bytes of the transmit timestamp fraction WITH the bottom
// bit zeroed out. This is because we use that bottom bit
// as a flag about whether the message contains 1 or 2 bytes.
//
// The next bytes are just zeroes. This is still a valid nonce
// because the nonce will not be repeated for 2^32 seconds =
// 136 years as long as packets are not sent too quickly.
func (packet *NTPPacket) GetNonce() (nonce []byte) {
	nonce = make([]byte, 16)
	binary.BigEndian.PutUint32(nonce, packet.TxTimeSec)
	binary.BigEndian.PutUint32(nonce[4:], packet.TxTimeFrac&^0x1FFFF)
	binary.BigEndian.PutUint64(nonce[8:], 0)

	return nonce
}

// List of all the Stratum 1 NTP IPs that the receiver might "contact" for an update.
var NtpServerIps = []string{
	"216.239.35.12",
	"216.239.35.0",
	"216.239.35.4",
	"216.239.35.8",
	"216.239.35.12",
	"34.220.201.22",
	"69.89.207.99",
	"204.2.134.163",
	"192.46.215.60",
	"73.239.136.185",
	"81.21.76.27",
	"95.182.219.178",
	"185.224.145.68",
	"89.238.136.135",
	"185.103.117.60",
	"152.70.69.232",
	"162.159.200.1",
	"46.19.96.19",
	"59.103.236.10",
	"23.106.249.200",
	"91.207.136.55",
	"185.209.85.222",
	"194.190.168.1",
	"195.58.1.117",
	"213.234.203.30",
	"65.100.46.164",
	"64.79.100.196",
	"185.117.82.71",
	"143.107.229.210",
	"129.250.35.250",
	"216.218.254.202",
	"45.33.65.68",
	"147.135.201.174",
	"129.146.193.200",
	"45.55.58.103",
	"62.101.228.30",
	"108.61.73.244",
	"38.229.52.9",
	"162.159.200.1",
	"74.6.168.73",
	"45.63.54.13",
	"13.55.50.68",
	"137.190.2.4",
	"194.58.205.148",
	"209.126.83.42",
	"104.131.139.195",
	"198.199.14.18",
	"52.42.72.58",
	"44.4.53.6",
	"91.209.24.19",
	"137.184.81.69",
	"212.83.158.83",
	"50.205.244.37",
	"38.229.56.9",
	"72.14.183.239",
	"36.91.114.86",
	"110.170.126.102",
	"193.47.147.20",
	"23.92.64.226",
	"154.51.12.220",
	"91.198.87.118",
	"185.216.231.84",
	"142.202.190.19",
	"144.172.118.20",
	"192.48.105.15",
	"45.125.1.20",
	"103.242.70.4",
	"50.205.244.110",
	"72.14.183.39",
	"38.100.216.142",
	"162.159.200.123",
	"45.79.51.42",
	"108.62.122.57",
	"178.63.52.31",
	"104.156.229.103",
	"51.195.120.107",
	"91.206.8.36",
	"108.61.73.243",
	"104.131.155.175",
	"50.205.244.110",
	"51.15.175.180",
	"212.18.3.18",
	"69.164.213.136",
	"104.131.155.175",
	"50.205.244.37",
	"41.175.51.165",
	"72.30.35.89",
	"171.66.97.126",
	"78.153.129.227",
}
