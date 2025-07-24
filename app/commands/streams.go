package commands

import (
	"errors"
	"fmt"
	"math"
	"redis-clone-go/app/protocol"
	"redis-clone-go/app/store"
	"slices"
	"strconv"
	"strings"
	"time"
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
			handleStreamListeners(args[0], storedValue, streamEntry)

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

	var timeoutMs int64
	var err error
	var blocking bool

	if strings.ToUpper(args[0]) == "BLOCK" {
		timeoutMs, err = strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing block timeout: %w", err)
		}

		blocking = true
	}

	streamsIdx := findStreamsIndex(args)
	if streamsIdx < 0 {
		return nil, errors.New("missing streams argument")
	}

	keysAndIds := args[streamsIdx+1:]
	numKeys := len(keysAndIds) / 2
	results := map[string][]store.StreamEntry{}
	listeners := make([]chan store.StreamEntry, numKeys)

	//TODO: what happens if i have duplicate keys?
	for i := range numKeys {
		id, err := store.ParseStreamId(keysAndIds[i+numKeys])
		if err != nil {
			return nil, fmt.Errorf("error parsing stream id: %w", err)
		}

		key := keysAndIds[i]
		storedValue, ok := store.CM.Get(key)
		if !ok {
			if blocking {
				c := make(chan store.StreamEntry, 1)
				listeners[i] = c

				//TODO: this set operation is a concurrency issue
				store.CM.Set(key, store.NewStreamListener(c))
			}

			continue
		}

		if storedValue.Type != store.TypeStream {
			return nil, errWrongtypeOperation
		}

		result := getXReadResult(id, storedValue.Xval)
		//TODO: should only add if result has len > 0 -> but this affects the format function as well
		results[key] = result
	}

	//TODO: this case should only execute when there are actually entries behind the keys. the above todo is related
	if len(results) > 0 {
		return FormatXReadResponse(results, keysAndIds[:numKeys]), nil
	} else if !blocking {
		return protocol.FormatNullBulkString(), nil
	}

	var timeoutChannel <-chan time.Time
	if timeoutMs > 0 {
		timeoutChannel = time.After(time.Duration(timeoutMs * int64(time.Millisecond)))
	}

	select {
	case result, ok := <-listeners[0]:
		if !ok {
			return nil, errors.New("error receiving value from stream")
		}

		//TODO: need to listen on channel that would return key+entries
		return FormatXReadResponse(
			map[string][]store.StreamEntry{"key": {result}},
			[]string{"key"},
		), nil

	case <-timeoutChannel:
		//TODO: read key from listeners array
		err = removeStreamListener(args[3], listeners[0])
		if err != nil {
			return nil, fmt.Errorf("error removing channel: %w", err)
		}

		return protocol.FormatNullBulkString(), nil
	}
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

func findStreamsIndex(array []string) int {
	s := strings.ToUpper("STREAMS")

	for i := range len(array) {
		if s == strings.ToUpper(array[i]) {
			return i
		}
	}

	return -1
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
func FormatXReadResponse(response map[string][]store.StreamEntry, orderedKeys []string) []byte {
	count := 0
	for _, entries := range response {
		if len(entries) > 0 {
			count++
		}
	}

	result := fmt.Appendf(nil, "*%v\r\n", count)

	for _, key := range orderedKeys {
		entries := response[key]

		if len(entries) == 0 {
			continue
		}

		result = append(result, []byte("*2\r\n")...)
		result = append(result, protocol.FormatBulkString(key)...)
		result = append(result, FormatStreamEntries(entries)...)
	}

	return result
}

func FormatStreamEntries(entries []store.StreamEntry) []byte {
	result := fmt.Appendf(nil, "*%v\r\n", len(entries))

	for _, entry := range entries {
		result = append(result, FormatStreamEntry(entry)...)
	}

	return result
}

func FormatStreamEntry(entry store.StreamEntry) []byte {
	result := fmt.Append(nil, "*2\r\n")

	result = append(result, protocol.FormatBulkString(entry.Id.String())...)
	result = append(result, protocol.FormatBulkStringArray(entry.Pairs)...)

	return result
}

func removeStreamListener(key string, c chan store.StreamEntry) error {
	_, err := store.CM.Update(
		key,
		func(storedValue *store.StoredValue) error {
			if storedValue.Type != store.TypeStream {
				return errWrongtypeOperation
			}

			storedValue.StreamListeners = slices.DeleteFunc(storedValue.StreamListeners, func(channel chan store.StreamEntry) bool {
				return channel == c
			})

			return nil
		},
	)

	if err != nil {
		if errors.Is(err, store.ErrKeyNotFound) {
			return errors.New("error getting stream for key")
		}

		return err
	}

	return nil
}

func handleStreamListeners(key string, storedValue *store.StoredValue, latestEntry store.StreamEntry) {
	if len(storedValue.StreamListeners) == 0 || len(storedValue.Xval) == 0 {
		return
	}

	//TODO: remove this
	fmt.Printf("handling stream listeners for: %v\r\n", key)

	//TODO: the key needs to be published in the channel as well
	for i := range len(storedValue.StreamListeners) {
		storedValue.StreamListeners[i] <- latestEntry
		close(storedValue.ListListeners[i])
	}

	storedValue.StreamListeners = storedValue.StreamListeners[:0]
}
