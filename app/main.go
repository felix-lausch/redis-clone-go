package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
)

var cm = &ConcurrentMap{
	db: make(map[string]StoredValue),
}

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	fmt.Println("Listening..")
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		fmt.Println("client connected")
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		command, err := parseResp(reader)
		if err == io.EOF {
			fmt.Println("client disconnected")
			return
		} else if err != nil {
			fmt.Println("Error reading from connection: ", err.Error())
			continue
		}

		response, err := handleCommand(command)
		if err != nil {
			conn.Write(fmt.Appendf(nil, "-ERROR %v\r\n", err))
		}

		conn.Write(response)
	}
}

func handleCommand(command *Command) ([]byte, error) {
	switch command.Name {
	case "PING":
		return ping()
	case "ECHO":
		return echo(command.Args)
	case "SET":
		return set(command.Args)
	case "GET":
		return get(command.Args)
	default:
		return nil, fmt.Errorf("unkown command '%v'", command)
	}
}
