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
			handleStreamListeners(storedValue, streamEntry)

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
	results := []xReadResult{}
	listeners := make([]store.StreamListener, numKeys)

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
				listener := store.StreamListener{C: c, Id: id}
				listeners[i] = listener

				//TODO: this set operation is a concurrency issue
				store.CM.Set(key, store.NewStreamListener(listener))
			}

			continue
		}

		if storedValue.Type != store.TypeStream {
			return nil, errWrongtypeOperation
		}

		result := getxReadResult(key, id, storedValue.Xval)

		if len(result.entries) > 0 {
			results = append(results, result)
		} else if blocking {
			c := make(chan store.StreamEntry, 1)
			listener := store.StreamListener{C: c, Id: id}
			listeners[i] = listener

			storedValue.AddStreamListener(listener)
			//TODO: this set operation is a concurrency issue
			store.CM.Set(key, storedValue)
		}
	}

	if len(results) > 0 {
		return FormatXReadResponse(results), nil
	} else if !blocking {
		return protocol.FormatNullBulkString(), nil
	}

	var timeoutChannel <-chan time.Time
	if timeoutMs > 0 {
		timeoutChannel = time.After(time.Duration(timeoutMs * int64(time.Millisecond)))
		fmt.Print(timeoutChannel) //TODO: remove this
	}

	select {
	//TODO: need to listen on channel that would return key+entries
	case result, ok := <-listeners[0].C:
		if !ok {
			return nil, errors.New("error receiving value from stream")
		}

		return FormatXReadResponse([]xReadResult{{args[3], []store.StreamEntry{result}}}), nil

	case <-timeoutChannel:
		//TODO: remove all listeners -> listeners should contain key
		err = removeStreamListener(args[3], listeners[0].Id)
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

func getxReadResult(key string, id store.StreamId, entries []store.StreamEntry) xReadResult {
	for i, entry := range entries {
		if entry.Id.IsEqualTo(id) {
			return xReadResult{key, entries[i+1:]}
		} else if entry.Id.IsGreaterThan(id) {
			return xReadResult{key, entries[i:]}
		}
	}

	return xReadResult{key, entries[:0]}
}

// TODO: should these sit here or inside of the protocols package?
func FormatXReadResponse(results []xReadResult) []byte {
	response := fmt.Appendf(nil, "*%v\r\n", len(results))

	for _, result := range results {
		response = append(response, []byte("*2\r\n")...)
		response = append(response, protocol.FormatBulkString(result.key)...)
		response = append(response, FormatStreamEntries(result.entries)...)
	}

	return response
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

func removeStreamListener(key string, id store.StreamId) error {
	_, err := store.CM.Update(
		key,
		func(sv *store.StoredValue) error {
			if sv.Type != store.TypeStream {
				return errWrongtypeOperation
			}

			sv.StreamListeners = slices.DeleteFunc(sv.StreamListeners, func(l store.StreamListener) bool {
				return l.Id.IsEqualTo(id)
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

func handleStreamListeners(storedValue *store.StoredValue, latestEntry store.StreamEntry) {
	if len(storedValue.StreamListeners) == 0 || len(storedValue.Xval) == 0 {
		return
	}

	remainingListeners := make([]store.StreamListener, 0, len(storedValue.StreamListeners))

	//TODO: the key needs to be published in the channel as well
	for _, listener := range storedValue.StreamListeners {
		if latestEntry.Id.IsGreaterThan(listener.Id) {
			listener.C <- latestEntry
			close(listener.C)
		} else {
			remainingListeners = append(remainingListeners, listener)
		}
	}

	storedValue.StreamListeners = remainingListeners
}

type xReadResult struct {
	key     string
	entries []store.StreamEntry
}
