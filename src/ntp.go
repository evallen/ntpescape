package main

import (
	"time"
)

// NTP time starts at 00:00:00 1 Jan 1900 UTC, while
// Unix time starts at 00:00:00 1 Jan 1970 UTC.
// 
// This calculates the number of seconds between those two timeslots.
// There were 70*365 (+17 leap) days in that period, each with
// 24*60*60 seconds.
const secsSinceNTPEpoch = (70*365 + 17) * (24*60*60)

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

func GenerateClientPkt() *NTPPacket {
	// Flags:
	// 00 --------- Leap year (0: no warning)
	//    100 ----- Version (4)
	//        011 - Mode (3: client)
	flags := uint8(0x23)
	ntpSecs, ntpFrac := GetNTPTime(time.Now())

	packet := &NTPPacket{
		Flags: flags,
		TxTimeSec: ntpSecs,
		TxTimeFrac: ntpFrac,
	}

	return packet
}

// Gets the NTP time in the form (NTP seconds, NTP fraction)
// from a Go time.Time.
func GetNTPTime(now time.Time) (uint32, uint32) {
	unixSecs := uint32(now.Unix())
	nanoseconds := uint64(now.Nanosecond())
	

	ntpSecs := unixSecs + secsSinceNTPEpoch

	// The NTP fraction is a uint32 representing the
	// current fraction of a second where each increment
	// in its value represents 1/2^32 seconds. So:
	// 
	// 		ntpFrac * (1/2^32) = nanoseconds / 1e9
	// => 	ntpFrac = nanoseconds * 2^32 / 1e9
	ntpFrac := uint32((nanoseconds << 32) / 1e9)

	return ntpSecs, ntpFrac
}