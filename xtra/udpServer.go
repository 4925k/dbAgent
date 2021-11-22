package main

import (
	"fmt"
	"net"
)

func main() {
	p := make([]byte, 2048)
	addr := net.UDPAddr{
		Port: 1234,
		IP:   net.ParseIP("192.168.10.64"),
	}
	ser, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Printf("Some error %v\n", err)
		return
	}
	for {
		_, _, err := ser.ReadFromUDP(p)
		fmt.Print(string(p))
		if err != nil {
			fmt.Printf("Some error  %v", err)
			continue
		}
	}
}
