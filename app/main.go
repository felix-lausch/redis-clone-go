package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports in stage 1 (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit
var kvStore = make(map[string]string)

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
	for {
		buffer := make([]byte, 256)

		n, err := conn.Read(buffer)
		if err == io.EOF {
			fmt.Println("client disconnected")
			return
		} else if err != nil {
			fmt.Println("Error reading from connection: ", err.Error())
			continue
		}

		msg := string(buffer[:n])

		//parse command and args
		command, args, err := parseInput(msg)
		if err != nil {
			fmt.Println("Error parsing input: ", err.Error())
			conn.Write([]byte("-ERROR input could not be parsed\r\n"))
			continue
		}

		//handle command
		response, err := handleCommand(command, args)
		if err != nil {
			conn.Write(fmt.Appendf(nil, "-ERROR %v\r\n", err))
		}

		conn.Write(response)
	}
}

func parseInput(input string) (command string, args []string, err error) {
	const arrayChar byte = '*'
	const bulkStringChar byte = '$'

	if input[0] != arrayChar {
		return "", nil, errors.New("input is not of type 'Array'")
	}

	splintInput := strings.Split(input, "\r\n")

	arrLength, err := strconv.Atoi(splintInput[0][1:])
	if err != nil {
		return "", nil, errors.New("array length couldn't be parsed")
	}

	inputParts := make([]string, 0, arrLength)
	for i := 1; i < arrLength*2; i += 2 {
		if splintInput[i][0] != bulkStringChar {
			return "", nil, errors.New("input is not of type 'String'")
		}

		strLength, err := strconv.Atoi(splintInput[i][1:])
		if err != nil {
			return "", nil, fmt.Errorf("string length parse: %w", err)
		}

		str := splintInput[i+1]
		if len(str) != strLength {
			return "", nil, errors.New("string did not have expected length")
		}

		inputParts = append(inputParts, str)
	}

	command = strings.ToUpper(inputParts[0])
	args = inputParts[1:]

	return command, args, nil
}

func handleCommand(command string, args []string) ([]byte, error) {
	switch command {
	case "PING":
		{
			return formatSimpleString("PONG"), nil
		}
	case "ECHO":
		{
			if len(args) != 1 {
				return nil, errors.New("wrong number of arguments for command")
			}

			return formatBulkString(args[0]), nil
		}
	case "SET":
		{
			if len(args) != 2 {
				return nil, errors.New("wrong number of arguments for command")
			}

			kvStore[args[0]] = args[1]
			return formatSimpleString("OK"), nil
		}
	case "GET":
		{
			if len(args) != 1 {
				return nil, errors.New("wrong number of arguments for command")
			}

			v, ok := kvStore[args[0]]
			if !ok {
				return formatNullBulkString(), nil
			}

			return formatBulkString(v), nil
		}
	default:
		{
			return nil, fmt.Errorf("unkown command '%v'", command)
		}
	}
}

func formatSimpleString(input string) []byte {
	return fmt.Appendf(nil, "+%v\r\n", input)
}

func formatBulkString(input string) []byte {
	return fmt.Appendf(nil, "$%v\r\n%v\r\n", len(input), input)
}

func formatNullBulkString() []byte {
	return []byte("$-1\r\n")
}
