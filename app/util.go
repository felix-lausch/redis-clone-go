package main

import "fmt"

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func reverseArray(array []string) []string {
	for i, j := 0, len(array)-1; i < j; i, j = i+1, j-1 {
		array[i], array[j] = array[j], array[i]
	}

	return array
}

func formatSimpleString(input string) []byte {
	return fmt.Appendf(nil, "+%v\r\n", input)
}

func formatBulkString(input string) []byte {
	return fmt.Appendf(nil, "$%v\r\n%v\r\n", len(input), input)
}

func formatNullBulkString() []byte {
	return []byte("$-1\r\n")
}

func formatInt(num int, signed bool) []byte {
	if signed {
		return fmt.Appendf(nil, ":%+d\r\n", num)
	}

	return fmt.Appendf(nil, ":%d\r\n", num)
}

func formatBulkStringArray(elements []string) []byte {
	array := fmt.Appendf(nil, "*%v\r\n", len(elements))

	for i := range elements {
		array = append(array, formatBulkString(elements[i])...)
	}

	return array
}

func formatError(err error) []byte {
	return fmt.Appendf(nil, "-ERROR %v\r\n", err)
}
