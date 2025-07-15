package main

import (
	"bufio"
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
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		command, args, err := parseResp(reader)
		if err == io.EOF {
			fmt.Println("client disconnected")
			return
		} else if err != nil {
			fmt.Println("Error reading from connection: ", err.Error())
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

func parseResp(reader *bufio.Reader) (command string, args []string, err error) {
	const arrayIndicator byte = '*'

	typeIndicator, err := reader.ReadByte()
	if err == io.EOF {
		return "", nil, err
	} else if err != nil {
		return "", nil, errors.New("error reading first byte")
	}

	if typeIndicator != arrayIndicator {
		return "", nil, errors.New("input is not of type 'Array'")
	}

	arrayLengthStr, err := reader.ReadString('\n')
	if err != nil {
		return "", nil, errors.New("error reading array length")
	}

	arrLen, err := strconv.Atoi(strings.TrimSuffix(arrayLengthStr, "\r\n"))
	if err != nil {
		return "", nil, errors.New("array length couldn't be parsed")
	}

	command, args, err = parseBulkStringArray(reader, arrLen)
	if err != nil {
		return "", nil, fmt.Errorf("error parsing bulk string array: %w", err)
	}

	return command, args, nil
}

func parseBulkStringArray(reader *bufio.Reader, length int) (command string, args []string, err error) {
	const bulkStringIndicator byte = '$'

	inputParts := make([]string, 0, length)

	for range length {
		typeIndicator, err := reader.ReadByte()
		if err != nil {
			return "", nil, errors.New("error reading type byte")
		}

		if typeIndicator != bulkStringIndicator {
			return "", nil, errors.New("input is not of type 'Bulk string'")
		}

		stringLengthStr, err := reader.ReadString('\n')
		if err != nil {
			return "", nil, errors.New("error reading bulk string length")
		}

		strLength, err := strconv.Atoi(strings.TrimSuffix(stringLengthStr, "\r\n"))
		if err != nil {
			return "", nil, fmt.Errorf("bulk string length parse: %w", err)
		}

		stringBuffer := make([]byte, strLength)
		_, err = io.ReadFull(reader, stringBuffer)
		if err != nil {
			return "", nil, fmt.Errorf("error reading bulk string: %w", err)
		}

		inputParts = append(inputParts, string(stringBuffer))

		if _, err := reader.Discard(2); err != nil {
			return "", nil, fmt.Errorf("failed to discard CRLF: %w", err)
		}
	}

	if len(inputParts) == 0 {
		return "", nil, errors.New("no command found")
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
