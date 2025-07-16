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
	"time"
)

var kvStore = make(map[string]StoredValue)

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

		response, err := handleCommand(command, args)
		if err != nil {
			conn.Write(fmt.Appendf(nil, "-ERROR %v\r\n", err))
		}

		conn.Write(response)
	}
}

// TODO: parsing logic should be moved to its own file and could most likely be refactored
func parseResp(reader *bufio.Reader) (command string, args []string, err error) {
	const arrayIndicator byte = '*'

	arrLen, err := parseTypeInfo(reader, arrayIndicator)
	if err == io.EOF {
		return "", nil, err
	} else if err != nil {
		return "", nil, errors.New("error parsing type information")
	}

	command, args, err = parseBulkStringArray(reader, arrLen)
	if err != nil {
		return "", nil, fmt.Errorf("error parsing bulk string array: %w", err)
	}

	return command, args, nil
}

func parseTypeInfo(reader *bufio.Reader, expectedTypeIndicator byte) (int, error) {
	typeIndicator, err := reader.ReadByte()
	if err == io.EOF {
		return 0, err
	} else if err != nil {
		return 0, errors.New("error reading first byte")
	}

	if typeIndicator != expectedTypeIndicator {
		return 0, errors.New("input is not of type 'Array'")
	}

	lengthStr, err := reader.ReadString('\n')
	if err != nil {
		return 0, errors.New("error reading array length")
	}

	length, err := strconv.Atoi(strings.TrimSuffix(lengthStr, "\r\n"))
	if err != nil {
		return 0, errors.New("array length couldn't be parsed")
	}

	return length, nil
}

func parseBulkStringArray(reader *bufio.Reader, length int) (command string, args []string, err error) {
	const bulkStringIndicator byte = '$'

	inputParts := make([]string, 0, length)

	for range length {
		strLength, err := parseTypeInfo(reader, bulkStringIndicator)
		if err != nil {
			return "", nil, errors.New("error parsing type information")
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

// TODO: this method needs to be refactored. the command execution should be a seperate function every time.
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
		//TODO: for set and get i need to handle race conditions and concurrency
	case "SET":
		{
			if len(args) < 2 {
				return nil, errors.New("wrong number of arguments for command")
			}

			expiresBy := int64(-1)
			if len(args) == 4 {
				if strings.ToUpper(args[2]) != "PX" {
					return nil, fmt.Errorf("unknown argument: %v", args[2])
				}

				ms, err := strconv.ParseInt(args[3], 10, 64)
				if err != nil {
					return nil, errors.New("expire time couldn't be parsed")
				}

				expiresBy = time.Now().UnixMilli() + ms
			}

			kvStore[args[0]] = *NewStoredValue(args[1], expiresBy)
			return formatSimpleString("OK"), nil
		}
	case "GET":
		{
			if len(args) != 1 {
				return nil, errors.New("wrong number of arguments for command")
			}

			storedValue, ok := kvStore[args[0]]
			if !ok {
				return formatNullBulkString(), nil
			} else if storedValue.expiresBy != -1 && time.Now().UnixMilli() > storedValue.expiresBy {
				fmt.Println("tried to access expired value")
				delete(kvStore, args[0])

				return formatNullBulkString(), nil
			}

			return formatBulkString(storedValue.val), nil
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

type StoredValue struct {
	val       string
	expiresBy int64
}

func NewStoredValue(v string, exp int64) *StoredValue {
	return &StoredValue{val: v, expiresBy: exp}
}
