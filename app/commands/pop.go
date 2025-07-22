package commands

import (
	"errors"
	"fmt"
	"redis-clone-go/app/protocol"
	"redis-clone-go/app/store"
	"slices"
	"strconv"
	"time"
)

func Lpop(args []string) ([]byte, error) {
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

	_, err := store.CM.Update(
		args[0],
		func(storedValue *store.StoredValue) error {
			if !storedValue.IsList {
				return errWrongtypeOperation
			}

			if count > len(storedValue.Lval) {
				count = len(storedValue.Lval)
			}

			result = storedValue.Lval[0:count]
			storedValue.Lval = storedValue.Lval[count:]
			return nil
		})

	if err != nil {
		if errors.Is(err, store.ErrKeyNotFound) {
			return protocol.FormatNullBulkString(), nil
		}

		return nil, err
	}

	if len(args) == 1 {
		return protocol.FormatBulkString(result[0]), nil
	}

	return protocol.FormatBulkStringArray(result), nil
}

func Blpop(args []string) ([]byte, error) {
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

	_, err = store.CM.SetOrUpdate(
		args[0],
		func() store.StoredValue {
			return store.NewListListener(c)
		},
		func(storedValue *store.StoredValue) error {
			if !storedValue.IsList {
				return errWrongtypeOperation
			}

			if len(storedValue.Lval) > 0 {
				result = storedValue.Lval[0]
				storedValue.Lval = storedValue.Lval[1:]
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
		return protocol.FormatBulkStringArray([]string{args[0], result}), nil
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

		return protocol.FormatBulkStringArray([]string{args[0], result}), nil

	case <-timeoutChannel:
		err = removeChannel(args[0], c)
		if err != nil {
			return nil, fmt.Errorf("error removing channel: %w", err)
		}

		return protocol.FormatNullBulkString(), nil
	}
}

func removeChannel(key string, c chan string) error {
	_, err := store.CM.Update(
		key,
		func(storedValue *store.StoredValue) error {
			if !storedValue.IsList {
				return errWrongtypeOperation
			}

			storedValue.Listeners = slices.DeleteFunc(storedValue.Listeners, func(channel chan string) bool {
				return channel == c
			})

			return nil
		},
	)

	if err != nil {
		if errors.Is(err, store.ErrKeyNotFound) {
			return errors.New("error getting list for key")
		}

		return err
	}

	return nil
}
