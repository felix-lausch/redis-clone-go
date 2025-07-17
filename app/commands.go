package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var errWrongtypeOperation = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
var errArgNumber = errors.New("wrong number of arguments for command")

func ping() ([]byte, error) {
	return formatSimpleString("PONG"), nil
}

func echo(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errArgNumber
	}

	return formatBulkString(args[0]), nil
}

func set(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, errArgNumber
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

	cm.Set(args[0], StoredValue{args[1], nil, false, expiresBy})
	return formatSimpleString("OK"), nil
}

func get(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errArgNumber
	}

	storedValue, ok := cm.Get(args[0])
	if !ok {
		return formatNullBulkString(), nil
	} else if storedValue.isList {
		return nil, errWrongtypeOperation
	} else if storedValue.expiresBy != -1 && time.Now().UnixMilli() > storedValue.expiresBy {
		fmt.Println("tried to access expired value")
		cm.Delete(args[0])

		return formatNullBulkString(), nil
	}

	return formatBulkString(storedValue.val), nil
}

func rpush(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, errArgNumber
	}

	storedValue, ok := cm.Get(args[0])
	if !ok {
		cm.Set(args[0], StoredValue{"", args[1:], true, -1})
		return formatInt(len(args[1:]), false), nil
	}

	if !storedValue.isList {
		return nil, errWrongtypeOperation
	}

	storedValue.lval = append(storedValue.lval, args[1:]...)
	cm.Set(args[0], storedValue)

	return formatInt(len(storedValue.lval), false), nil
}

func lrange(args []string) ([]byte, error) {
	if len(args) != 3 {
		return nil, errArgNumber
	}

	start, err := strconv.Atoi(args[1])
	if err != nil {
		return nil, errors.New("lrange start couldn't be parsed")
	}

	if start < 0 {
		start = 0
	}

	stop, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, errors.New("lrange stop couldn't be parsed")
	}

	storedValue, ok := cm.Get(args[0])
	if !ok || start > stop {
		return formatBulkStringArray([]string{}), nil
	}

	if !storedValue.isList {
		return nil, errWrongtypeOperation
	}

	if start > len(storedValue.lval)-1 {
		return formatBulkStringArray([]string{}), nil
	}

	if stop > (len(storedValue.lval) - 1) {
		stop = len(storedValue.lval) - 1
	}

	return formatBulkStringArray(storedValue.lval[start : stop+1]), nil
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

func formatInt(num int, signed bool) []byte {
	if signed {
		return fmt.Appendf(nil, ":%+d\r\n", num)
	}

	return fmt.Appendf(nil, ":%d\r\n", num)
}

func formatBulkStringArray(elements []string) []byte {
	array := fmt.Appendf(nil, "*%v\r\n", len(elements))

	for i := range elements {
		array = append(array, formatBulkString(elements[i])...)
	}

	return array
}

func formatError(err error) []byte {
	return fmt.Appendf(nil, "-ERROR %v\r\n", err)
}
