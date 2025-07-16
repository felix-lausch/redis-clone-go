package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ping() ([]byte, error) {
	return formatSimpleString("PONG"), nil
}

func echo(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("wrong number of arguments for command")
	}

	return formatBulkString(args[0]), nil
}

func set(args []string) ([]byte, error) {
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

	cm.Set(args[0], StoredValue{args[1], expiresBy})
	return formatSimpleString("OK"), nil
}

func get(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("wrong number of arguments for command")
	}

	storedValue, ok := cm.Get(args[0])
	if !ok {
		return formatNullBulkString(), nil
	} else if storedValue.expiresBy != -1 && time.Now().UnixMilli() > storedValue.expiresBy {
		fmt.Println("tried to access expired value")
		cm.Delete(args[0])

		return formatNullBulkString(), nil
	}

	return formatBulkString(storedValue.val), nil
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

func formatError(err error) []byte {
	return fmt.Appendf(nil, "-ERROR %v\r\n", err)
}
