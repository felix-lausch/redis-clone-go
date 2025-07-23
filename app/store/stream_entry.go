package store

type StreamEntry struct {
	Id    StreamId
	Pairs []string
}

func EmptyStreamEntry(id StreamId) StreamEntry {
	return StreamEntry{id, []string{}}
}

func NewStreamEntry(id StreamId, pairs []string) StreamEntry {
	return StreamEntry{id, pairs}
}
