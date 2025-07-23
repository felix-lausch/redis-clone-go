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
			entry := store.EmptyStreamEntry(streamId)

			return store.NewStreamValue([]store.StreamEntry{entry})
		},
		func(storedValue *store.StoredValue) error {
			if storedValue.Type != store.TypeStream {
				return errWrongtypeOperation
			}

			streamId.GenerateValues(storedValue.Xval)

			if !streamId.CanAppendKey(storedValue.Xval) {
				return errStreamIdTooSmall
			}

			streamEntry := store.NewStreamEntry(streamId, args[2:])
			storedValue.Xval = append(storedValue.Xval, streamEntry)

			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	return protocol.FormatBulkString(streamId.String()), nil
}

func XRange(args []string) ([]byte, error) {
	if len(args) != 3 {
		return nil, errArgNumber
	}

	start, err := store.ParseStreamId(args[1])
	if err != nil {
		return nil, fmt.Errorf("error parsing start: %w", err)
		// return nil, errors.New("Invalid stream ID specified as stream command argument")
	}

	end, err := store.ParseStreamId(args[2])
	if err != nil {
		return nil, fmt.Errorf("error parsing end: %w", err)
		// return nil, errors.New("Invalid stream ID specified as stream command argument")
	}

	storedValue, ok := store.CM.Get(args[0])
	if !ok {
		return protocol.FormatBulkStringArray([]string{}), nil
	}

	if storedValue.Type != store.TypeStream {
		return nil, errWrongtypeOperation
	}

	//read the correct range from xval
	startIdx, _ := FindIndex(start, storedValue.Xval)
	endIdx, _ := FindIndex(end, storedValue.Xval)

	result := storedValue.Xval[startIdx : endIdx+1]

	//format response

	resultStrings := []string{}

	//TODO: improve this to not have so much byte<->string back and forth
	for _, val := range result {
		pairs := protocol.FormatBulkStringArray(val.Pairs)
		entry := protocol.FormatBulkStringArray([]string{val.Id.String(), string(pairs)})

		resultStrings = append(resultStrings, string(entry))
	}

	return protocol.FormatBulkStringArray(resultStrings), nil
}

func FindIndex(id store.StreamId, entries []store.StreamEntry) (int, bool) {
	for i, val := range entries {
		//TODO: this is too simple, it doesnt handle * yet

		if id.Ms == val.Id.Ms && id.Sequence == val.Id.Sequence {
			return i, true
		}
	}

	//TODO: how to handle not finding anything?
	return -1, false
}
