package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"redis-clone-go/app/commands"
	"redis-clone-go/app/protocol"
)

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
		command, err := protocol.ParseCommand(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("client disconnected")
				return
			}

			fmt.Println("Error reading from connection: ", err.Error())
			continue
		}

		response, err := handleCommand(command)
		if err != nil {
			conn.Write(protocol.FormatError(err))
		}

		conn.Write(response)
	}
}

func handleCommand(command *protocol.Command) ([]byte, error) {
	switch command.Name {
	case "PING":
		return commands.Ping()
	case "ECHO":
		return commands.Echo(command.Args)
	case "SET":
		return commands.Set(command.Args)
	case "GET":
		return commands.Get(command.Args)
	case "RPUSH":
		return commands.Rpush(command.Args)
	case "LRANGE":
		return commands.Lrange(command.Args)
	case "LPUSH":
		return commands.Lpush(command.Args)
	case "LLEN":
		return commands.Llen(command.Args)
	case "LPOP":
		return commands.Lpop(command.Args)
	case "BLPOP":
		return commands.Blpop(command.Args)
	case "TYPE":
		return commands.Type(command.Args)
	case "XADD":
		return commands.XAdd(command.Args)
	default:
		return nil, fmt.Errorf("unkown command '%v'", command.Name)
	}
}
