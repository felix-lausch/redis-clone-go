package main

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

func parseResp(reader *bufio.Reader) (*Command, error) {
	const arrayIndicator byte = '*'

	//TODO: figure out better way to handle error wrapping
	arrLen, err := parseTypeInfo(reader, arrayIndicator)
	if err == io.EOF {
		return nil, err
	} else if err != nil {
		return nil, errors.New("error parsing type information")
	}

	command, err := parseBulkStringArray(reader, arrLen)
	if err != nil {
		return nil, fmt.Errorf("error parsing bulk string array: %w", err)
	}

	return command, nil
}

func parseTypeInfo(reader *bufio.Reader, expectedTypeIndicator byte) (int, error) {
	typeIndicator, err := reader.ReadByte()
	if err == io.EOF {
		return 0, err
	} else if err != nil {
		return 0, errors.New("error reading first byte")
	}

	if typeIndicator != expectedTypeIndicator {
		return 0, errors.New("input is not of type 'Array'")
	}

	lengthStr, err := reader.ReadString('\n')
	if err != nil {
		return 0, errors.New("error reading array length")
	}

	length, err := strconv.Atoi(strings.TrimSuffix(lengthStr, "\r\n"))
	if err != nil {
		return 0, errors.New("array length couldn't be parsed")
	}

	return length, nil
}

func parseBulkStringArray(reader *bufio.Reader, length int) (*Command, error) {
	const bulkStringIndicator byte = '$'

	inputParts := make([]string, 0, length)

	for range length {
		strLength, err := parseTypeInfo(reader, bulkStringIndicator)
		if err != nil {
			return nil, errors.New("error parsing type information")
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
