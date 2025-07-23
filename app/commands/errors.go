package commands

import "errors"

var errWrongtypeOperation = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
var errArgNumber = errors.New("ERR wrong number of arguments for command")
var errStreamIdTooSmall = errors.New("ERR The ID specified in XADD is equal or smaller than the target stream top item")
