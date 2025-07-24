package commands

import (
	"errors"
	"redis-clone-go/app/protocol"
	"redis-clone-go/app/store"
	"strconv"
)

func Rpush(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, errArgNumber
	}

	returnCodeCraftersError := false
	updatedValue, err := store.CM.SetOrUpdate(
		args[0],
		func() store.StoredValue {
			return store.NewListValue(args[1:])
		},
		func(storedValue *store.StoredValue) error {
			if storedValue.Type != store.TypeList {
				return errWrongtypeOperation
			}

			if len(storedValue.ListListeners) == 0 {
				storedValue.Lval = append(storedValue.Lval, args[1:]...)
				return nil
			}

			handleListListeners(storedValue, args[1:], false)
			returnCodeCraftersError = true
			return nil
		})

	if err != nil {
		return nil, err
	}

	if returnCodeCraftersError {
		return protocol.FormatInt(1, false), nil
	}

	return protocol.FormatInt(len(updatedValue.Lval), false), nil
}

func Lpush(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, errArgNumber
	}

	storedValue, err := store.CM.SetOrUpdate(
		args[0],
		func() store.StoredValue {
			values := reverseArray(args[1:])
			return store.NewListValue(values)
		},
		func(storedValue *store.StoredValue) error {
			if storedValue.Type != store.TypeList {
				return errWrongtypeOperation
			}

			if len(storedValue.ListListeners) == 0 {
				storedValue.Lval = append(reverseArray(args[1:]), storedValue.Lval...)
				return nil
			}

			valsReversed := reverseArray(args[1:])
			handleListListeners(storedValue, valsReversed, true)
			return nil
		})

	if err != nil {
		return nil, err
	}

	return protocol.FormatInt(len(storedValue.Lval), false), nil
}

func handleListListeners(storedValue *store.StoredValue, listValues []string, prepend bool) {
	limit := min(len(storedValue.ListListeners), len(listValues))

	for i := range limit {
		storedValue.ListListeners[i] <- listValues[i]
		close(storedValue.ListListeners[i])
	}

	storedValue.ListListeners = storedValue.ListListeners[limit:]

	if prepend {
		storedValue.Lval = append(listValues[limit:], storedValue.Lval...)
	} else {
		storedValue.Lval = append(storedValue.Lval, listValues[limit:]...)
	}
}

func Lrange(args []string) ([]byte, error) {
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

	storedValue, ok := store.CM.Get(args[0])
	if !ok {
		return protocol.FormatBulkStringArray([]string{}), nil
	}

	if storedValue.Type != store.TypeList {
		return nil, errWrongtypeOperation
	}

	lRangeSlice := getLRangeSlice(start, stop, storedValue.Lval)
	return protocol.FormatBulkStringArray(lRangeSlice), nil
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

func Llen(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errArgNumber
	}

	storedValue, ok := store.CM.Get(args[0])
	if !ok {
		return protocol.FormatInt(0, false), nil
	}

	if storedValue.Type != store.TypeList {
		return nil, errWrongtypeOperation
	}

	return protocol.FormatInt(len(storedValue.Lval), false), nil
}

func reverseArray(array []string) []string {
	for i, j := 0, len(array)-1; i < j; i, j = i+1, j-1 {
		array[i], array[j] = array[j], array[i]
	}

	return array
}
