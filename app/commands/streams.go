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
			if streamId.GenerateSequence {
				if streamId.Ms == 0 {
					streamId.Sequence = 1
				} else {
					streamId.Sequence = 0
				}

				streamId.GenerateSequence = false
			}

			return store.NewStreamValue([]store.StreamId{streamId})
		},
		func(storedValue *store.StoredValue) error {
			if storedValue.Type != store.TypeStream {
				return errWrongtypeOperation
			}

			if !CanAppendKey(storedValue.Xval, &streamId) {
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

// parse -> (newId.generate(latest)) -> insert

// TODO: this method is mixing the insert and generation steps
func CanAppendKey(streamIds []store.StreamId, newId *store.StreamId) bool {
	if len(streamIds) <= 0 {
		if newId.GenerateSequence {
			if newId.Ms == 0 {
				newId.Sequence = 1
			} else {
				newId.Sequence = 0
			}

			newId.GenerateSequence = false
		}

		return true
	}

	latest := streamIds[len(streamIds)-1]

	if newId.Ms < latest.Ms {
		return false
	}

	if newId.GenerateSequence {
		if latest.Ms == newId.Ms {
			newId.Sequence = latest.Sequence + 1
		} else {
			newId.Sequence = 0
		}

		newId.GenerateSequence = false
	}

	if newId.Ms == latest.Ms && newId.Sequence <= latest.Sequence {
		return false
	}

	return true
}
