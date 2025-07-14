package main

import (
	"fmt"
	"net"
	"os"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	fmt.Println("connected")

	for {
		msg := make([]byte, 256)

		_, err := conn.Read(msg)
		if err != nil {
			fmt.Println("Error reading from connection: ", err.Error())
			continue
		}

		conn.Write([]byte("+PONG\r\n"))
	}

	fmt.Println("closing program..")
}
