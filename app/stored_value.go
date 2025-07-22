package main

type StoredValue struct {
	val       string
	lval      []string
	isList    bool
	expiresBy int64
	listeners []chan string
}

func (sv *StoredValue) AddChannel(c chan string) {
	if sv.listeners == nil {
		sv.listeners = []chan string{c}
	} else {
		sv.listeners = append(sv.listeners, c)
	}
}
