package commands

import (
	"redis-clone-go/app/protocol"
	"redis-clone-go/app/store"
)

func XAdd(args []string) ([]byte, error) {
	if len(args) < 2 || len(args)%2 != 0 {
		return nil, errArgNumber
	}

	//TODO: check id, generate id

	storedValue, ok := store.CM.Get(args[0])
	if !ok {
		store.CM.Set(args[0], store.NewStreamValue(map[string]string{args[1]: ""}))
		return protocol.FormatBulkString(args[1]), nil
	}

	if storedValue.Type != store.TypeStream {
		return nil, errWrongtypeOperation
	}

	//TODO: store key value pairs to stream

	return protocol.FormatBulkString(args[1]), nil
}
