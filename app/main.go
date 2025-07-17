package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
)

var cm = &ConcurrentMap[StoredValue]{
	db: make(map[string]StoredValue),
}

// TODO: i can probably add an array of listeners inside the StoredValue struct
// then when it is retrieved during an update, the first listener in the array will get the value and be popped
// var listeners = &ConcurrentMap[[]Listener]{
// 	db: make(map[string][]Listener),
// }

type Listener struct {
	c chan string
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
			conn.Write(formatError(err))
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
	case "RPUSH":
		return rpush(command.Args)
	case "LRANGE":
		return lrange(command.Args)
	case "LPUSH":
		return lpush(command.Args)
	case "LLEN":
		return llen(command.Args)
	case "LPOP":
		return lpop(command.Args)
	case "BLPOP":
		return blpop(command.Args)
	default:
		return nil, fmt.Errorf("unkown command '%v'", command.Name)
	}
}
