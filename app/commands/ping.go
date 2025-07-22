package commands

import "redis-clone-go/app/protocol"

func Ping() ([]byte, error) {
	return protocol.FormatSimpleString("PONG"), nil
}

func Echo(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errArgNumber
	}

	return protocol.FormatBulkString(args[0]), nil
}
