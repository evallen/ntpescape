package main

import (
	"context"
	"fmt"
	"net"

	"github.com/evallen/ntpescape/common"
)

func main() {
	for _, name := range common.NtpServerNames {
		ips, err := net.DefaultResolver.LookupIP(context.Background(), "ip4", name)
		if err != nil {
			continue
		}

		fmt.Printf("\"%v\",\n", ips[0])
	}
}
