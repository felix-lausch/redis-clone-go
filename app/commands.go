package main

import (
	"errors"
	"fmt"
	"slices"
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

	cm.Set(args[0], StoredValue{args[1], nil, false, expiresBy, nil})
	return formatSimpleString("OK"), nil
}

func get(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errArgNumber
	}

	storedValue, ok := cm.Get(args[0])
	if !ok {
		return formatNullBulkString(), nil
	}

	if storedValue.isList {
		return nil, errWrongtypeOperation
	}

	if storedValue.expiresBy != -1 && time.Now().UnixMilli() > storedValue.expiresBy {
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

	returnCodeCraftersError := false
	updatedValue, err := cm.SetOrUpdate(
		args[0],
		func() StoredValue {
			return StoredValue{"", args[1:], true, -1, nil}
		},
		func(storedValue *StoredValue) error {
			if !storedValue.isList {
				return errWrongtypeOperation
			}

			if len(storedValue.listeners) == 0 {
				storedValue.lval = append(storedValue.lval, args[1:]...)
				return nil
			}

			handleListeners(storedValue, args[1:], false)
			returnCodeCraftersError = true
			return nil
		})

	if err != nil {
		return nil, err
	}

	if returnCodeCraftersError {
		return formatInt(1, false), nil
	}

	return formatInt(len(updatedValue.lval), false), nil
}

func lpush(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, errArgNumber
	}

	storedValue, err := cm.SetOrUpdate(
		args[0],
		func() StoredValue {
			values := reverseArray(args[1:])
			return StoredValue{"", values, true, -1, nil}
		},
		func(storedValue *StoredValue) error {
			if !storedValue.isList {
				return errWrongtypeOperation
			}

			if len(storedValue.listeners) == 0 {
				storedValue.lval = append(reverseArray(args[1:]), storedValue.lval...)
				return nil
			}

			valsReversed := reverseArray(args[1:])
			handleListeners(storedValue, valsReversed, true)
			return nil
		})

	if err != nil {
		return nil, err
	}

	return formatInt(len(storedValue.lval), false), nil
}

func handleListeners(storedValue *StoredValue, listValues []string, prepend bool) {
	limit := min(len(storedValue.listeners), len(listValues))

	for i := range limit {
		storedValue.listeners[i] <- listValues[i]
		close(storedValue.listeners[i])
	}

	storedValue.listeners = storedValue.listeners[limit:]

	if prepend {
		storedValue.lval = append(listValues[limit:], storedValue.lval...)
	} else {
		storedValue.lval = append(storedValue.lval, listValues[limit:]...)
	}
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

	result := []string{}

	_, err := cm.Update(
		args[0],
		func(storedValue *StoredValue) error {
			if !storedValue.isList {
				return errWrongtypeOperation
			}

			if count > len(storedValue.lval) {
				count = len(storedValue.lval)
			}

			result = storedValue.lval[0:count]
			storedValue.lval = storedValue.lval[count:]
			return nil
		})

	if err != nil {
		if errors.Is(err, errKeyNotFound) {
			return formatNullBulkString(), nil
		}

		return nil, err
	}

	if len(args) == 1 {
		return formatBulkString(result[0]), nil
	}

	return formatBulkStringArray(result), nil
}

func blpop(args []string) ([]byte, error) {
	//TODO: add multiple list args
	if len(args) != 2 {
		return nil, errArgNumber
	}

	timeout, err := strconv.ParseFloat(args[1], 64)
	if err != nil || timeout < 0 {
		return nil, errors.New("timeout couldn't be parsed")
	}

	result := ""
	c := make(chan string, 1)

	_, err = cm.SetOrUpdate(
		args[0],
		func() StoredValue {
			return StoredValue{"", []string{}, true, -1, []chan string{c}}
		},
		func(storedValue *StoredValue) error {
			if !storedValue.isList {
				return errWrongtypeOperation
			}

			if len(storedValue.lval) > 0 {
				result = storedValue.lval[0]
				storedValue.lval = storedValue.lval[1:]
				return nil
			}

			storedValue.AddChannel(c)
			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	if result != "" {
		return formatBulkStringArray([]string{args[0], result}), nil
	}

	var timeoutChannel <-chan time.Time
	if timeout > 0 {
		timeoutChannel = time.After(time.Duration(timeout * float64(time.Second)))
	}

	select {
	case result, ok := <-c:
		if !ok {
			return nil, errors.New("error receiving value from list")
		}

		return formatBulkStringArray([]string{args[0], result}), nil

	case <-timeoutChannel:
		err = removeChannel(args[0], c)
		if err != nil {
			return nil, fmt.Errorf("error removing channel: %w", err)
		}

		return formatNullBulkString(), nil
	}
}

func removeChannel(key string, c chan string) error {
	_, err := cm.Update(
		key,
		func(storedValue *StoredValue) error {
			if !storedValue.isList {
				return errWrongtypeOperation
			}

			storedValue.listeners = slices.DeleteFunc(storedValue.listeners, func(channel chan string) bool {
				return channel == c
			})

			return nil
		},
	)

	if err != nil {
		if errors.Is(err, errKeyNotFound) {
			return errors.New("error getting list for key")
		}

		return err
	}

	return nil
}
