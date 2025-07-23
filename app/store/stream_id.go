package store

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type StreamId struct {
	Ms               int64
	GenerateMs       bool
	Sequence         int64
	GenerateSequence bool
}

func (id StreamId) String() string {
	return fmt.Sprintf("%v-%v", id.Ms, id.Sequence)
}

func ParseStreamId(id string) (StreamId, error) {
	const idSeparator string = "-"

	//TODO: handle cases including '*'

	splitId := strings.Split(id, idSeparator)
	if len(splitId) != 2 {
		return StreamId{}, errors.New("id couldn't be parsed")
	}

	msPart, err := strconv.ParseInt(splitId[0], 10, 64)
	if err != nil {
		return StreamId{}, fmt.Errorf("error parsing millisecond part: %w", err)
	}

	if splitId[1] == "*" {
		return StreamId{msPart, false, -1, true}, nil
	}

	sequencePart, err := strconv.ParseInt(splitId[1], 10, 64)
	if err != nil {
		return StreamId{}, fmt.Errorf("error parsing sequence part: %w", err)
	}

	return StreamId{msPart, false, sequencePart, false}, nil
}
