package store

import "time"

type StoredValueType int

const (
	TypeString StoredValueType = iota
	TypeList
	TypeStream
)

type StoredValue struct {
	Val       string
	Lval      []string
	Xval      []StreamEntry
	Type      StoredValueType
	ExpiresBy int64
	Listeners []chan string
}

func (sv *StoredValue) AddChannel(c chan string) {
	if sv.Listeners == nil {
		sv.Listeners = []chan string{c}
	} else {
		sv.Listeners = append(sv.Listeners, c)
	}
}

func (sv *StoredValue) IsExpired() bool {
	return sv.ExpiresBy != -1 && time.Now().UnixMilli() > sv.ExpiresBy
}

func NewStringValue(val string, expiresBy int64) StoredValue {
	return StoredValue{val, nil, nil, TypeString, expiresBy, nil}
}

func NewListValue(lval []string) StoredValue {
	return StoredValue{"", lval, nil, TypeList, -1, nil}
}

func NewListListener(c chan string) StoredValue {
	return StoredValue{"", []string{}, nil, TypeList, -1, []chan string{c}}
}

func NewStreamValue(xval []StreamEntry) StoredValue {
	return StoredValue{"", nil, xval, TypeStream, -1, nil}
}
