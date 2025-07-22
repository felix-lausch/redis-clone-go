package protocol

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Command struct {
	Name string
	Args []string
}

func ParseCommand(reader *bufio.Reader) (*Command, error) {
	const arrayIndicator byte = '*'

	arrLen, err := parseTypeInfo(reader, arrayIndicator)
	if err != nil {
		return nil, errorParseTypeInfo(err)
	}

	command, err := parseBulkStringArray(reader, arrLen)
	if err != nil {
		return nil, fmt.Errorf("error parsing bulk string array: %w", err)
	}

	return command, nil
}

func parseTypeInfo(reader *bufio.Reader, expectedTypeIndicator byte) (int, error) {
	typeIndicator, err := reader.ReadByte()
	if err != nil {
		return 0, fmt.Errorf("error reading type header: %w", err)
	}

	if typeIndicator != expectedTypeIndicator {
		return 0, fmt.Errorf("input is not of type %q", expectedTypeIndicator)
	}

	lengthStr, err := reader.ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("error reading type length: %w", err)
	}

	length, err := strconv.Atoi(strings.TrimSuffix(lengthStr, "\r\n"))
	if err != nil {
		return 0, fmt.Errorf("type length couldn't be parsed: %w", err)
	}

	return length, nil
}

func parseBulkStringArray(reader *bufio.Reader, length int) (*Command, error) {
	const bulkStringIndicator byte = '$'

	if length <= 0 {
		return nil, fmt.Errorf("invalid length: %d", length)
	}

	inputParts := make([]string, 0, length)

	for range length {
		strLength, err := parseTypeInfo(reader, bulkStringIndicator)
		if err != nil {
			return nil, errorParseTypeInfo(err)
		}

		stringBuffer := make([]byte, strLength)
		_, err = io.ReadFull(reader, stringBuffer)
		if err != nil {
			return nil, fmt.Errorf("error reading bulk string: %w", err)
		}

		inputParts = append(inputParts, string(stringBuffer))

		if _, err := reader.Discard(2); err != nil {
			return nil, fmt.Errorf("failed to discard CRLF: %w", err)
		}
	}

	if len(inputParts) == 0 {
		return nil, errors.New("no command found")
	}

	return &Command{
		strings.ToUpper(inputParts[0]),
		inputParts[1:]}, nil
}

func errorParseTypeInfo(err error) error {
	return fmt.Errorf("error parsing type information: %w", err)
}
