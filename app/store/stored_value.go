package store

type StoredValue struct {
	Val       string
	Lval      []string
	IsList    bool
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

func NewStringValue(val string, expiresBy int64) StoredValue {
	return StoredValue{val, nil, false, expiresBy, nil}
}

func NewListValue(lval []string) StoredValue {
	return StoredValue{"", lval, true, -1, nil}
}

func NewListListener(c chan string) StoredValue {
	return StoredValue{"", []string{}, true, -1, []chan string{c}}
}
