package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
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

	fmt.Println("closing program..")
}

func handleConnection(conn net.Conn) {
	for {
		reader := bufio.NewReader(conn)

		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from connection: ", err.Error())
			continue
		}

		// msg := make([]byte, 256)

		// n, err := conn.Read(msg)
		// if err != nil {
		// 	fmt.Println("Error reading from connection: ", err.Error())
		// 	continue
		// }

		//parse command and args
		command, args, err := parseInput(msg)
		if err != nil {
			conn.Write([]byte("-ERROR input could not be parsed\r\n"))
			continue
		}

		//handle command
		response, err := handleCommand(command, args)
		if err != nil {
			conn.Write(fmt.Appendf(nil, "-ERROR %v\r\n", err))
		}
		fmt.Println(command)

		conn.Write(response)
	}
}

func parseInput(i string) (command string, args []string, err error) {
	if string(i[0]) != "*" {
		return "", nil, errors.New("input is not of type 'Array'")
	}
	//detect type -> the first byte indicates the type

	// test
	splintInput := strings.Split(strings.TrimSpace(i), " ")

	//parse command
	command = strings.ToUpper(splintInput[0])
	fmt.Println(command)

	//parse args
	args = splintInput[1:]

	return command, args, nil
}

func handleCommand(command string, args []string) ([]byte, error) {
	switch command {
	case "PING":
		{
			return []byte("+PONG\r\n"), nil
		}
	case "ECHO":
		{
			if len(args) < 1 {
				return nil, errors.New("wrong number of arguments for command")
			}

			return fmt.Appendf(nil, "%v \"%v\"\r\n", command, args[0]), nil
		}
	default:
		{
			return nil, fmt.Errorf("unkown command '%v'", command)
		}
	}
}
