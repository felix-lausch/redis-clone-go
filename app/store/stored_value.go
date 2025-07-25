package store

import "time"

type StoredValueType int

const (
	TypeString StoredValueType = iota
	TypeList
	TypeStream
)

type StoredValue struct {
	Val             string
	Lval            []string
	Xval            []StreamEntry
	Type            StoredValueType
	ExpiresBy       int64
	ListListeners   []chan string
	StreamListeners []StreamListener
}

func (sv *StoredValue) AddListListener(c chan string) {
	if sv.ListListeners == nil {
		sv.ListListeners = []chan string{c}
	} else {
		sv.ListListeners = append(sv.ListListeners, c)
	}
}

func (sv *StoredValue) AddStreamListener(listener StreamListener) {
	if sv.StreamListeners == nil {
		sv.StreamListeners = []StreamListener{listener}
	} else {
		sv.StreamListeners = append(sv.StreamListeners, listener)
	}
}

func (sv *StoredValue) IsExpired() bool {
	return sv.ExpiresBy != -1 && time.Now().UnixMilli() > sv.ExpiresBy
}

func NewStringValue(val string, expiresBy int64) StoredValue {
	return StoredValue{val, nil, nil, TypeString, expiresBy, nil, nil}
}

func NewListValue(lval []string) StoredValue {
	return StoredValue{"", lval, nil, TypeList, -1, nil, nil}
}

func NewListListener(c chan string) StoredValue {
	return StoredValue{"", []string{}, nil, TypeList, -1, []chan string{c}, nil}
}

func NewStreamValue(xval []StreamEntry) StoredValue {
	return StoredValue{"", nil, xval, TypeStream, -1, nil, nil}
}

// TODO: find better name?
func NewStreamListener(listener StreamListener) StoredValue {
	return StoredValue{"", nil, []StreamEntry{}, TypeStream, -1, nil, []StreamListener{listener}}
}

type StreamListener struct {
	C   chan StreamEntry
	Id  StreamId
	Key string
}
