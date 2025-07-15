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

		// buffer := make([]byte, 256)

		// n, err := conn.Read(buffer)
		// if err == io.EOF {
		// 	fmt.Println("client disconnected")
		// 	return
		// } else if err != nil {
		// 	fmt.Println("Error reading from connection: ", err.Error())
		// 	continue
		// }

		// msg := string(buffer[:n])

		// //parse command and args
		// command, args, err := parseInput(msg)
		// if err != nil {
		// 	fmt.Println("Error parsing input: ", err.Error())
		// 	conn.Write([]byte("-ERROR input could not be parsed\r\n"))
		// 	continue
		// }

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
		return "", nil, fmt.Errorf("error parsing bulk string array: %w", err.Error())
	}

	return command, args, nil
}

func parseBulkStringArray(reader *bufio.Reader, len int) (command string, args []string, err error) {
	const bulkStringIndicator byte = '$'

	inputParts := make([]string, 0, len)

	for i := 1; i < len*2; i += 2 {
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

		//TODO: this is source of errors to just discard nilly willy
		reader.Discard(2) //discard carriage return
	}

	command = strings.ToUpper(inputParts[0])
	args = inputParts[1:]

	return command, args, nil
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
