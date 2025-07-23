package protocol

import "fmt"

func FormatSimpleString(input string) []byte {
	return fmt.Appendf(nil, "+%v\r\n", input)
}

func FormatBulkString(input string) []byte {
	return fmt.Appendf(nil, "$%v\r\n%v\r\n", len(input), input)
}

func FormatNullBulkString() []byte {
	return []byte("$-1\r\n")
}

func FormatInt(num int, signed bool) []byte {
	if signed {
		return fmt.Appendf(nil, ":%+d\r\n", num)
	}

	return fmt.Appendf(nil, ":%d\r\n", num)
}

func FormatBulkStringArray(elements []string) []byte {
	array := fmt.Appendf(nil, "*%v\r\n", len(elements))

	for i := range elements {
		array = append(array, FormatBulkString(elements[i])...)
	}

	return array
}

func FormatError(err error) []byte {
	return fmt.Appendf(nil, "-%v\r\n", err)
}
