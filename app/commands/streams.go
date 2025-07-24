package commands

import (
	"errors"
	"fmt"
	"math"
	"redis-clone-go/app/protocol"
	"redis-clone-go/app/store"
	"strings"
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

	start, err := parseXRangeStartId(args[1])
	if err != nil {
		return nil, fmt.Errorf("error parsing start: %w", err)
	}

	end, err := parseXRangeEndId(args[2])
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

	startIdx, _ := findIndex(start, storedValue.Xval, true)
	endIdx, _ := findIndex(end, storedValue.Xval, false)

	result := storedValue.Xval[startIdx : endIdx+1]

	return FormatStreamEntries(result), nil
}

func XRead(args []string) ([]byte, error) {
	if len(args) < 3 || len(args)%2 != 1 {
		return nil, errArgNumber
	}

	if strings.ToLower(args[0]) != "streams" {
		return nil, errors.New("you didnt write streams :(")
	}

	results := map[string][]store.StreamEntry{}

	//TODO: what happens if i have multiple same keys?
	for i := 1; i < len(args); i += 2 {
		id, err := store.ParseStreamId(args[i+1])
		if err != nil {
			return nil, fmt.Errorf("error parsing stream id: %w", err)
		}

		storedValue, ok := store.CM.Get(args[i])
		if !ok {
			return nil, errors.New("blocking not implemented")
		}

		if storedValue.Type != store.TypeStream {
			return nil, errWrongtypeOperation
		}

		result := getXReadResult(id, storedValue.Xval)
		results[args[i]] = result
	}

	return FormatXReadResponse(results), nil
}

func parseXRangeStartId(id string) (store.StreamId, error) {
	if id == "-" {
		//minimum id
		return store.StreamId{Ms: 0, Sequence: 1}, nil
	}

	return store.ParseStreamId(id)
}

func parseXRangeEndId(id string) (store.StreamId, error) {
	if id == "+" {
		//maximum id
		return store.StreamId{Ms: math.MaxInt64, Sequence: math.MaxInt64}, nil
	}

	return store.ParseStreamId(id)
}

func findIndex(id store.StreamId, entries []store.StreamEntry, start bool) (int, bool) {
	for i, val := range entries {
		//TODO: this is too simple, it doesnt handle * yet

		if id.Ms == val.Id.Ms && id.Sequence == val.Id.Sequence {
			return i, true
		}
	}

	if start {
		return 0, true
	}

	return len(entries) - 1, true
}

func getXReadResult(id store.StreamId, entries []store.StreamEntry) []store.StreamEntry {
	idx := 0

	for i, entry := range entries {
		if entry.Id.IsEqualTo(id) {
			idx = i + 1
			break
		} else if entry.Id.IsGreaterThan(id) {
			idx = i
			break
		}
	}

	return entries[idx:]
}

// TODO: should these sit here or inside of the protocols package?
func FormatStreamEntries(entries []store.StreamEntry) []byte {
	result := fmt.Appendf(nil, "*%v\r\n", len(entries))

	for _, entry := range entries {
		result = append(result, FormatStreamEntry(entry)...)
	}

	return result
}

func FormatXReadResponse(response map[string][]store.StreamEntry) []byte {
	result := fmt.Appendf(nil, "*%v\r\n", len(response))

	for key, entries := range response {
		result = append(result, []byte("*2\r\n")...)
		result = append(result, protocol.FormatBulkString(key)...)
		result = append(result, FormatStreamEntries(entries)...)
	}

	return result
}

func FormatStreamEntry(entry store.StreamEntry) []byte {
	result := fmt.Append(nil, "*2\r\n")

	result = append(result, protocol.FormatBulkString(entry.Id.String())...)
	result = append(result, protocol.FormatBulkStringArray(entry.Pairs)...)

	return result
}
