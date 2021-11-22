package logstash

import (
	"fmt"
	"log"
	"net"
)

//sends to logstash ip
func Send(ip, data string, port int) {
	address := fmt.Sprintf("%v:%v", ip, port)
	conn, err := net.Dial("udp", address)
	if err != nil {
		log.Printf("sending to logstash error %v", err)
		return
	}
	fmt.Fprintf(conn, data)
	conn.Close()
}
