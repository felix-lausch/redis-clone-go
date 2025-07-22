package commands

import (
	"errors"
	"fmt"
	"redis-clone-go/app/protocol"
	"redis-clone-go/app/store"
	"strconv"
	"strings"
	"time"
)

func Set(args []string) ([]byte, error) {
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

	store.CM.Set(args[0], store.NewStringValue(args[1], expiresBy))

	return protocol.FormatSimpleString("OK"), nil
}

func Get(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errArgNumber
	}

	storedValue, ok := store.CM.Get(args[0])
	if !ok {
		return protocol.FormatNullBulkString(), nil
	}

	if storedValue.Type != store.TypeString {
		return nil, errWrongtypeOperation
	}

	if storedValue.ExpiresBy != -1 && time.Now().UnixMilli() > storedValue.ExpiresBy {
		fmt.Println("tried to access expired value")
		store.CM.Delete(args[0])

		return protocol.FormatNullBulkString(), nil
	}

	return protocol.FormatBulkString(storedValue.Val), nil
}

func Type(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errArgNumber
	}

	storedValue, ok := store.CM.Get(args[0])
	if !ok {
		return protocol.FormatSimpleString("none"), nil
	}

	if storedValue.IsExpired() {
		store.CM.Delete(args[0])
		return protocol.FormatSimpleString("none"), nil
	}

	switch storedValue.Type {
	case store.TypeList:
		return protocol.FormatSimpleString("list"), nil
	case store.TypeStream:
		return protocol.FormatSimpleString("stream"), nil
	case store.TypeString:
		return protocol.FormatSimpleString("string"), nil
	default:
		return protocol.FormatSimpleString("none"), nil
	}
}
