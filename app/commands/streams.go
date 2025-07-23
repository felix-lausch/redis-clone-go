package commands

import (
	"errors"
	"fmt"
	"redis-clone-go/app/protocol"
	"redis-clone-go/app/store"
)

func XAdd(args []string) ([]byte, error) {
	if len(args) < 2 || len(args)%2 != 0 {
		return nil, errArgNumber
	}

	streamId, err := store.ParseStreamId(args[1])
	if err != nil {
		return nil, fmt.Errorf("error parsing stream id: %w", err)
	}

	if streamId.Ms == 0 && streamId.Sequence == 0 {
		return nil, errors.New("ERR The ID specified in XADD must be greater than 0-0")
	}

	_, err = store.CM.SetOrUpdate(
		args[0],
		func() store.StoredValue {
			streamId.GenerateValues(nil)
			return store.NewStreamValue([]store.StreamId{streamId})
		},
		func(storedValue *store.StoredValue) error {
			if storedValue.Type != store.TypeStream {
				return errWrongtypeOperation
			}

			streamId.GenerateValues(storedValue.Xval)

			if !streamId.CanAppendKey(storedValue.Xval) {
				return errStreamIdTooSmall
			}

			storedValue.Xval = append(storedValue.Xval, streamId)

			//TODO: store key value pairs to stream

			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	return protocol.FormatBulkString(streamId.String()), nil
}
