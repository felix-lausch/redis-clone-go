package commands

import (
	"errors"
	"fmt"
	"redis-clone-go/app/protocol"
	"redis-clone-go/app/store"
)

// func XAdd(args []string) ([]byte, error) {
// 	if len(args) < 2 || len(args)%2 != 0 {
// 		return nil, errArgNumber
// 	}

// 	//TODO: check if args[0] contains asterisk, if so -> call generate method
// 	streamId, err := store.ParseStreamId(args[1])
// 	if err != nil {
// 		return nil, fmt.Errorf("error parsing stream id: %w", err)
// 	}

// 	if streamId.Ms == 0 && streamId.Sequence == 0 {
// 		return nil, errors.New("ERR The ID specified in XADD must be greater than 0-0")
// 	}

// 	storedValue, ok := store.CM.Get(args[0])
// 	if !ok {
// 		store.CM.Set(args[0], store.NewStreamValue([]store.StreamId{streamId}))
// 		return protocol.FormatBulkString(streamId.String()), nil
// 	}

// 	if storedValue.Type != store.TypeStream {
// 		return nil, errWrongtypeOperation
// 	}

// 	if len(storedValue.Xval) > 0 {
// 		latest := storedValue.Xval[len(storedValue.Xval)-1]

// 		//if stream ms is smaller than latest ms -> err

// 		if streamId.Ms < latest.Ms {
// 			return nil, errors.New("ERR The ID specified in XADD is equal or smaller than the target stream top item")
// 		}

// 		if streamId.Ms == latest.Ms && streamId.Sequence <= latest.Sequence {
// 			return nil, errors.New("ERR The ID specified in XADD is equal or smaller than the target stream top item")
// 		}

// 		// if latest.Ms < streamId.Ms || (latest.Ms == streamId.Ms && latest.Sequence >= streamId.Sequence) {
// 		// 	return nil, errors.New("ERR The ID specified in XADD is equal or smaller than the target stream top item")
// 		// }
// 	}

// 	storedValue.Xval = append(storedValue.Xval, streamId)
// 	//TODO: store key value pairs to stream

// 	return protocol.FormatBulkString(streamId.String()), nil
// }

func XAdd(args []string) ([]byte, error) {
	if len(args) < 2 || len(args)%2 != 0 {
		return nil, errArgNumber
	}

	//TODO: check if args[0] contains asterisk, if so -> call generate method
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
			return store.NewStreamValue([]store.StreamId{streamId})
		},
		func(storedValue *store.StoredValue) error {
			if storedValue.Type != store.TypeStream {
				return errWrongtypeOperation
			}

			if !CanAppendKey(storedValue.Xval, streamId) {
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

// TODO: should this be a method of streamid?
func CanAppendKey(streamIds []store.StreamId, newId store.StreamId) bool {
	if len(streamIds) > 0 {
		latest := streamIds[len(streamIds)-1]

		if newId.Ms < latest.Ms {
			return false
		}

		if newId.Ms == latest.Ms && newId.Sequence <= latest.Sequence {
			return false
		}
	}

	return true
}
