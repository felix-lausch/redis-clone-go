package store

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type StreamId struct {
	Ms               int64
	generateMs       bool
	Sequence         int64
	generateSequence bool
}

func (id StreamId) String() string {
	return fmt.Sprintf("%v-%v", id.Ms, id.Sequence)
}

func (id *StreamId) GenerateValues(previousIds []StreamEntry) {
	var latestId *StreamId

	if len(previousIds) > 0 {
		latestId = &previousIds[len(previousIds)-1].Id
	}

	if id.generateMs {
		id.Ms = time.Now().UnixMilli()
		id.generateMs = false
	}

	if !id.generateSequence {
		return
	}

	latestMs := int64(0)
	latestSequence := int64(0)

	if latestId != nil {
		latestMs = latestId.Ms
		latestSequence = latestId.Sequence
	}

	if id.Ms == latestMs {
		id.Sequence = latestSequence + 1
	} else {
		id.Sequence = 0
	}

	id.generateSequence = false
}

func (id *StreamId) CanAppendKey(previousIds []StreamEntry) bool {
	if len(previousIds) == 0 {
		return true
	}

	latest := previousIds[len(previousIds)-1].Id

	if id.Ms < latest.Ms {
		return false
	}

	if id.Ms == latest.Ms && id.Sequence <= latest.Sequence {
		return false
	}

	return true
}

func ParseStreamId(id string) (StreamId, error) {
	const idSeparator string = "-"
	const asterisk string = "*"

	if id == asterisk {
		return StreamId{-1, true, -1, true}, nil
	}

	splitId := strings.Split(id, idSeparator)
	if len(splitId) != 2 {
		return StreamId{}, errors.New("id couldn't be parsed")
	}

	msPart, err := strconv.ParseInt(splitId[0], 10, 64)
	if err != nil {
		return StreamId{}, fmt.Errorf("error parsing millisecond part: %w", err)
	}

	if splitId[1] == asterisk {
		return StreamId{msPart, false, -1, true}, nil
	}

	sequencePart, err := strconv.ParseInt(splitId[1], 10, 64)
	if err != nil {
		return StreamId{}, fmt.Errorf("error parsing sequence part: %w", err)
	}

	return StreamId{msPart, false, sequencePart, false}, nil
}
