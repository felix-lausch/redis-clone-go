package commands

import "errors"

var errWrongtypeOperation = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")
var errArgNumber = errors.New("wrong number of arguments for command")
