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
			streamEntry := store.NewStreamEntry(streamId, args[2:])

			return store.NewStreamValue([]store.StreamEntry{streamEntry})
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

	start, err := ParseXRangeStreamId(args[1])
	if err != nil {
		return nil, fmt.Errorf("error parsing start: %w", err)
	}

	end, err := ParseXRangeStreamId(args[2])
	if err != nil {
		return nil, fmt.Errorf("error parsing end: %w", err)
	}

	storedValue, ok := store.CM.Get(args[0])
	if !ok {
		return protocol.FormatBulkStringArray([]string{}), nil
	}

	if storedValue.Type != store.TypeStream {
		return nil, errWrongtypeOperation
	}

	startIdx, _ := FindIndex(start, storedValue.Xval)
	endIdx, _ := FindIndex(end, storedValue.Xval)

	result := storedValue.Xval[startIdx : endIdx+1]

	return FormatStreamEntries(result), nil
}

func ParseXRangeStreamId(id string) (store.StreamId, error) {
	if id == "-" {
		//minimum id
		return store.StreamId{Ms: 0, Sequence: 1}, nil
	}

	return store.ParseStreamId(id)
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

func FormatStreamEntries(entries []store.StreamEntry) []byte {
	result := fmt.Appendf(nil, "*%v\r\n", len(entries))

	for _, entry := range entries {
		result = append(result, FormatStreamEntry(entry)...)
	}

	return result
}

func FormatStreamEntry(entry store.StreamEntry) []byte {
	array := fmt.Append(nil, "*2\r\n")

	array = append(array, protocol.FormatBulkString(entry.Id.String())...)
	array = append(array, protocol.FormatBulkStringArray(entry.Pairs)...)

	return array
}
