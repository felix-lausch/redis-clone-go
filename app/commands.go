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

func lpush(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, errArgNumber
	}

	storedValue, ok := cm.Get(args[0])
	if !ok {
		values := reverseArray(args[1:])

		cm.Set(args[0], StoredValue{"", values, true, -1})
		return formatInt(len(values), false), nil
	}

	if !storedValue.isList {
		return nil, errWrongtypeOperation
	}

	storedValue.lval = append(reverseArray(args[1:]), storedValue.lval...)
	cm.Set(args[0], storedValue)

	return formatInt(len(storedValue.lval), false), nil
}

func reverseArray(array []string) []string {
	for i, j := 0, len(array)-1; i < j; i, j = i+1, j-1 {
		array[i], array[j] = array[j], array[i]
	}

	return array
}

func lrange(args []string) ([]byte, error) {
	if len(args) != 3 {
		return nil, errArgNumber
	}

	start, err := strconv.Atoi(args[1])
	if err != nil {
		return nil, errors.New("lrange start couldn't be parsed")
	}

	stop, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, errors.New("lrange stop couldn't be parsed")
	}

	storedValue, ok := cm.Get(args[0])
	if !ok {
		return formatBulkStringArray([]string{}), nil
	}

	if !storedValue.isList {
		return nil, errWrongtypeOperation
	}

	lRangeSlice := getLRangeSlice(start, stop, storedValue.lval)
	return formatBulkStringArray(lRangeSlice), nil
}

func llen(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errArgNumber
	}

	storedValue, ok := cm.Get(args[0])
	if !ok {
		return formatInt(0, false), nil
	}

	if !storedValue.isList {
		return nil, errWrongtypeOperation
	}

	return formatInt(len(storedValue.lval), false), nil
}

func lpop(args []string) ([]byte, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, errArgNumber
	}

	count := 1

	if len(args) == 2 {
		argCount, err := strconv.Atoi(args[1])
		if err != nil {
			return nil, errors.New("count could not be parsed")
		}

		count = argCount
	}

	storedValue, ok := cm.Get(args[0])
	if !ok || len(storedValue.lval) == 0 {
		return formatNullBulkString(), nil
	}

	if !storedValue.isList {
		return nil, errWrongtypeOperation
	}

	if count > len(storedValue.lval) {
		count = len(storedValue.lval)
	}

	result := storedValue.lval[0:count]
	storedValue.lval = storedValue.lval[count:]
	cm.Set(args[0], storedValue)

	if len(args) == 1 {
		return formatBulkString(result[0]), nil
	}

	return formatBulkStringArray(result), nil
}

func getLRangeSlice(start, stop int, array []string) []string {
	//translate negative indices to positive ones
	if start < 0 {
		start = max(0, start+len(array))
	}

	if stop < 0 {
		stop = max(0, stop+len(array))
	}

	//return early if start is nonsensical
	if start > len(array) || start > stop {
		return []string{}
	}

	//ensure upper bounds
	start = min(len(array), start)
	stop = min(len(array)-1, stop)

	return array[start : stop+1]
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
